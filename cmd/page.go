package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
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
	StatusSpecified bool
	Limit           int
}

func newPageCmd() *cobra.Command {
	pageCmd := &cobra.Command{
		Use:   "page",
		Short: "Manage Confluence pages",
	}

	pageCmd.AddCommand(newPageListCmd())

	return pageCmd
}

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
	cmd.Flags().IntVar(&opts.Limit, "limit", 25, "number of results per page")

	originalRunE := cmd.RunE
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		opts.StatusSpecified = cmd.Flags().Changed("status")
		return originalRunE(cmd, args)
	}

	return cmd
}

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

	result, err := cli.ListPages(spaceID, opts.Limit, opts.Cursor, statuses)
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

func runPageList(out io.Writer, opts *PageListOptions) error {
	cfg, err := loadConfig("")
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	return RunPageListWithConfig(out, opts, cfg)
}

func resolveProfile(cfg *config.Config) (*config.Profile, error) {
	if profileFlag != "" {
		profile := cfg.FindProfile(profileFlag)
		if profile == nil {
			return nil, fmt.Errorf("profile %q not found", profileFlag)
		}
		return profile, nil
	}

	profile := cfg.CurrentProfile()
	if profile == nil {
		return nil, fmt.Errorf("no current profile; run 'cfl config init' or 'cfl use <name>'")
	}
	return profile, nil
}

func resolvePageListSpaceID(opts *PageListOptions, profile *config.Profile, cli *client.Client) (string, error) {
	if opts.SpaceID != "" && opts.SpaceKey != "" {
		return "", fmt.Errorf("--space-id and --space-key are mutually exclusive; specify only one")
	}
	if opts.SpaceID != "" {
		return opts.SpaceID, nil
	}

	spaceKey := opts.SpaceKey
	if spaceKey == "" {
		spaceKey = profile.SpaceKey
	}
	if spaceKey == "" {
		return "", fmt.Errorf("space key is required; specify --space-key or configure space_key in profile")
	}

	return cli.ResolveSpaceIDByKey(spaceKey)
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
		statuses = append(statuses, status)
	}
	if len(statuses) == 0 {
		return nil, false, fmt.Errorf("status must not be empty")
	}
	return statuses, true, nil
}
