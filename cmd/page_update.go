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

// RunPageUpdateWithConfig runs the page update command with a provided config.
func RunPageUpdateWithConfig(out io.Writer, opts *pageUpdateOptions, cfg *config.Config) error {
	bodyFile := strings.TrimSpace(opts.BodyFile)
	if bodyFile == "" {
		return fmt.Errorf("--body-file is required")
	}
	bodyInput, err := loadPageStorageBody(bodyFile, opts.BodyFormat)
	if err != nil {
		return err
	}
	title := resolvePageTitle(opts.Title, bodyInput.FrontMatterTitle)
	if err := validatePageTitleSources(opts.Title, bodyInput.FrontMatterTitle); err != nil {
		return err
	}
	if title == "" {
		return fmt.Errorf("--title is required")
	}

	profile, err := resolveProfile(cfg)
	if err != nil {
		return err
	}

	cli, err := client.New(context.Background(), profile, os.Getenv("CONFLUENCE_API_TOKEN"))
	if err != nil {
		return err
	}

	current, err := cli.GetPage(opts.PageID)
	if err != nil {
		var httpErr *client.HTTPError
		if errors.As(err, &httpErr) && httpErr.StatusCode == 404 {
			return fmt.Errorf("page %q not found", opts.PageID)
		}
		return err
	}
	if current.Version.Number < 1 {
		return fmt.Errorf("page %q has invalid current version", opts.PageID)
	}

	updated, err := cli.UpdatePage(opts.PageID, title, bodyInput.StorageBody, opts.ParentID, current.Version.Number+1)
	if err != nil {
		var httpErr *client.HTTPError
		if errors.As(err, &httpErr) &&
			(httpErr.StatusCode == 409 || (httpErr.StatusCode == 400 && strings.Contains(strings.ToLower(httpErr.Body), "version"))) {
			return fmt.Errorf("update conflict: page %q was modified by another user; fetch latest and retry", opts.PageID)
		}
		return err
	}

	switch outputFlag {
	case "table":
		id := updated.ID
		if strings.TrimSpace(id) == "" {
			id = opts.PageID
		}
		_, err = fmt.Fprintf(out, "Updated page %q (id: %q).\n", updated.Title, id)
		return err
	case "json":
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(updated)
	default:
		return fmt.Errorf("unsupported output format: %s", outputFlag)
	}
}
