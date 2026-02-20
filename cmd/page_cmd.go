package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/takymt/cflcli/internal/body"
)

func newPageCmd() *cobra.Command {
	pageCmd := &cobra.Command{
		Use:   "page",
		Short: "Manage Confluence pages",
	}

	pageCmd.AddCommand(newPageListCmd())
	pageCmd.AddCommand(newPageGetCmd())
	pageCmd.AddCommand(newPageCreateCmd())
	pageCmd.AddCommand(newPageUpdateCmd())
	pageCmd.AddCommand(newPageDeleteCmd())

	return pageCmd
}

func newPageGetCmd() *cobra.Command {
	opts := &pageGetOptions{}

	cmd := &cobra.Command{
		Use:   "get <page-id>",
		Short: "Get page body in storage format",
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("<page-id> is required\nUsage: cfl page get <page-id>")
			}
			if len(args) > 1 {
				return fmt.Errorf("too many arguments\nUsage: cfl page get <page-id>")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.PageID = args[0]
			return runPageGet(cmd.OutOrStdout(), opts)
		},
	}

	return cmd
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

func newPageCreateCmd() *cobra.Command {
	opts := &pageCreateOptions{}

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create page",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runPageCreate(cmd.OutOrStdout(), opts)
		},
	}

	cmd.Flags().StringVar(&opts.Title, "title", "", "page title")
	cmd.Flags().StringVar(&opts.BodyFile, "body-file", "", "path to storage format body file")
	cmd.Flags().StringVar(&opts.BodyFormat, "body-format", body.FormatMarkdown, "body file format (markdown | storage)")
	cmd.Flags().StringVar(&opts.ParentID, "parent-id", "", "parent page id")
	cmd.Flags().StringVar(&opts.SpaceID, "space-id", "", "space id (numeric)")
	cmd.Flags().StringVar(&opts.SpaceKey, "space-key", "", "space key (mutually exclusive with --space-id)")

	return cmd
}

func newPageUpdateCmd() *cobra.Command {
	opts := &pageUpdateOptions{}

	cmd := &cobra.Command{
		Use:   "update <page-id>",
		Short: "Update page",
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("<page-id> is required\nUsage: cfl page update <page-id>")
			}
			if len(args) > 1 {
				return fmt.Errorf("too many arguments\nUsage: cfl page update <page-id>")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.PageID = args[0]
			return runPageUpdate(cmd.OutOrStdout(), opts)
		},
	}

	cmd.Flags().StringVar(&opts.Title, "title", "", "page title")
	cmd.Flags().StringVar(&opts.BodyFile, "body-file", "", "path to storage format body file")
	cmd.Flags().StringVar(&opts.BodyFormat, "body-format", body.FormatMarkdown, "body file format (markdown | storage)")
	cmd.Flags().StringVar(&opts.ParentID, "parent-id", "", "parent page id")

	return cmd
}

func newPageDeleteCmd() *cobra.Command {
	opts := &pageDeleteOptions{}

	cmd := &cobra.Command{
		Use:   "delete <page-id>",
		Short: "Delete page",
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("<page-id> is required\nUsage: cfl page delete <page-id>")
			}
			if len(args) > 1 {
				return fmt.Errorf("too many arguments\nUsage: cfl page delete <page-id>")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.PageID = args[0]
			return runPageDelete(cmd.OutOrStdout(), opts)
		},
	}

	return cmd
}
