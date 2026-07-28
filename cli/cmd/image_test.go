package cmd

import (
	"io"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/raids-lab/crater/cli/internal/api"
	"github.com/raids-lab/crater/cli/internal/i18n"
	"github.com/spf13/cobra"
)

func TestImageAndBuildListsExposeCommonPaginationFlags(t *testing.T) {
	commands := map[string]*cobra.Command{
		"image ls":             imageLsCmd,
		"admin image ls":       adminImageLsCmd,
		"image build ls":       imageBuildLsCmd,
		"admin image build-ls": adminImageBuildLsCmd,
	}
	for name, command := range commands {
		t.Run(name, func(t *testing.T) {
			for flagName, defaultValue := range map[string]string{
				"page":      "1",
				"page-size": strconv.Itoa(api.DefaultCLIPageSize),
				"all-pages": "false",
			} {
				flag := command.Flags().Lookup(flagName)
				if flag == nil {
					t.Fatalf("missing --%s", flagName)
				}
				if flag.DefValue != defaultValue {
					t.Fatalf("--%s default = %q, want %q", flagName, flag.DefValue, defaultValue)
				}
			}
		})
	}
}

func TestReadImageListOptionsAggregatesPaginationAndFilterIssues(t *testing.T) {
	previousLanguage := i18n.GetCurrentLanguage()
	i18n.SetLanguage("en")
	t.Cleanup(func() { i18n.SetLanguage(previousLanguage) })

	command := &cobra.Command{}
	addListPaginationFlags(command)
	command.Flags().String("type", "", "")
	command.Flags().String("visibility", "", "")
	for flag, value := range map[string]string{
		"page":       "0",
		"page-size":  "201",
		"type":       "invalid-task",
		"visibility": "SharedWithEveryone",
	} {
		if err := command.Flags().Set(flag, value); err != nil {
			t.Fatalf("set --%s: %v", flag, err)
		}
	}

	_, _, err := readImageListOptions(command, true)
	if err == nil {
		t.Fatal("expected invalid image list options to fail")
	}
	for _, message := range []string{
		"page must be at least 1",
		"page-size must be between 1 and 200",
		"invalid image type: invalid-task",
		"invalid image visibility: SharedWithEveryone",
	} {
		if !strings.Contains(err.Error(), message) {
			t.Fatalf("error %q does not include %q", err, message)
		}
	}
}

func TestImageFiltersRunBeforePaginationAndPreserveOrder(t *testing.T) {
	command := &cobra.Command{}
	command.Flags().String("arch", "", "")
	command.Flags().String("visibility", "", "")
	command.Flags().String("owner", "", "")
	command.Flags().String("search", "", "")
	for flag, value := range map[string]string{
		"arch":       "linux/amd64",
		"visibility": "Private",
		"owner":      "ALI",
		"search":     "tensorflow",
	} {
		if err := command.Flags().Set(flag, value); err != nil {
			t.Fatalf("set --%s: %v", flag, err)
		}
	}
	description := "TensorFlow tools"
	images := []api.ImageInfo{
		{
			ID: 3, ImageLink: "registry.example/tensorflow:v2", TaskType: "custom",
			ImageShareStatus: "Private", Archs: []string{"linux/amd64"},
			UserInfo: api.UserInfo{Username: "alice"},
		},
		{
			ID: 2, ImageLink: "registry.example/torch:v2", Description: &description,
			TaskType: "custom", ImageShareStatus: "Private",
			Archs: []string{"linux/amd64"}, UserInfo: api.UserInfo{Nickname: "Alice"},
		},
		{
			ID: 1, ImageLink: "registry.example/tensorflow:v1", TaskType: "custom",
			ImageShareStatus: "Private", Archs: []string{"linux/amd64"},
			UserInfo: api.UserInfo{Username: "bob"},
		},
		{
			ID: 4, ImageLink: "registry.example/tensorflow:pytorch", TaskType: "pytorch",
			ImageShareStatus: "Private", Archs: []string{"linux/amd64"},
			UserInfo: api.UserInfo{Username: "alice"},
		},
	}

	filtered := filterImagesByTaskType(images, "custom")
	filtered = filterImages(command, filtered)
	page := paginateList(filtered, api.ListOptions{Page: 2, PageSize: 1})
	if page.Total != 2 || len(page.Items) != 1 || page.Items[0].ID != 2 {
		t.Fatalf("unexpected filtered page: %#v", page)
	}
}

func TestImageBuildPageUsesTypedItemsAndTableSummary(t *testing.T) {
	previousLanguage := i18n.GetCurrentLanguage()
	i18n.SetLanguage("en")
	t.Cleanup(func() { i18n.SetLanguage(previousLanguage) })
	previousOutputJSON := outputJSON
	outputJSON = false
	t.Cleanup(func() { outputJSON = previousOutputJSON })

	builds := []api.KanikoInfo{{ID: 8}, {ID: 7}}
	page := paginateList(builds, api.ListOptions{Page: 2, PageSize: 1})
	if len(page.Items) != 1 || page.Items[0].ID != 7 {
		t.Fatalf("unexpected typed build page: %#v", page)
	}
	got := captureImageTestStdout(t, func() {
		if err := writeListPage("builds", page, false, printKanikoTable); err != nil {
			t.Fatalf("write build page: %v", err)
		}
	})
	if !strings.Contains(got, "Page 2, 2 items total") {
		t.Fatalf("table output has no page summary: %q", got)
	}
}

func TestPrintImageTableIncludesVisibility(t *testing.T) {
	previousLanguage := i18n.GetCurrentLanguage()
	i18n.SetLanguage("en")
	t.Cleanup(func() { i18n.SetLanguage(previousLanguage) })

	got := captureImageTestStdout(t, func() {
		printImageTable([]api.ImageInfo{{
			ID:               1,
			ImageLink:        "registry.example/demo:v1",
			TaskType:         "custom",
			ImageShareStatus: "Private",
			Archs:            []string{"linux/amd64"},
			UserInfo:         api.UserInfo{Nickname: "alice"},
		}})
	})
	if !strings.Contains(got, "VISIBILITY") || !strings.Contains(got, "Private") {
		t.Fatalf("table must display image visibility, got %q", got)
	}
	if strings.Contains(got, "CREATED") {
		t.Fatalf("table must not display the removed CREATED column, got %q", got)
	}
}

func TestImageCommandGroupsRejectUnknownSubcommands(t *testing.T) {
	tests := []struct {
		name string
		cmd  *cobra.Command
	}{
		{name: "build", cmd: imageBuildCmd},
		{name: "share", cmd: imageShareCmd},
		{name: "user cuda", cmd: imageCudaCmd},
		{name: "harbor", cmd: imageHarborCmd},
		{name: "quota", cmd: imageQuotaCmd},
		{name: "admin image", cmd: adminImageCmd},
		{name: "admin cuda", cmd: adminImageCudaCmd},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Cobra treats arguments on a command group as positional input when
			// no matching child exists. Exercise the group's RunE directly so the
			// contract remains independent of the process-wide root command state.
			err := tt.cmd.RunE(tt.cmd, []string{"unknown"})
			if err == nil || !strings.Contains(err.Error(), `unknown command "unknown"`) {
				t.Fatalf("unknown subcommand error = %v", err)
			}
		})
	}
}

func captureImageTestStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdout pipe: %v", err)
	}
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = old })

	fn()
	if err := w.Close(); err != nil {
		t.Fatalf("close stdout pipe: %v", err)
	}
	output, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read stdout pipe: %v", err)
	}
	return string(output)
}
