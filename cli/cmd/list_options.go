package cmd

import (
	"fmt"
	"os"

	"github.com/raids-lab/crater/cli/internal/api"
	"github.com/raids-lab/crater/cli/internal/i18n"
	"github.com/raids-lab/crater/cli/internal/output"
	"github.com/raids-lab/crater/cli/pkg/errorcodes"
	"github.com/spf13/cobra"
)

const maxCLIPageSize = 200

func addListPaginationFlags(cmd *cobra.Command) {
	cmd.Flags().Int("page", 1, i18n.T("flag_page"))
	cmd.Flags().Int("page-size", api.DefaultCLIPageSize, i18n.T("flag_page-size"))
	cmd.Flags().Bool("all-pages", false, i18n.T("flag_all-pages"))
}

func addJobListFlags(cmd *cobra.Command) {
	addListPaginationFlags(cmd)
	cmd.Flags().String("sort", "", i18n.T("flag_sort"))
}

func listPaginationOptions(
	cmd *cobra.Command,
	maxPageSize int,
) (api.ListOptions, []usageIssue) {
	page, _ := cmd.Flags().GetInt("page")
	pageSize, _ := cmd.Flags().GetInt("page-size")
	allPages, _ := cmd.Flags().GetBool("all-pages")
	if allPages && !cmd.Flags().Changed("page-size") {
		pageSize = maxPageSize
	}
	issues := make([]usageIssue, 0, 2)
	if page < 1 {
		issues = append(issues, invalidIssue("page", i18n.T("err_page_min")))
	}
	if pageSize < 1 || pageSize > maxPageSize {
		issues = append(issues, invalidIssue(
			"page-size",
			i18n.T("err_page_size_range", maxPageSize),
		))
	}
	if !allPages && page > int(^uint(0)>>1)/max(pageSize, 1) {
		issues = append(issues, usageIssue{
			Code:    errorcodes.ErrInvalidFlagValue,
			Message: i18n.T("err_page_overflow"),
			Field:   "page",
		})
	}
	return api.ListOptions{
		Page:     page,
		PageSize: pageSize,
		AllPages: allPages,
	}, issues
}

func readListPaginationOptions(cmd *cobra.Command) (api.ListOptions, error) {
	return readListPaginationOptionsWithMax(cmd, maxCLIPageSize)
}

func readListPaginationOptionsWithMax(
	cmd *cobra.Command,
	maxPageSize int,
) (api.ListOptions, error) {
	options, issues := listPaginationOptions(cmd, maxPageSize)
	if len(issues) > 0 {
		return api.ListOptions{}, errUsageFromIssues(issues)
	}
	return options, nil
}

func paginateList[T any](items []T, options api.ListOptions) api.Page[T] {
	options = options.Normalize()
	total := int64(len(items))
	if options.AllPages {
		if items == nil {
			items = []T{}
		}
		return api.Page[T]{
			Items:    items,
			Total:    total,
			Page:     1,
			PageSize: options.PageSize,
		}
	}

	if options.Page > int(^uint(0)>>1)/options.PageSize {
		return emptyListPage[T](total, options)
	}
	start := (options.Page - 1) * options.PageSize
	if start >= len(items) {
		return emptyListPage[T](total, options)
	}
	end := min(start+options.PageSize, len(items))
	return api.Page[T]{
		Items:    items[start:end],
		Total:    total,
		Page:     options.Page,
		PageSize: options.PageSize,
	}
}

func emptyListPage[T any](total int64, options api.ListOptions) api.Page[T] {
	return api.Page[T]{
		Items:    []T{},
		Total:    total,
		Page:     options.Page,
		PageSize: options.PageSize,
	}
}

func listPagePayload[T any](key string, page api.Page[T], allPages bool) map[string]interface{} {
	if page.Items == nil {
		page.Items = []T{}
	}
	payload := map[string]interface{}{key: page.Items}
	if !allPages {
		payload["pagination"] = map[string]interface{}{
			"page":      page.Page,
			"page_size": page.PageSize,
			"total":     page.Total,
		}
	}
	return payload
}

func writeListPage[T any](
	payloadKey string,
	page api.Page[T],
	allPages bool,
	table func([]T),
) error {
	if outputJSON || table == nil {
		return output.WriteSuccessJSON(
			os.Stdout,
			output.SuccessEnvelope(listPagePayload(payloadKey, page, allPages)),
		)
	}
	table(page.Items)
	printListPageSummary(page, allPages)
	return nil
}

func printListPageSummary[T any](page api.Page[T], allPages bool) {
	if !allPages {
		fmt.Println(i18n.T("list_page_summary", page.Page, page.Total))
	}
}
