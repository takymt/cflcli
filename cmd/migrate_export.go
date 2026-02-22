package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	"github.com/takymt/cflcli/internal/config"
	migratepkg "github.com/takymt/cflcli/internal/migrate"
)

type migrateExportOptions struct {
	SpaceID        string
	SpaceKey       string
	RootPageID     string
	Out            string
	AttachmentsDir string
}

type migrateExportResult struct {
	SpaceID        string                `json:"space_id"`
	SpaceKey       string                `json:"space_key"`
	Out            string                `json:"out"`
	AttachmentsDir string                `json:"attachments_dir"`
	Pages          []migrateExportedPage `json:"pages"`
	Warnings       []string              `json:"warnings,omitempty"`
}

type migrateExportedPage struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	ParentID string `json:"parent_id,omitempty"`
	File     string `json:"file"`
}

const defaultMigrateAttachmentsDir = migratepkg.DefaultAttachmentsDir

func newMigrateExportCmd() *cobra.Command {
	opts := &migrateExportOptions{}

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export Confluence pages as markdown",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runMigrateExport(cmd.OutOrStdout(), opts)
		},
	}

	cmd.Flags().StringVar(&opts.SpaceID, "space-id", "", "space id (numeric)")
	cmd.Flags().StringVar(&opts.SpaceKey, "space-key", "", "space key (mutually exclusive with --space-id)")
	cmd.Flags().StringVar(&opts.RootPageID, "root-page-id", "", "root page id to export as subtree")
	cmd.Flags().StringVar(&opts.Out, "out", "", "output directory (required)")
	cmd.Flags().StringVar(&opts.AttachmentsDir, "attachments-dir", defaultMigrateAttachmentsDir, "attachments directory under --out")

	return cmd
}

func runMigrateExport(out io.Writer, opts *migrateExportOptions) error {
	cfg, err := loadConfig("")
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	return RunMigrateExportWithConfig(out, opts, cfg)
}

// RunMigrateExportWithConfig runs migrate export with a provided config.
func RunMigrateExportWithConfig(out io.Writer, opts *migrateExportOptions, cfg *config.Config) error {
	if opts == nil {
		return fmt.Errorf("options are required")
	}
	outDir := strings.TrimSpace(opts.Out)
	if outDir == "" {
		return fmt.Errorf("--out is required")
	}

	runtime, err := newPageRuntime(cfg)
	if err != nil {
		return err
	}

	spaceID, err := runtime.resolveSpaceID(opts.SpaceID, opts.SpaceKey)
	if err != nil {
		return err
	}
	spaceKey, err := resolveMigrateExportSpaceKey(runtime, opts, spaceID)
	if err != nil {
		return err
	}

	exportResult, err := migratepkg.Export(runtime.Client, &migratepkg.ExportRequest{
		SpaceID:        spaceID,
		SpaceKey:       spaceKey,
		RootPageID:     strings.TrimSpace(opts.RootPageID),
		OutDir:         outDir,
		AttachmentsDir: strings.TrimSpace(opts.AttachmentsDir),
	})
	if err != nil {
		return err
	}

	result := migrateExportResult{
		SpaceID:        exportResult.SpaceID,
		SpaceKey:       exportResult.SpaceKey,
		Out:            exportResult.OutDir,
		AttachmentsDir: exportResult.AttachmentsDir,
		Pages:          make([]migrateExportedPage, 0, len(exportResult.Pages)),
		Warnings:       make([]string, 0, len(exportResult.Warnings)),
	}
	for _, page := range exportResult.Pages {
		result.Pages = append(result.Pages, migrateExportedPage{
			ID:       page.ID,
			Title:    page.Title,
			ParentID: page.ParentID,
			File:     page.File,
		})
	}
	for _, warning := range exportResult.Warnings {
		if msg := strings.TrimSpace(warning.Message); msg != "" {
			result.Warnings = append(result.Warnings, msg)
		}
	}

	switch outputFlag {
	case "table":
		if _, err := fmt.Fprintf(out, "Exported %d pages to %q.\n", len(result.Pages), result.Out); err != nil {
			return err
		}
		if len(result.Warnings) == 0 {
			return nil
		}
		if _, err := fmt.Fprintf(out, "Warnings (%d):\n", len(result.Warnings)); err != nil {
			return err
		}
		for _, warning := range result.Warnings {
			if _, err := fmt.Fprintf(out, "- %s\n", warning); err != nil {
				return err
			}
		}
		return nil
	case "json":
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	default:
		return fmt.Errorf("unsupported output format: %s", outputFlag)
	}
}

func resolveMigrateExportSpaceKey(runtime *pageRuntime, opts *migrateExportOptions, resolvedSpaceID string) (string, error) {
	if explicit := strings.TrimSpace(opts.SpaceKey); explicit != "" {
		return explicit, nil
	}
	if profileSpaceKey := strings.TrimSpace(runtime.Profile.SpaceKey); profileSpaceKey != "" && strings.TrimSpace(opts.SpaceID) == "" {
		return profileSpaceKey, nil
	}
	return runtime.Client.ResolveSpaceKeyByID(resolvedSpaceID)
}
