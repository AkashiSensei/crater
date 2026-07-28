package cmd

import (
	"fmt"
	"strings"

	"github.com/raids-lab/crater/cli/internal/api"
	"github.com/raids-lab/crater/cli/internal/completion"
	"github.com/spf13/cobra"
)

var modelDownloadCmd = &cobra.Command{
	Use:   "model-download",
	Short: "View model and dataset downloads",
	Long:  "View model and dataset download records, details, and logs.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return errUnknownSubcommand(cmd, args[0])
		}
		return cmd.Help()
	},
}

var modelDownloadLsCmd = &cobra.Command{Use: "ls", Short: "List model downloads", Args: noArgs, RunE: runModelDownloadLs}
var modelDownloadGetCmd = &cobra.Command{Use: "get <id>", Short: "Get a model download", Args: exactArgs(1, "id"), RunE: runModelDownloadGet}
var modelDownloadLogsCmd = &cobra.Command{Use: "logs <id>", Short: "Show model download logs", Args: exactArgs(1, "id"), RunE: runModelDownloadLogs}
var adminModelDownloadCmd = &cobra.Command{Use: "model-download", Short: "View admin model and dataset downloads"}
var adminModelDownloadLsCmd = &cobra.Command{Use: "ls", Short: "List model downloads", Args: noArgs, RunE: runAdminModelDownloadLs}

func runModelDownloadLs(cmd *cobra.Command, _ []string) error {
	return runDownloadLs(cmd, nil)
}

func runAdminModelDownloadLs(cmd *cobra.Command, _ []string) error {
	options, err := readDownloadListOptions(cmd)
	if err != nil {
		return err
	}
	client, err := activeModelDownloadClient()
	if err != nil {
		return err
	}
	downloads, err := client.ListAdminDownloads()
	if err != nil {
		return cliErrFromAPI(err)
	}
	downloads = filterAdminModelDownloads(downloads, options)
	page := paginateList(downloads, options.ListOptions)
	return writeListPage("downloads", page, options.AllPages, printDownloadTable)
}

func runModelDownloadGet(cmd *cobra.Command, args []string) error {
	id, err := requiredUintArg(args, "download_label_id", "id")
	if err != nil {
		return err
	}
	return runRawRead(cmd, rawReadSpec{PayloadKey: "download", Path: fmt.Sprintf("%s/%s", api.ModelDownloadListPath, api.UintPath(id)), Params: noParams, Table: printRawObject})
}

func runModelDownloadLogs(cmd *cobra.Command, args []string) error {
	id, err := requiredUintArg(args, "download_label_id", "id")
	if err != nil {
		return err
	}
	return runRawStringRead(cmd, fmt.Sprintf("%s/%s/logs", api.ModelDownloadListPath, api.UintPath(id)), nil, "logs")
}

func filterAdminModelDownloads(
	downloads []api.ModelDownloadResp,
	options api.ModelDownloadListOptions,
) []api.ModelDownloadResp {
	search := strings.ToLower(options.Search)
	filtered := downloads[:0]
	for _, download := range downloads {
		if options.Category != "" && download.Category != options.Category {
			continue
		}
		if options.Status != "" && download.Status != options.Status {
			continue
		}
		if search != "" && !strings.Contains(strings.ToLower(download.Name), search) {
			continue
		}
		filtered = append(filtered, download)
	}
	return filtered
}

func init() {
	addDownloadListFlags(modelDownloadLsCmd)
	addDownloadListFlags(adminModelDownloadLsCmd)
	completion.RegisterFlagValue([]string{"model-download", "ls"}, "category", staticValueCompleter(downloadCategories, nil))
	completion.RegisterFlagValue([]string{"model-download", "ls"}, "status", staticValueCompleter(downloadStatuses, nil))
	completion.RegisterFlagValue([]string{"admin", "model-download", "ls"}, "category", staticValueCompleter(downloadCategories, nil))
	completion.RegisterFlagValue([]string{"admin", "model-download", "ls"}, "status", staticValueCompleter(downloadStatuses, nil))
	modelDownloadCmd.AddCommand(modelDownloadLsCmd, modelDownloadGetCmd, modelDownloadLogsCmd)
	rootCmd.AddCommand(modelDownloadCmd)
	adminModelDownloadCmd.AddCommand(adminModelDownloadLsCmd)
	adminCmd.AddCommand(adminModelDownloadCmd)
}
