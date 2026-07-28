package cmd

import (
	"reflect"
	"strings"
	"testing"

	"github.com/raids-lab/crater/cli/internal/api"
	"github.com/raids-lab/crater/cli/internal/i18n"
	"github.com/spf13/cobra"
)

func TestBillingJobListFlags(t *testing.T) {
	userCommand := &cobra.Command{}
	addBillingJobListFlags(userCommand, false)
	if userCommand.Flags().Lookup("all") == nil {
		t.Fatal("user billing jobs must expose --all")
	}

	adminCommand := &cobra.Command{}
	addBillingJobListFlags(adminCommand, true)
	if adminCommand.Flags().Lookup("all") != nil {
		t.Fatal("admin billing jobs must not expose --all")
	}
	for _, name := range []string{"user", "days", "search", "page", "page-size", "all-pages"} {
		if adminCommand.Flags().Lookup(name) == nil {
			t.Fatalf("admin billing jobs is missing --%s", name)
		}
	}
}

func TestReadBillingJobListOptionsDefaults(t *testing.T) {
	command := &cobra.Command{}
	addBillingJobListFlags(command, false)

	options, err := readBillingJobListOptions(command, false)
	if err != nil {
		t.Fatalf("readBillingJobListOptions: %v", err)
	}
	if options.Page != 1 || options.PageSize != api.DefaultCLIPageSize || options.AllPages {
		t.Fatalf("pagination options = %#v", options.ListOptions)
	}
	if options.Days != 30 || options.Admin || options.All || options.User != "" || options.Search != "" {
		t.Fatalf("billing options = %#v", options)
	}
}

func TestReadBillingJobListOptionsAggregatesUsageIssues(t *testing.T) {
	previousLanguage := i18n.GetCurrentLanguage()
	i18n.SetLanguage("en")
	t.Cleanup(func() { i18n.SetLanguage(previousLanguage) })

	command := &cobra.Command{}
	addBillingJobListFlags(command, false)
	for flag, value := range map[string]string{
		"page":      "0",
		"page-size": "201",
		"days":      "-2",
	} {
		if err := command.Flags().Set(flag, value); err != nil {
			t.Fatalf("set --%s: %v", flag, err)
		}
	}

	_, err := readBillingJobListOptions(command, false)
	if err == nil {
		t.Fatal("expected invalid list options to fail")
	}
	for _, message := range []string{
		"page must be at least 1",
		"page-size must be between 1 and 200",
		"days must be -1 or greater",
	} {
		if !strings.Contains(err.Error(), message) {
			t.Fatalf("error %q does not include %q", err, message)
		}
	}
}

func TestFilterSortAndPaginateJobBilling(t *testing.T) {
	records := []api.JobBilling{
		{Name: "Zeta", JobName: "train-b", BilledPointsTotal: 3},
		{Name: "Other", JobName: "infer-c", BilledPointsTotal: 4},
		{Name: "Alpha Train", JobName: "train-a", BilledPointsTotal: 1},
	}

	filtered := filterJobBilling(records, "TRAIN")
	sortJobBilling(filtered)
	page := paginateList(filtered, api.ListOptions{Page: 1, PageSize: 1})

	if page.Total != 2 {
		t.Fatalf("total = %d, want 2", page.Total)
	}
	got := []string{page.Items[0].JobName}
	if !reflect.DeepEqual(got, []string{"train-a"}) {
		t.Fatalf("first page job names = %v, want [train-a]", got)
	}
}

func TestBillingListUsesCommonPaginationPayload(t *testing.T) {
	page := api.Page[api.JobBilling]{
		Items: []api.JobBilling{{
			Name: "training", JobName: "train-a", BilledPointsTotal: 1.5,
		}},
		Total: 21, Page: 2, PageSize: 15,
	}

	paged := listPagePayload("billing", page, false)
	metadata, ok := paged["pagination"].(map[string]interface{})
	if !ok {
		t.Fatalf("pagination metadata = %#v", paged["pagination"])
	}
	if metadata["page"] != 2 || metadata["page_size"] != 15 || metadata["total"] != int64(21) {
		t.Fatalf("pagination metadata = %#v", metadata)
	}
	records, ok := paged["billing"].([]api.JobBilling)
	if !ok || len(records) != 1 || records[0].JobName != "train-a" {
		t.Fatalf("billing payload = %#v", paged["billing"])
	}

	allPages := listPagePayload("billing", page, true)
	if _, ok := allPages["pagination"]; ok {
		t.Fatalf("all-pages payload unexpectedly has pagination: %#v", allPages)
	}
}

func TestPrintJobBillingTableUsesTypedFieldsAndI18nHeaders(t *testing.T) {
	previousLanguage := i18n.GetCurrentLanguage()
	i18n.SetLanguage("en")
	t.Cleanup(func() { i18n.SetLanguage(previousLanguage) })

	got := captureOrderListStdout(t, func() {
		printJobBillingTable([]api.JobBilling{{
			Name: "training", JobName: "train-a", BilledPointsTotal: 12.5,
		}})
	})
	for _, value := range []string{
		i18n.T("table_name"),
		i18n.T("table_job_name"),
		i18n.T("table_points"),
		"training",
		"train-a",
		"12.5",
	} {
		if !strings.Contains(got, value) {
			t.Fatalf("table output %q does not include %q", got, value)
		}
	}
}
