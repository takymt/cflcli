package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	"github.com/takymt/cflcli/internal/client"
	"github.com/takymt/cflcli/internal/config"
)

type pageDeleteOptions struct {
	PageID string
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

func runPageDelete(out io.Writer, opts *pageDeleteOptions) error {
	cfg, err := loadConfig("")
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	return RunPageDeleteWithConfig(out, opts.PageID, cfg)
}

// RunPageDeleteWithConfig runs the page delete command with a provided config.
func RunPageDeleteWithConfig(out io.Writer, pageID string, cfg *config.Config) error {
	pageID = strings.TrimSpace(pageID)
	if pageID == "" {
		return fmt.Errorf("<page-id> is required")
	}

	runtime, err := newPageRuntime(cfg, "")
	if err != nil {
		return err
	}

	if err := runtime.Client.DeletePage(pageID); err != nil {
		var httpErr *client.HTTPError
		if errors.As(err, &httpErr) && httpErr.StatusCode == 404 {
			return fmt.Errorf("page %q not found", pageID)
		}
		return err
	}

	switch outputFlag {
	case "table":
		_, err = fmt.Fprintf(out, "Deleted page %q.\n", pageID)
		return err
	case "json":
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(struct {
			ID      string `json:"id"`
			Deleted bool   `json:"deleted"`
		}{
			ID:      pageID,
			Deleted: true,
		})
	default:
		return fmt.Errorf("unsupported output format: %s", outputFlag)
	}
}
