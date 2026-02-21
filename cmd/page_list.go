package cmd

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	"github.com/takymt/cflcli/internal/client"
	"github.com/takymt/cflcli/internal/config"
	"github.com/takymt/cflcli/internal/output"
)

// PageListOptions holds options for page listing.
type PageListOptions struct {
	SpaceID         string
	SpaceKey        string
	Cursor          string
	Status          string
	Sort            string
	StatusSpecified bool
	Limit           int
}

var pageListAllowedStatuses = map[string]struct{}{
	"current":  {},
	"archived": {},
	"deleted":  {},
	"trashed":  {},
}

var pageListAllowedSorts = map[string]struct{}{
	"id":             {},
	"-id":            {},
	"created-date":   {},
	"-created-date":  {},
	"modified-date":  {},
	"-modified-date": {},
	"title":          {},
	"-title":         {},
}

const pageListAllowedSortValues = "id, -id, created-date, -created-date, modified-date, -modified-date, title, -title"

func newPageListCmd() *cobra.Command {
	opts := &PageListOptions{}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List pages",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runPageList(cmd.OutOrStdout(), opts)
		},
	}

	cmd.Flags().StringVar(&opts.SpaceID, "space-id", "", "space id (numeric)")
	cmd.Flags().StringVar(&opts.SpaceKey, "space-key", "", "space key (mutually exclusive with --space-id)")
	cmd.Flags().StringVar(&opts.Cursor, "cursor", "", "pagination cursor")
	cmd.Flags().StringVar(&opts.Status, "status", "", "page status filter (comma-separated)")
	cmd.Flags().StringVar(&opts.Sort, "sort", "", "sort order")
	cmd.Flags().IntVar(&opts.Limit, "limit", 25, "number of results per page")
	cmd.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		if strings.Contains(err.Error(), "flag needs an argument: --sort") {
			return fmt.Errorf("flag needs an argument: --sort; allowed values: %s", pageListAllowedSortValues)
		}
		return err
	})

	originalRunE := cmd.RunE
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		opts.StatusSpecified = cmd.Flags().Changed("status")
		return originalRunE(cmd, args)
	}

	return cmd
}

func runPageList(out io.Writer, opts *PageListOptions) error {
	cfg, err := loadConfig("")
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	return RunPageListWithConfig(out, opts, cfg)
}

// RunPageListWithConfig runs the page list command with a provided config.
func RunPageListWithConfig(out io.Writer, opts *PageListOptions, cfg *config.Config) error {
	if err := validatePageListLimit(opts.Limit); err != nil {
		return err
	}

	runtime, err := newPageRuntime(cfg, "")
	if err != nil {
		return err
	}

	spaceID, err := runtime.resolveSpaceID(opts.SpaceID, opts.SpaceKey)
	if err != nil {
		return err
	}

	statuses, showStatus, err := resolvePageListStatuses(opts)
	if err != nil {
		return err
	}

	sort, err := resolvePageListSort(opts)
	if err != nil {
		return err
	}

	result, err := runtime.Client.ListPages(spaceID, opts.Limit, opts.Cursor, statuses, sort)
	if err != nil {
		var httpErr *client.HTTPError
		if opts.Cursor != "" && errors.As(err, &httpErr) && httpErr.StatusCode >= 400 && httpErr.StatusCode < 500 {
			return fmt.Errorf("cursor %q is invalid or expired; rerun without --cursor to get a new cursor: %w", opts.Cursor, err)
		}
		return err
	}

	switch outputFlag {
	case "table":
		return output.WritePagesTable(out, result.Results, showStatus)
	case "json":
		return output.WritePageListJSON(out, output.PageListOutput{
			Request: output.PageListRequest{SpaceID: spaceID, Limit: opts.Limit},
			Next:    result.Links.Next,
			Results: result.Results,
		})
	default:
		return fmt.Errorf("unsupported output format: %s", outputFlag)
	}
}

func validatePageListLimit(limit int) error {
	if limit < 1 || limit > 250 {
		return fmt.Errorf("limit must be between 1 and 250")
	}
	return nil
}

func resolvePageListStatuses(opts *PageListOptions) ([]string, bool, error) {
	if !opts.StatusSpecified {
		return []string{"current"}, false, nil
	}

	var statuses []string
	for _, raw := range strings.Split(opts.Status, ",") {
		status := strings.TrimSpace(raw)
		if status == "" {
			return nil, false, fmt.Errorf("status must not be empty")
		}
		if _, ok := pageListAllowedStatuses[status]; !ok {
			return nil, false, fmt.Errorf("invalid status %q; allowed values: current, archived, deleted, trashed", status)
		}
		statuses = append(statuses, status)
	}
	if len(statuses) == 0 {
		return nil, false, fmt.Errorf("status must not be empty")
	}
	return statuses, true, nil
}

func resolvePageListSort(opts *PageListOptions) (string, error) {
	sort := strings.TrimSpace(opts.Sort)
	if sort == "" {
		return "", nil
	}
	if _, ok := pageListAllowedSorts[sort]; !ok {
		return "", fmt.Errorf("invalid sort %q; allowed values: %s", sort, pageListAllowedSortValues)
	}
	return sort, nil
}
