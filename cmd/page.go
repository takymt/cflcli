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

type pageListOptions struct {
	spaceID string
	limit   int
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
	opts := &pageListOptions{}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List pages",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runPageList(cmd.OutOrStdout(), opts)
		},
	}

	cmd.Flags().StringVar(&opts.spaceID, "space-id", "", "space id")
	cmd.Flags().IntVar(&opts.limit, "limit", 25, "number of results per page")

	return cmd
}

func runPageListWithConfig(out io.Writer, opts *pageListOptions, cfg *config.Config) error {
	if err := validatePageListLimit(opts.limit); err != nil {
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

	result, err := cli.ListPages(opts.spaceID, opts.limit, "")
	if err != nil {
		return err
	}

	switch outputFlag {
	case "table":
		return output.WritePagesTable(out, result.Results)
	case "json":
		return output.WritePageListJSON(out, output.PageListOutput{
			Request: output.PageListRequest{SpaceID: opts.spaceID, Limit: opts.limit},
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

func runPageList(out io.Writer, opts *pageListOptions) error {
	cfg, err := loadConfig("")
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	return runPageListWithConfig(out, opts, cfg)
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
