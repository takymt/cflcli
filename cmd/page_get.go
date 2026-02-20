package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/takymt/cflcli/internal/client"
	"github.com/takymt/cflcli/internal/config"
)

// RunPageGetWithConfig runs the page get command with a provided config.
func RunPageGetWithConfig(out io.Writer, pageID string, cfg *config.Config) error {
	profile, err := resolveProfile(cfg)
	if err != nil {
		return err
	}

	cli, err := client.New(context.Background(), profile, os.Getenv("CONFLUENCE_API_TOKEN"))
	if err != nil {
		return err
	}

	page, err := cli.GetPage(pageID)
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
