package cmd

import (
	"strings"
	"testing"

	"github.com/raids-lab/crater/cli/internal/api"
	"github.com/raids-lab/crater/cli/internal/i18n"
	"github.com/spf13/cobra"
)

func TestFilterAndSortAdminUsers(t *testing.T) {
	users := []api.AdminUser{
		{
			ID: 1, Name: "bob", Role: adminUserRoleUser, Status: adminUserStatusActive,
			Attributes: &api.UserAttribute{Nickname: "Bob"},
		},
		{
			ID: 3, Name: "alice", Role: adminUserRoleAdmin, Status: adminUserStatusActive,
			Attributes: &api.UserAttribute{Nickname: "Alice Zhang"},
		},
		{
			ID: 2, Name: "carol", Role: adminUserRoleAdmin, Status: adminUserStatusInactive,
		},
	}
	filtered := filterAdminUsers(users, "zhang", "Admin", "Active")
	if len(filtered) != 1 || filtered[0].Name != "alice" {
		t.Fatalf("unexpected filtered users: %#v", filtered)
	}

	users = []api.AdminUser{{ID: 1, Name: "one"}, {ID: 3, Name: "three"}, {ID: 2, Name: "two"}}
	sortAdminUsers(users)
	if users[0].ID != 3 || users[1].ID != 2 || users[2].ID != 1 {
		t.Fatalf("unexpected sorted users: %#v", users)
	}
}

func TestAdminUserListAggregatesPaginationAndDomainIssues(t *testing.T) {
	previousLanguage := i18n.GetCurrentLanguage()
	i18n.SetLanguage("en")
	t.Cleanup(func() { i18n.SetLanguage(previousLanguage) })

	cmd := &cobra.Command{}
	cmd.Flags().Bool("base", false, "")
	cmd.Flags().String("search", "", "")
	cmd.Flags().String("role", "", "")
	cmd.Flags().String("status", "", "")
	addListPaginationFlags(cmd)
	_ = cmd.Flags().Set("page", "0")
	_ = cmd.Flags().Set("role", "Owner")
	_ = cmd.Flags().Set("status", "Disabled")

	_, err := readAdminUserListOptions(cmd)
	if err == nil {
		t.Fatal("expected invalid admin user list options to fail")
	}
	for _, want := range []string{"page", "Owner", "Disabled"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q does not contain %q", err, want)
		}
	}
}

func TestAdminUserBaseRejectsRoleAndStatusFilters(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Bool("base", false, "")
	cmd.Flags().String("search", "", "")
	cmd.Flags().String("role", "", "")
	cmd.Flags().String("status", "", "")
	addListPaginationFlags(cmd)
	_ = cmd.Flags().Set("base", "true")
	_ = cmd.Flags().Set("role", "Admin")

	_, err := readAdminUserListOptions(cmd)
	if err == nil || !strings.Contains(err.Error(), "--base") {
		t.Fatalf("unexpected error: %v", err)
	}
}
