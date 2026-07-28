package cmd

import (
	"reflect"
	"strings"
	"testing"

	"github.com/raids-lab/crater/cli/internal/api"
	"github.com/spf13/cobra"
)

func TestReadListPaginationOptionsDefaults(t *testing.T) {
	cmd := &cobra.Command{}
	addListPaginationFlags(cmd)

	options, err := readListPaginationOptions(cmd)
	if err != nil {
		t.Fatalf("readListPaginationOptions: %v", err)
	}
	if options.Page != 1 || options.PageSize != api.DefaultCLIPageSize || options.AllPages {
		t.Fatalf("unexpected defaults: %#v", options)
	}
}

func TestReadListPaginationOptionsUsesMaxBatchForAllPages(t *testing.T) {
	tests := []struct {
		name        string
		maxPageSize int
		explicit    string
		want        int
	}{
		{name: "common maximum", maxPageSize: maxCLIPageSize, want: maxCLIPageSize},
		{name: "endpoint maximum", maxPageSize: 100, want: 100},
		{name: "explicit value", maxPageSize: maxCLIPageSize, explicit: "15", want: 15},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			addListPaginationFlags(cmd)
			if err := cmd.Flags().Set("all-pages", "true"); err != nil {
				t.Fatal(err)
			}
			if tt.explicit != "" {
				if err := cmd.Flags().Set("page-size", tt.explicit); err != nil {
					t.Fatal(err)
				}
			}

			options, err := readListPaginationOptionsWithMax(cmd, tt.maxPageSize)
			if err != nil {
				t.Fatalf("readListPaginationOptionsWithMax: %v", err)
			}
			if options.PageSize != tt.want || !options.AllPages {
				t.Fatalf("unexpected all-pages options: %#v", options)
			}
		})
	}
}

func TestReadListPaginationOptionsRejectsExplicitInvalidAllPagesSize(t *testing.T) {
	cmd := &cobra.Command{}
	addListPaginationFlags(cmd)
	if err := cmd.Flags().Set("all-pages", "true"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("page-size", "201"); err != nil {
		t.Fatal(err)
	}

	_, err := readListPaginationOptions(cmd)
	if err == nil || !strings.Contains(err.Error(), "page-size must be between 1 and 200") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReadListPaginationOptionsRejectsInvalidValues(t *testing.T) {
	cmd := &cobra.Command{}
	addListPaginationFlags(cmd)
	if err := cmd.Flags().Set("page", "0"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("page-size", "201"); err != nil {
		t.Fatal(err)
	}

	_, err := readListPaginationOptions(cmd)
	if err == nil {
		t.Fatal("expected invalid pagination values to fail")
	}
	if !strings.Contains(err.Error(), "page must be at least 1") ||
		!strings.Contains(err.Error(), "page-size must be between 1 and 200") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPaginateList(t *testing.T) {
	page := paginateList([]int{1, 2, 3, 4, 5}, api.ListOptions{Page: 2, PageSize: 2})
	if !reflect.DeepEqual(page.Items, []int{3, 4}) ||
		page.Total != 5 || page.Page != 2 || page.PageSize != 2 {
		t.Fatalf("unexpected page: %#v", page)
	}
}

func TestPaginateListEmptyAndAllPages(t *testing.T) {
	empty := paginateList([]int{1}, api.ListOptions{Page: 2, PageSize: 1})
	if empty.Items == nil || len(empty.Items) != 0 || empty.Total != 1 {
		t.Fatalf("unexpected empty page: %#v", empty)
	}
	all := paginateList([]int{1, 2}, api.ListOptions{
		Page: 9, PageSize: 1, AllPages: true,
	})
	if !reflect.DeepEqual(all.Items, []int{1, 2}) || all.Page != 1 || all.Total != 2 {
		t.Fatalf("unexpected all-pages result: %#v", all)
	}
}

func TestListPagePayloadOmitsPaginationForAllPages(t *testing.T) {
	page := api.Page[int]{Total: 0, Page: 1, PageSize: 15}
	current := listPagePayload("items", page, false)
	if _, ok := current["pagination"]; !ok {
		t.Fatalf("current-page payload missing pagination: %#v", current)
	}
	currentItems, ok := current["items"].([]int)
	if !ok || currentItems == nil {
		t.Fatalf("current-page empty items must be a non-nil array: %#v", current["items"])
	}
	all := listPagePayload("items", page, true)
	if _, ok := all["pagination"]; ok {
		t.Fatalf("all-pages payload unexpectedly contains pagination: %#v", all)
	}
	items, ok := all["items"].([]int)
	if !ok || items == nil {
		t.Fatalf("empty items must be a non-nil array: %#v", all["items"])
	}
}

func TestNodePodFiltersExplicitNamespaceBeforePagination(t *testing.T) {
	const workloadNamespace = "team-workloads"
	pods := []api.PodInfo{
		{Name: "workspace-b", Namespace: workloadNamespace, Status: "Running"},
		{Name: "system", Namespace: "kube-system", Status: "Running"},
		{Name: "workspace-a", Namespace: workloadNamespace, Status: "Pending"},
	}
	filtered := filterNodePods(pods, nodePodListOptions{
		Namespace: workloadNamespace,
	})
	page := paginateList(filtered, api.ListOptions{PageSize: 1})
	if page.Total != 2 || len(page.Items) != 1 {
		t.Fatalf("unexpected filtered page: %#v", page)
	}
	if page.Items[0].Namespace != workloadNamespace {
		t.Fatalf("unexpected namespace: %#v", page.Items[0])
	}
}

func TestNodePodNamespaceFlagsConflict(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("namespace", "", "")
	cmd.Flags().String("status", "", "")
	cmd.Flags().String("type", "", "")
	cmd.Flags().String("search", "", "")
	cmd.Flags().Bool("all-namespaces", false, "")
	addListPaginationFlags(cmd)
	if err := cmd.Flags().Set("namespace", "kube-system"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("all-namespaces", "true"); err != nil {
		t.Fatal(err)
	}

	_, err := readNodePodListOptions(cmd)
	if err == nil || !strings.Contains(err.Error(), "--namespace and --all-namespaces") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNodePodNamespaceIsRequiredUnlessAllNamespaces(t *testing.T) {
	newCommand := func() *cobra.Command {
		cmd := &cobra.Command{}
		cmd.Flags().String("namespace", "", "")
		cmd.Flags().String("status", "", "")
		cmd.Flags().String("type", "", "")
		cmd.Flags().String("search", "", "")
		cmd.Flags().Bool("all-namespaces", false, "")
		addListPaginationFlags(cmd)
		return cmd
	}

	t.Run("missing namespace", func(t *testing.T) {
		_, err := readNodePodListOptions(newCommand())
		if err == nil || !strings.Contains(err.Error(), "--namespace or --all-namespaces is required") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("all namespaces", func(t *testing.T) {
		cmd := newCommand()
		if err := cmd.Flags().Set("all-namespaces", "true"); err != nil {
			t.Fatal(err)
		}
		options, err := readNodePodListOptions(cmd)
		if err != nil {
			t.Fatalf("readNodePodListOptions: %v", err)
		}
		if !options.AllNamespaces || options.Namespace != "" {
			t.Fatalf("unexpected options: %#v", options)
		}
	})
}

func TestNormalizePodTypesPrefersControllerOwner(t *testing.T) {
	notController := false
	controller := true
	pods := []api.PodInfo{{
		Name: "worker",
		OwnerReference: []api.OwnerReference{
			{APIVersion: "apps/v1", Kind: "ReplicaSet", Controller: &notController},
			{APIVersion: "batch.volcano.sh/v1alpha1", Kind: "Job", Controller: &controller},
		},
	}}
	normalizePodTypes(pods)
	if pods[0].Type != "batch.volcano.sh/v1alpha1/Job" {
		t.Fatalf("unexpected pod type: %q", pods[0].Type)
	}
}

func TestNormalizePodTypesFindsSupportedOwnerWithoutController(t *testing.T) {
	pods := []api.PodInfo{{
		Name: "worker",
		OwnerReference: []api.OwnerReference{
			{APIVersion: "v1", Kind: "ConfigMap"},
			{APIVersion: "aisystem.github.com/v1alpha1", Kind: "AIJob"},
		},
	}}
	normalizePodTypes(pods)
	if pods[0].Type != "aisystem.github.com/v1alpha1/AIJob" {
		t.Fatalf("unexpected pod type: %q", pods[0].Type)
	}
}

func TestFinalizeJobListPagePaginatesLocalFilters(t *testing.T) {
	items := []api.JobInfo{{Name: "one"}, {Name: "two"}, {Name: "three"}}
	page := finalizeJobListPage(
		items,
		api.Page[api.JobInfo]{},
		api.JobListOptions{ListOptions: api.ListOptions{Page: 2, PageSize: 1}},
		true,
	)
	if page.Total != 3 || len(page.Items) != 1 || page.Items[0].Name != "two" {
		t.Fatalf("unexpected locally filtered page: %#v", page)
	}
}

func TestFinalizeJobListPagePreservesServerMetadata(t *testing.T) {
	items := []api.JobInfo{{Name: "one"}}
	serverPage := api.Page[api.JobInfo]{Total: 20, Page: 2, PageSize: 15}
	page := finalizeJobListPage(
		items,
		serverPage,
		api.JobListOptions{ListOptions: api.ListOptions{Page: 2, PageSize: 15}},
		false,
	)
	if page.Total != 20 || page.Page != 2 || !reflect.DeepEqual(page.Items, items) {
		t.Fatalf("unexpected server page: %#v", page)
	}
}

func TestJobFetchListOptionsPreservesAllPagesBatchSize(t *testing.T) {
	allPages := jobFetchListOptions(api.JobListOptions{
		ListOptions: api.ListOptions{Page: 7, PageSize: 25, AllPages: true},
	}, true)
	if allPages.PageSize != 25 {
		t.Fatalf("all-pages page size = %d, want 25", allPages.PageSize)
	}

	localPage := jobFetchListOptions(api.JobListOptions{
		ListOptions: api.ListOptions{Page: 2, PageSize: 15},
	}, true)
	if localPage.PageSize != maxCLIPageSize {
		t.Fatalf("local-filter fetch page size = %d, want %d", localPage.PageSize, maxCLIPageSize)
	}
}

func TestJobListIssuesAggregateSearchAndSortValidation(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Int("days", 0, "")
	cmd.Flags().String("search", "", "")
	cmd.Flags().String("sort", "", "")
	cmd.Flags().String("status", "", "")
	cmd.Flags().String("type", "", "")
	cmd.Flags().Bool("interactive", false, "")
	cmd.Flags().Bool("batch", false, "")
	cmd.Flags().String("from", "", "")
	cmd.Flags().String("to", "", "")

	_ = cmd.Flags().Set("search", strings.Repeat("界", jobMaxSearchRunes+1))
	_ = cmd.Flags().Set("sort", "name,-name,unsupported,createdAt")
	issues := jobListFilterIssues(cmd)

	messages := make([]string, len(issues))
	for index := range issues {
		messages[index] = issues[index].Message
	}
	got := strings.Join(messages, "\n")
	for _, want := range []string{
		"128",
		"at most 3",
		`duplicate sort field "name"`,
		`unsupported sort field "unsupported"`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("issues %q do not contain %q", got, want)
		}
	}
}
