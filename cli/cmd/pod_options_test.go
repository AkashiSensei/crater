package cmd

import (
	"errors"
	"testing"

	"github.com/raids-lab/crater/cli/internal/clierror"
	"github.com/raids-lab/crater/cli/pkg/errorcodes"
	"github.com/spf13/cobra"
)

func newPodOptionsTestCommand(t *testing.T, namespace string) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{}
	cmd.Flags().String("namespace", "", "")
	if namespace != "" {
		if err := cmd.Flags().Set("namespace", namespace); err != nil {
			t.Fatal(err)
		}
	}
	return cmd
}

func TestPodNSAndNameRequiresNamespace(t *testing.T) {
	cmd := newPodOptionsTestCommand(t, "")

	_, _, err := podNSAndName(cmd, []string{"worker-0"})
	var cliErr *clierror.Error
	if !errors.As(err, &cliErr) {
		t.Fatalf("expected structured CLI error, got %T: %v", err, err)
	}
	if cliErr.Category != errorcodes.CategoryUsage || cliErr.Code != errorcodes.ErrMissingRequiredFlag {
		t.Fatalf("unexpected CLI error: %#v", cliErr)
	}
}

func TestPodNSAndNameSupportsNamespaceFlagAndLegacyArguments(t *testing.T) {
	t.Run("namespace flag", func(t *testing.T) {
		cmd := newPodOptionsTestCommand(t, "team-a")

		namespace, name, err := podNSAndName(cmd, []string{"worker-0"})
		if err != nil {
			t.Fatalf("podNSAndName: %v", err)
		}
		if namespace != "team-a" || name != "worker-0" {
			t.Fatalf("unexpected target: namespace=%q name=%q", namespace, name)
		}
	})

	t.Run("legacy positional namespace", func(t *testing.T) {
		cmd := newPodOptionsTestCommand(t, "")

		namespace, name, err := podNSAndName(cmd, []string{"legacy-ns", "worker-0"})
		if err != nil {
			t.Fatalf("podNSAndName: %v", err)
		}
		if namespace != "legacy-ns" || name != "worker-0" {
			t.Fatalf("unexpected target: namespace=%q name=%q", namespace, name)
		}
	})
}

func TestPodLogsRequiresNamespaceAndSupportsLegacyArguments(t *testing.T) {
	t.Run("missing namespace", func(t *testing.T) {
		cmd := newPodOptionsTestCommand(t, "")

		_, _, _, err := podNSNameAndContainer(cmd, []string{"worker-0", "main"})
		var cliErr *clierror.Error
		if !errors.As(err, &cliErr) {
			t.Fatalf("expected structured CLI error, got %T: %v", err, err)
		}
		if cliErr.Category != errorcodes.CategoryUsage || cliErr.Code != errorcodes.ErrMissingRequiredFlag {
			t.Fatalf("unexpected CLI error: %#v", cliErr)
		}
	})

	t.Run("legacy positional namespace", func(t *testing.T) {
		cmd := newPodOptionsTestCommand(t, "")

		namespace, name, container, err := podNSNameAndContainer(cmd, []string{"legacy-ns", "worker-0", "main"})
		if err != nil {
			t.Fatalf("podNSNameAndContainer: %v", err)
		}
		if namespace != "legacy-ns" || name != "worker-0" || container != "main" {
			t.Fatalf("unexpected target: namespace=%q name=%q container=%q", namespace, name, container)
		}
	})
}

func TestPodNamespaceFlagConflictsWithLegacyArguments(t *testing.T) {
	tests := []struct {
		name string
		args []string
		logs bool
	}{
		{name: "diagnostic command", args: []string{"legacy-ns", "worker-0"}},
		{name: "logs command", args: []string{"legacy-ns", "worker-0", "main"}, logs: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newPodOptionsTestCommand(t, "team-a")
			var err error
			if tt.logs {
				_, _, _, err = podNSNameAndContainer(cmd, tt.args)
			} else {
				_, _, err = podNSAndName(cmd, tt.args)
			}

			var cliErr *clierror.Error
			if !errors.As(err, &cliErr) {
				t.Fatalf("expected structured CLI error, got %T: %v", err, err)
			}
			if cliErr.Category != errorcodes.CategoryUsage || cliErr.Code != errorcodes.ErrInvalidFlagValue {
				t.Fatalf("unexpected CLI error: %#v", cliErr)
			}
		})
	}
}

func TestPodCommandsHaveNoHardcodedNamespaceDefault(t *testing.T) {
	commands := []*cobra.Command{
		podContainersCmd,
		podEventsCmd,
		podLogsCmd,
		podIngressesCmd,
		podNodeportsCmd,
	}

	for _, cmd := range commands {
		namespace, err := cmd.Flags().GetString("namespace")
		if err != nil {
			t.Fatalf("%s namespace flag: %v", cmd.Name(), err)
		}
		if namespace != "" {
			t.Fatalf("%s namespace default = %q, want empty", cmd.Name(), namespace)
		}
	}
}
