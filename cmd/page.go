package cmd

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/takymt/cflcli/internal/client"
	"github.com/takymt/cflcli/internal/config"
	"github.com/takymt/cflcli/internal/output"
)

// PageListOptions holds options for page listing.
type PageListOptions struct {
	SpaceID string
	Limit   int
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
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runPageList(cmd.OutOrStdout(), opts)
		},
	}

	cmd.Flags().StringVar(&opts.SpaceID, "space-id", "", "space id")
	cmd.Flags().IntVar(&opts.Limit, "limit", 25, "number of results per page")

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

	result, err := cli.ListPages(opts.SpaceID, opts.Limit, "")
	if err != nil {
		return err
	}

	switch outputFlag {
	case "table":
		return output.WritePagesTable(out, result.Results)
	case "json":
		return output.WritePageListJSON(out, output.PageListOutput{
			Request: output.PageListRequest{SpaceID: opts.SpaceID, Limit: opts.Limit},
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
