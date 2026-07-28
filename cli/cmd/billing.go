package cmd

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/raids-lab/crater/cli/internal/api"
	"github.com/raids-lab/crater/cli/internal/i18n"
	"github.com/spf13/cobra"
)

var billingCmd = &cobra.Command{
	Use:   "billing",
	Short: "View billing information",
	Long:  "View billing status, prices, summaries, and job billing records.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return errUnknownSubcommand(cmd, args[0])
		}
		return cmd.Help()
	},
}

var billingStatusCmd = &cobra.Command{Use: "status", Short: "Get billing feature status", Args: noArgs, RunE: runBillingStatus}
var billingSummaryCmd = &cobra.Command{Use: "summary", Short: "Get current billing summary", Args: noArgs, RunE: runContextBilling}
var billingPricesCmd = &cobra.Command{Use: "prices", Short: "List billing prices", Args: noArgs, RunE: runResourcePrices}
var billingJobsCmd = &cobra.Command{Use: "jobs", Short: "List job billing records", Args: noArgs, RunE: runBillingJobs}
var billingJobCmd = &cobra.Command{Use: "job <name>", Short: "Get job billing detail", Args: exactArgs(1, "job-name"), RunE: runBillingJob}
var adminBillingCmd = &cobra.Command{Use: "billing", Short: "View admin billing information"}
var adminBillingStatusCmd = &cobra.Command{Use: "status", Short: "Get billing feature status", Args: noArgs, RunE: runAdminBillingStatus}
var adminBillingJobsCmd = &cobra.Command{Use: "jobs", Short: "List job billing records", Args: noArgs, RunE: runAdminBillingJobs}

func runBillingStatus(cmd *cobra.Command, _ []string) error {
	return runRawRead(cmd, rawReadSpec{PayloadKey: "billing", Path: api.SystemConfigPrefix + "/billing", Params: noParams, Table: printRawObject})
}

func runAdminBillingStatus(cmd *cobra.Command, _ []string) error {
	return runRawRead(cmd, rawReadSpec{PayloadKey: "billing", Path: api.AdminSysConfigPfx + "/billing", Params: noParams, Table: printRawObject})
}

func runBillingJobs(cmd *cobra.Command, _ []string) error {
	return listJobBilling(cmd, false)
}

func runAdminBillingJobs(cmd *cobra.Command, _ []string) error {
	return listJobBilling(cmd, true)
}

type billingJobListOptions struct {
	api.ListOptions
	Admin  bool
	All    bool
	User   string
	Days   int
	Search string
}

func listJobBilling(cmd *cobra.Command, admin bool) error {
	options, err := readBillingJobListOptions(cmd, admin)
	if err != nil {
		return err
	}
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	records, err := client.ListJobBilling(api.JobBillingListOptions{
		Admin:    options.Admin,
		All:      options.All,
		Username: options.User,
		Days:     options.Days,
	})
	if err != nil {
		return cliErrFromAPI(err)
	}
	records = filterJobBilling(records, options.Search)
	sortJobBilling(records)
	page := paginateList(records, options.ListOptions)
	return writeListPage("billing", page, options.AllPages, printJobBillingTable)
}

func readBillingJobListOptions(cmd *cobra.Command, admin bool) (billingJobListOptions, error) {
	listOptions, issues := listPaginationOptions(cmd, maxCLIPageSize)
	days, _ := cmd.Flags().GetInt("days")
	if days < -1 {
		issues = append(issues, invalidIssue("days", i18n.T("err_invalid_days")))
	}
	if len(issues) > 0 {
		return billingJobListOptions{}, errUsageFromIssues(issues)
	}

	all := false
	if !admin {
		all, _ = cmd.Flags().GetBool("all")
	}
	user, _ := cmd.Flags().GetString("user")
	search, _ := cmd.Flags().GetString("search")
	return billingJobListOptions{
		ListOptions: listOptions,
		Admin:       admin,
		All:         all,
		User:        strings.TrimSpace(user),
		Days:        days,
		Search:      strings.TrimSpace(search),
	}, nil
}

func filterJobBilling(records []api.JobBilling, search string) []api.JobBilling {
	search = strings.ToLower(strings.TrimSpace(search))
	if search == "" {
		return records
	}
	filtered := make([]api.JobBilling, 0, len(records))
	for _, record := range records {
		if strings.Contains(strings.ToLower(record.Name), search) ||
			strings.Contains(strings.ToLower(record.JobName), search) {
			filtered = append(filtered, record)
		}
	}
	return filtered
}

func sortJobBilling(records []api.JobBilling) {
	sort.SliceStable(records, func(left, right int) bool {
		leftJobName := strings.ToLower(records[left].JobName)
		rightJobName := strings.ToLower(records[right].JobName)
		if leftJobName != rightJobName {
			return leftJobName < rightJobName
		}
		leftName := strings.ToLower(records[left].Name)
		rightName := strings.ToLower(records[right].Name)
		if leftName != rightName {
			return leftName < rightName
		}
		return records[left].BilledPointsTotal < records[right].BilledPointsTotal
	})
}

func addBillingJobListFlags(cmd *cobra.Command, admin bool) {
	if !admin {
		cmd.Flags().Bool("all", false, i18n.T("flag_all"))
	}
	cmd.Flags().String("user", "", i18n.T("flag_user"))
	cmd.Flags().Int("days", 30, i18n.T("flag_days"))
	cmd.Flags().String("search", "", i18n.T("flag_search"))
	addListPaginationFlags(cmd)
}

func runBillingJob(cmd *cobra.Command, args []string) error {
	name, err := requiredArg(args, "job_label_name", "name")
	if err != nil {
		return err
	}
	return runRawRead(cmd, rawReadSpec{PayloadKey: "billing", Path: fmt.Sprintf("%s/%s/billing", api.VCJobsPrefix, name), Params: noParams, Table: printRawObject})
}

func printJobBillingTable(records []api.JobBilling) {
	fmt.Printf("%s %s %s\n",
		i18n.PadRight(i18n.T("table_name"), 28),
		i18n.PadRight(i18n.T("table_job_name"), 34),
		i18n.PadRight(i18n.T("table_points"), 14),
	)
	for _, record := range records {
		fmt.Printf("%s %s %s\n",
			i18n.PadRight(record.Name, 28),
			i18n.PadRight(record.JobName, 34),
			i18n.PadRight(strconv.FormatFloat(record.BilledPointsTotal, 'f', -1, 64), 14),
		)
	}
}

func init() {
	addBillingJobListFlags(billingJobsCmd, false)
	billingCmd.AddCommand(billingStatusCmd, billingSummaryCmd, billingPricesCmd, billingJobsCmd, billingJobCmd)
	rootCmd.AddCommand(billingCmd)
	addBillingJobListFlags(adminBillingJobsCmd, true)
	adminBillingCmd.AddCommand(adminBillingStatusCmd, adminBillingJobsCmd)
	adminCmd.AddCommand(adminBillingCmd)
}
