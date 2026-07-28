// Copyright 2026 The Crater Project Team, RAIDS-Lab
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/raids-lab/crater/cli/internal/api"
	"github.com/raids-lab/crater/cli/internal/clierror"
	"github.com/raids-lab/crater/cli/internal/i18n"
	"github.com/raids-lab/crater/cli/internal/output"
	"github.com/raids-lab/crater/cli/internal/session"
	internalversion "github.com/raids-lab/crater/cli/internal/version"
	"github.com/raids-lab/crater/cli/pkg/errorcodes"
	"github.com/spf13/cobra"
)

type compatibilityCLI struct {
	ProductVersion                string `json:"product_version"`
	APIVersion                    int    `json:"api_version"`
	MinSupportedBackendAPIVersion int    `json:"min_supported_backend_api_version"`
}

type compatibilityBackend struct {
	ProductVersion            string `json:"product_version"`
	ShortCommitSHA            string `json:"short_commit_sha"`
	BuildType                 string `json:"build_type"`
	BuildTime                 string `json:"build_time"`
	APIVersion                int    `json:"api_version"`
	MinSupportedCLIAPIVersion int    `json:"min_supported_cli_api_version"`
	VersionsReported          bool   `json:"-"`
}

type compatibilityResult struct {
	PlatformURL string                              `json:"platform_url"`
	Status      internalversion.CompatibilityStatus `json:"status"`
	CLI         compatibilityCLI                    `json:"cli"`
	Backend     compatibilityBackend                `json:"backend"`
}

var compatibilityPlatform string

var compatibilityCmd = &cobra.Command{
	Use:   "compatibility",
	Short: i18n.T("compatibility_short"),
	Long:  i18n.T("compatibility_long"),
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return errTooManyArgs(cmd, len(args), 0)
		}
		return nil
	},
	RunE: runCompatibility,
}

func runCompatibility(_ *cobra.Command, _ []string) error {
	platformURL, err := resolveCompatibilityPlatform(compatibilityPlatform)
	if err != nil {
		return err
	}

	info, apiErr := api.NewClient(platformURL).GetCompatibility()
	if apiErr != nil {
		return compatibilityAPIError(platformURL, apiErr)
	}

	result := newCompatibilityResult(platformURL, info)
	switch result.Status {
	case internalversion.CompatibilityCompatible:
		return writeCompatibilityResult(result)
	case internalversion.CompatibilityCLITooOld,
		internalversion.CompatibilityBackendTooOld,
		internalversion.CompatibilityBothTooOld:
		return compatibilityMismatchError(result)
	default:
		return &clierror.Error{
			Category: errorcodes.CategoryAPI,
			Code:     errorcodes.ErrAPIOther,
			Message:  i18n.T("err_invalid_compatibility_response"),
			Context:  compatibilityContext(result),
		}
	}
}

func compatibilityAPIError(platformURL string, err error) error {
	var requestErr *api.RequestError
	if !errors.As(err, &requestErr) || requestErr.HTTPStatus != 404 {
		return cliErrFromAPI(err)
	}

	result := newCompatibilityResult(platformURL, nil)
	context := compatibilityContext(result)
	for key, value := range apiContextFromRequest(requestErr) {
		context[key] = value
	}
	context["reason"] = "compatibility_endpoint_not_found"
	return &clierror.Error{
		Category: errorcodes.CategoryAPI,
		Code:     errorcodes.ErrAPIVersionMismatch,
		Message: compatibilityMismatchMessage(
			result,
			[]string{i18n.T("err_compatibility_endpoint_not_found")},
			i18n.T("err_compatibility_downgrade_cli"),
		),
		Context: context,
	}
}

func resolveCompatibilityPlatform(flagValue string) (string, error) {
	if platformURL := strings.TrimSpace(flagValue); platformURL != "" {
		return platformURL, nil
	}
	st, err := session.LoadState()
	if err != nil {
		return "", &clierror.Error{
			Category: errorcodes.CategorySystem,
			Code:     errorcodes.ErrConfigWriteFailed,
			Message:  i18n.T("err_config_write", err.Error()),
		}
	}
	if st.ActiveContext.PlatformURL == "" {
		return "", errUsageFromIssues([]usageIssue{{
			Code:    errorcodes.ErrMissingRequiredFlag,
			Message: i18n.T("err_compatibility_platform_required"),
			Field:   "platform",
		}})
	}
	return st.ActiveContext.PlatformURL, nil
}

func newCompatibilityResult(platformURL string, info *api.CompatibilityInfo) compatibilityResult {
	result := compatibilityResult{
		PlatformURL: platformURL,
		Status:      internalversion.CompatibilityUnknown,
		CLI: compatibilityCLI{
			ProductVersion:                internalversion.EffectiveProductVersion(),
			APIVersion:                    internalversion.APIVersion,
			MinSupportedBackendAPIVersion: internalversion.MinSupportedBackendAPIVersion,
		},
	}
	if info == nil {
		return result
	}
	result.Backend = compatibilityBackend{
		ProductVersion:            info.AppVersion,
		ShortCommitSHA:            info.ShortCommitSHA,
		BuildType:                 info.BuildType,
		BuildTime:                 info.BuildTime,
		APIVersion:                info.APIVersion,
		MinSupportedCLIAPIVersion: info.MinSupportedCLIAPIVersion,
		VersionsReported:          info.VersionsReported,
	}
	if !info.VersionsReported {
		return result
	}
	result.Status = internalversion.EvaluateCompatibility(info.APIVersion, info.MinSupportedCLIAPIVersion)
	return result
}

func writeCompatibilityResult(result compatibilityResult) error {
	if outputJSON {
		return output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(map[string]interface{}{
			"compatibility": result,
		}))
	}
	fmt.Printf("%s: %s\n", i18n.T("compatibility_label_platform"), result.PlatformURL)
	fmt.Printf("%s: %s\n", i18n.T("compatibility_label_status"), i18n.T("compatibility_status_compatible"))
	fmt.Printf("%s: %s\n", i18n.T("compatibility_label_cli_product_version"), result.CLI.ProductVersion)
	fmt.Printf("%s: %d\n", i18n.T("compatibility_label_cli_api_version"), result.CLI.APIVersion)
	fmt.Printf("%s: %d\n", i18n.T("compatibility_label_cli_min_backend"), result.CLI.MinSupportedBackendAPIVersion)
	fmt.Printf("%s: %s\n", i18n.T("compatibility_label_backend_product_version"), compatibilityBuildText(result.Backend.ProductVersion))
	fmt.Printf("%s: %s\n", i18n.T("compatibility_label_backend_short_commit_sha"), compatibilityBuildText(result.Backend.ShortCommitSHA))
	fmt.Printf("%s: %s\n", i18n.T("compatibility_label_backend_build_type"), compatibilityBuildText(result.Backend.BuildType))
	fmt.Printf("%s: %s\n", i18n.T("compatibility_label_backend_build_time"), compatibilityBuildText(result.Backend.BuildTime))
	fmt.Printf("%s: %d\n", i18n.T("compatibility_label_backend_api_version"), result.Backend.APIVersion)
	fmt.Printf("%s: %d\n", i18n.T("compatibility_label_backend_min_cli"), result.Backend.MinSupportedCLIAPIVersion)
	return nil
}

func compatibilityMismatchError(result compatibilityResult) *clierror.Error {
	var conditions []string
	var action string
	switch result.Status {
	case internalversion.CompatibilityCLITooOld:
		conditions = []string{
			i18n.T("err_cli_api_too_old", result.CLI.APIVersion, result.Backend.MinSupportedCLIAPIVersion),
		}
		action = i18n.T("err_compatibility_upgrade_cli")
	case internalversion.CompatibilityBackendTooOld:
		conditions = []string{
			i18n.T("err_backend_api_too_old", result.Backend.APIVersion, result.CLI.MinSupportedBackendAPIVersion),
		}
		action = i18n.T("err_compatibility_downgrade_cli")
	default:
		conditions = []string{
			i18n.T("err_cli_api_too_old", result.CLI.APIVersion, result.Backend.MinSupportedCLIAPIVersion),
			i18n.T("err_backend_api_too_old", result.Backend.APIVersion, result.CLI.MinSupportedBackendAPIVersion),
		}
		action = i18n.T("err_compatibility_conflicting_limits")
	}
	return &clierror.Error{
		Category: errorcodes.CategoryAPI,
		Code:     errorcodes.ErrAPIVersionMismatch,
		Message:  compatibilityMismatchMessage(result, conditions, action),
		Context:  compatibilityContext(result),
	}
}

func compatibilityMismatchMessage(result compatibilityResult, conditions []string, action string) string {
	lines := []string{
		i18n.T("err_compatibility_mismatch_summary"),
		"",
		fmt.Sprintf("%s: %s", i18n.T("compatibility_label_platform"), result.PlatformURL),
		i18n.T("err_compatibility_situation_heading"),
	}
	for _, condition := range conditions {
		lines = append(lines, "  - "+condition)
	}
	lines = append(
		lines,
		"",
		i18n.T("err_compatibility_versions_heading"),
		fmt.Sprintf("  %s: %s", i18n.T("compatibility_label_cli_product_version"), result.CLI.ProductVersion),
		fmt.Sprintf("  %s: %s", i18n.T("compatibility_label_cli_api_version"), compatibilityVersionText(result.CLI.APIVersion, true)),
		fmt.Sprintf("  %s: %s", i18n.T("compatibility_label_cli_min_backend"), compatibilityVersionText(result.CLI.MinSupportedBackendAPIVersion, true)),
		fmt.Sprintf("  %s: %s", i18n.T("compatibility_label_backend_product_version"), compatibilityBuildText(result.Backend.ProductVersion)),
		fmt.Sprintf("  %s: %s", i18n.T("compatibility_label_backend_short_commit_sha"), compatibilityBuildText(result.Backend.ShortCommitSHA)),
		fmt.Sprintf("  %s: %s", i18n.T("compatibility_label_backend_build_type"), compatibilityBuildText(result.Backend.BuildType)),
		fmt.Sprintf("  %s: %s", i18n.T("compatibility_label_backend_build_time"), compatibilityBuildText(result.Backend.BuildTime)),
		fmt.Sprintf("  %s: %s", i18n.T("compatibility_label_backend_api_version"), compatibilityVersionText(result.Backend.APIVersion, result.Backend.VersionsReported)),
		fmt.Sprintf("  %s: %s", i18n.T("compatibility_label_backend_min_cli"), compatibilityVersionText(result.Backend.MinSupportedCLIAPIVersion, result.Backend.VersionsReported)),
		"",
		i18n.T("err_compatibility_guidance_heading"),
		"  - "+i18n.T("err_compatibility_continue"),
		"  - "+i18n.T("err_compatibility_troubleshoot_first"),
		"  - "+action,
	)
	return strings.Join(lines, "\n")
}

func compatibilityVersionText(version int, reported bool) string {
	if !reported || version < 0 {
		return i18n.T("compatibility_value_unknown")
	}
	return fmt.Sprintf("%d", version)
}

func compatibilityBuildText(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || strings.EqualFold(value, "unknown") {
		return i18n.T("compatibility_value_unknown")
	}
	return value
}

func compatibilityContext(result compatibilityResult) map[string]interface{} {
	return map[string]interface{}{
		"platform_url":                          result.PlatformURL,
		"status":                                result.Status,
		"cli_product_version":                   result.CLI.ProductVersion,
		"cli_api_version":                       result.CLI.APIVersion,
		"cli_min_supported_backend_api_version": result.CLI.MinSupportedBackendAPIVersion,
		"backend_product_version":               result.Backend.ProductVersion,
		"backend_short_commit_sha":              result.Backend.ShortCommitSHA,
		"backend_build_type":                    result.Backend.BuildType,
		"backend_build_time":                    result.Backend.BuildTime,
		"backend_api_version":                   result.Backend.APIVersion,
		"backend_min_supported_cli_api_version": result.Backend.MinSupportedCLIAPIVersion,
	}
}

func init() {
	compatibilityCmd.Flags().StringVarP(&compatibilityPlatform, "platform", "p", "", "Platform URL")
	rootCmd.AddCommand(compatibilityCmd)
}
