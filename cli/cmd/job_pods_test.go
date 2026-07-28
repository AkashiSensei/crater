package cmd

import (
	"strings"
	"testing"

	"github.com/raids-lab/crater/cli/internal/api"
	"github.com/spf13/cobra"
)

func TestFilterJobPodsBeforePagination(t *testing.T) {
	pods := []api.PodDetail{
		{Name: "worker-b", Phase: "Running"},
		{Name: "service", Phase: "Pending"},
		{Name: "worker-a", Phase: "Running"},
	}
	filtered := filterJobPods(pods, jobPodListOptions{Status: "Running", Search: "worker"})
	if len(filtered) != 2 {
		t.Fatalf("unexpected filtered pods: %#v", filtered)
	}
	page := paginateList(filtered, api.ListOptions{PageSize: 1})
	if page.Total != 2 || len(page.Items) != 1 {
		t.Fatalf("unexpected page: %#v", page)
	}
}

func TestJobPodOptionsAggregateIssues(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("status", "", "")
	cmd.Flags().String("search", "", "")
	addListPaginationFlags(cmd)
	_ = cmd.Flags().Set("page", "0")
	_ = cmd.Flags().Set("status", "NotAPhase")

	_, err := readJobPodListOptions(cmd)
	if err == nil || !strings.Contains(err.Error(), "page") ||
		!strings.Contains(err.Error(), "NotAPhase") {
		t.Fatalf("unexpected error: %v", err)
	}
}
