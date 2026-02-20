package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/takymt/cflcli/internal/client"
	"github.com/takymt/cflcli/internal/config"
	"github.com/takymt/cflcli/internal/output"
)

// RunPageListWithConfig runs the page list command with a provided config.
func RunPageListWithConfig(out io.Writer, opts *PageListOptions, cfg *config.Config) error {
	if err := validatePageListLimit(opts.Limit); err != nil {
		return err
	}

	profile, err := resolveProfile(cfg)
	if err != nil {
		return err
	}

	cli, err := client.New(context.Background(), profile, os.Getenv("CONFLUENCE_API_TOKEN"))
	if err != nil {
		return err
	}

	spaceID, err := resolvePageListSpaceID(opts, profile, cli)
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

	result, err := cli.ListPages(spaceID, opts.Limit, opts.Cursor, statuses, sort)
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
