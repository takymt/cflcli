package cmd

import (
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/takymt/cflcli/internal/client"
	"github.com/takymt/cflcli/internal/config"
)

type pageGetOptions struct {
	PageID string
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

func runPageGet(out io.Writer, opts *pageGetOptions) error {
	cfg, err := loadConfig("")
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	return RunPageGetWithConfig(out, opts.PageID, cfg)
}

// RunPageGetWithConfig runs the page get command with a provided config.
func RunPageGetWithConfig(out io.Writer, pageID string, cfg *config.Config) error {
	runtime, err := newPageRuntime(cfg)
	if err != nil {
		return err
	}

	page, err := runtime.Client.GetPage(pageID)
	if err != nil {
		var httpErr *client.HTTPError
		if errors.As(err, &httpErr) && httpErr.StatusCode == 404 {
			return fmt.Errorf("page %q not found", pageID)
		}
		return err
	}

	_, err = io.WriteString(out, page.Body.Storage.Value)
	return err
}
