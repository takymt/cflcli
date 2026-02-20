package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/takymt/cflcli/internal/client"
	"github.com/takymt/cflcli/internal/config"
)

// RunPageDeleteWithConfig runs the page delete command with a provided config.
func RunPageDeleteWithConfig(out io.Writer, pageID string, cfg *config.Config) error {
	pageID = strings.TrimSpace(pageID)
	if pageID == "" {
		return fmt.Errorf("<page-id> is required")
	}

	profile, err := resolveProfile(cfg)
	if err != nil {
		return err
	}

	cli, err := client.New(context.Background(), profile, os.Getenv("CONFLUENCE_API_TOKEN"))
	if err != nil {
		return err
	}

	if err := cli.DeletePage(pageID); err != nil {
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
