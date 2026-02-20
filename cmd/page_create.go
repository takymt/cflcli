package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/takymt/cflcli/internal/client"
	"github.com/takymt/cflcli/internal/config"
)

// RunPageCreateWithConfig runs the page create command with a provided config.
func RunPageCreateWithConfig(out io.Writer, opts *pageCreateOptions, cfg *config.Config) error {
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

	spaceID, err := resolvePageSpaceID(opts.SpaceID, opts.SpaceKey, profile, cli)
	if err != nil {
		return err
	}

	created, err := cli.CreatePage(spaceID, title, bodyInput.StorageBody, opts.ParentID)
	if err != nil {
		return err
	}

	switch outputFlag {
	case "table":
		_, err = fmt.Fprintf(out, "Created page %q (id: %q).\n", created.Title, created.ID)
		return err
	case "json":
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(created)
	default:
		return fmt.Errorf("unsupported output format: %s", outputFlag)
	}
}
