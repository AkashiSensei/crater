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

package version

import (
	"runtime/debug"
	"strings"
)

const (
	APIVersion                       = 1
	MinSupportedBackendAPIVersion    = 1
	APIVersionHeader                 = "X-Crater-API-Version"
	defaultDevelopmentProductVersion = "dev"
)

// ProductVersion may be overridden at build time with -ldflags.
var ProductVersion = defaultDevelopmentProductVersion

type CompatibilityStatus string

const (
	CompatibilityUnknown       CompatibilityStatus = "unknown"
	CompatibilityCompatible    CompatibilityStatus = "compatible"
	CompatibilityCLITooOld     CompatibilityStatus = "cli_maybe_too_old"
	CompatibilityBackendTooOld CompatibilityStatus = "backend_maybe_too_old"
	CompatibilityBothTooOld    CompatibilityStatus = "both_maybe_too_old"
)

func EffectiveProductVersion() string {
	if version := strings.TrimSpace(ProductVersion); version != "" && version != defaultDevelopmentProductVersion {
		return strings.TrimPrefix(version, "v")
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		version := strings.TrimSpace(info.Main.Version)
		if version != "" && version != "(devel)" {
			return strings.TrimPrefix(version, "v")
		}
	}
	return defaultDevelopmentProductVersion
}

func UserAgent() string {
	return "crater-cli/" + EffectiveProductVersion()
}

func EvaluateCompatibility(backendAPIVersion, minSupportedCLIAPIVersion int) CompatibilityStatus {
	return evaluateCompatibility(
		APIVersion,
		MinSupportedBackendAPIVersion,
		backendAPIVersion,
		minSupportedCLIAPIVersion,
	)
}

func evaluateCompatibility(
	cliAPIVersion,
	minSupportedBackendAPIVersion,
	backendAPIVersion,
	minSupportedCLIAPIVersion int,
) CompatibilityStatus {
	if cliAPIVersion <= 0 || minSupportedBackendAPIVersion <= 0 ||
		backendAPIVersion < 0 || minSupportedCLIAPIVersion < 0 {
		return CompatibilityUnknown
	}
	cliTooOld := cliAPIVersion < minSupportedCLIAPIVersion
	backendTooOld := backendAPIVersion < minSupportedBackendAPIVersion
	switch {
	case cliTooOld && backendTooOld:
		return CompatibilityBothTooOld
	case cliTooOld:
		return CompatibilityCLITooOld
	case backendTooOld:
		return CompatibilityBackendTooOld
	default:
		return CompatibilityCompatible
	}
}
