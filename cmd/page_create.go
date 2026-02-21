package cmd

import (
	"context"
	"encoding/json"
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

type pageCreateOptions struct {
	Title      string
	BodyFile   string
	BodyFormat string
	AssetsRoot string
	ParentID   string
	SpaceID    string
	SpaceKey   string
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
	cmd.Flags().StringVar(&opts.AssetsRoot, "assets-root", "", "base directory for /-prefixed markdown asset paths (default: body file directory)")
	cmd.Flags().StringVar(&opts.ParentID, "parent-id", "", "parent page id")
	cmd.Flags().StringVar(&opts.SpaceID, "space-id", "", "space id (numeric)")
	cmd.Flags().StringVar(&opts.SpaceKey, "space-key", "", "space key (mutually exclusive with --space-id)")

	return cmd
}

func runPageCreate(out io.Writer, opts *pageCreateOptions) error {
	cfg, err := loadConfig("")
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	return RunPageCreateWithConfig(out, opts, cfg)
}

// RunPageCreateWithConfig runs the page create command with a provided config.
func RunPageCreateWithConfig(out io.Writer, opts *pageCreateOptions, cfg *config.Config) error {
	bodyFile := strings.TrimSpace(opts.BodyFile)
	if bodyFile == "" {
		return fmt.Errorf("--body-file is required")
	}

	repoCfg, repoConfigPath, err := discoverRepoConfig(bodyFile)
	if err != nil {
		return err
	}
	assetsRoot := resolveAssetsRootFlagDefault(opts.AssetsRoot, repoCfg, repoConfigPath)
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

	spaceID, err := resolvePageSpaceIDWithRepoDefaults(opts.SpaceID, opts.SpaceKey, repoCfg, profile, cli)
	if err != nil {
		return err
	}

	created, err := cli.CreatePage(spaceID, title, bodyInput.StorageBody, parentID)
	if err != nil {
		return err
	}
	if err := attachment.UploadPageAssets(cli, created.ID, bodyInput.LocalImageAssets); err != nil {
		return fmt.Errorf("upload local image assets for page %q: %w", created.ID, err)
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
