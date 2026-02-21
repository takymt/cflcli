package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/takymt/cflcli/internal/attachment"
	"github.com/takymt/cflcli/internal/body"
	"github.com/takymt/cflcli/internal/client"
	"github.com/takymt/cflcli/internal/config"
)

type pageUpdateOptions struct {
	PageID     string
	Title      string
	BodyFile   string
	BodyFormat string
	AssetsRoot string
	ParentID   string
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
	cmd.Flags().StringVar(&opts.AssetsRoot, "assets-root", "", "base directory for /-prefixed markdown asset paths (default: body file directory)")
	cmd.Flags().StringVar(&opts.ParentID, "parent-id", "", "parent page id")

	return cmd
}

func runPageUpdate(out io.Writer, opts *pageUpdateOptions) error {
	cfg, err := loadConfig("")
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	return RunPageUpdateWithConfig(out, opts, cfg)
}

// RunPageUpdateWithConfig runs the page update command with a provided config.
func RunPageUpdateWithConfig(out io.Writer, opts *pageUpdateOptions, cfg *config.Config) error {
	bodyFile := strings.TrimSpace(opts.BodyFile)
	if bodyFile == "" {
		return fmt.Errorf("--body-file is required")
	}

	repoCfg, repoConfigPath, err := discoverRepoConfig(bodyFile)
	if err != nil {
		return err
	}
	assetsRoot, err := resolveAssetsRootFlagDefault(opts.AssetsRoot, repoCfg, repoConfigPath)
	if err != nil {
		return err
	}
	bodyInput, err := loadPageStorageBody(bodyFile, opts.BodyFormat, assetsRoot)
	if err != nil {
		return err
	}
	title := resolvePageTitle(opts.Title, bodyInput.FrontMatterTitle)
	if err := validatePageTitleSources(opts.Title, bodyInput.FrontMatterTitle); err != nil {
		return err
	}
	parentID := resolvePageParentID(opts.ParentID, bodyInput.FrontMatterParentID)
	if err := validatePageParentIDSources(opts.ParentID, bodyInput.FrontMatterParentID); err != nil {
		return err
	}
	if title == "" {
		return fmt.Errorf("--title is required")
	}

	profile, err := resolveProfile(cfg)
	if err != nil {
		return err
	}
	profile = applyRepoDomain(profile, repoCfg)

	cli, err := client.New(context.Background(), profile, os.Getenv("CFL_API_TOKEN"))
	if err != nil {
		return err
	}
	if err := attachment.UploadPageAssets(cli, opts.PageID, bodyInput.LocalImageAssets); err != nil {
		return fmt.Errorf("upload local image assets for page %q: %w", opts.PageID, err)
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

	updated, err := cli.UpdatePage(opts.PageID, title, bodyInput.StorageBody, parentID, current.Version.Number+1)
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
