//go:build integration

package integration

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/takymt/cflcli/cmd"
	"github.com/takymt/cflcli/internal/config"
)

func TestMigrateExportSmoke(t *testing.T) {
	if os.Getenv("CFL_IT_ENABLE_MIGRATE_EXPORT") != "1" {
		t.Skip("set CFL_IT_ENABLE_MIGRATE_EXPORT=1 to run migrate export integration test")
	}

	profile, token := integrationProfile(t)
	spaceID, spaceKey := migrateExportSpaceSelector(t)
	rootPageID := strings.TrimSpace(os.Getenv("CFL_IT_MIGRATE_ROOT_PAGE_ID"))

	outDir, stdout := runMigrateExportCLIIntegration(t, profile, token, spaceID, spaceKey, rootPageID)
	if markdownCount := countMarkdownFiles(t, outDir); markdownCount == 0 {
		t.Fatalf("no markdown files exported in %s", outDir)
	}
	if strings.TrimSpace(stdout) == "" {
		t.Fatalf("command output is empty")
	}
}

func TestMigrateExportAttachment404Regression(t *testing.T) {
	if os.Getenv("CFL_IT_ENABLE_MIGRATE_EXPORT_ATTACHMENT_404_REGRESSION") != "1" {
		t.Skip("set CFL_IT_ENABLE_MIGRATE_EXPORT_ATTACHMENT_404_REGRESSION=1 to run migrate export attachment 404 regression test")
	}

	profile, token := integrationProfile(t)
	spaceID, spaceKey := migrateExportSpaceSelector(t)

	pageID := strings.TrimSpace(os.Getenv("CFL_IT_MIGRATE_404_PAGE_ID"))
	filename := strings.TrimSpace(os.Getenv("CFL_IT_MIGRATE_404_ATTACHMENT_FILENAME"))
	if pageID == "" || filename == "" {
		t.Skip("set CFL_IT_MIGRATE_404_PAGE_ID and CFL_IT_MIGRATE_404_ATTACHMENT_FILENAME for migrate export attachment 404 regression test")
	}

	outDir, stdout := runMigrateExportCLIIntegration(t, profile, token, spaceID, spaceKey, pageID)
	if markdownCount := countMarkdownFiles(t, outDir); markdownCount == 0 {
		t.Fatalf("no markdown files exported in %s", outDir)
	}

	for _, want := range []string{
		"Exported ",
		"Warnings (",
		`download attachment "` + filename + `" for page "` + pageID + `" skipped: 404 Not Found`,
	} {
		if strings.Contains(stdout, want) {
			continue
		}
		t.Fatalf("missing %q in output:\n%s", want, stdout)
	}
}

func migrateExportSpaceSelector(t *testing.T) (string, string) {
	t.Helper()

	spaceID := strings.TrimSpace(os.Getenv("CFL_IT_SPACE_ID"))
	spaceKey := strings.TrimSpace(os.Getenv("CFL_IT_SPACE_KEY"))
	if spaceID == "" && spaceKey == "" {
		t.Skip("set CFL_IT_SPACE_ID or CFL_IT_SPACE_KEY for migrate export integration test")
	}
	if spaceID != "" && spaceKey != "" {
		t.Fatal("CFL_IT_SPACE_ID and CFL_IT_SPACE_KEY are mutually exclusive")
	}
	return spaceID, spaceKey
}

func runMigrateExportCLIIntegration(t *testing.T, profile *config.Profile, token, spaceID, spaceKey, rootPageID string) (string, string) {
	t.Helper()

	t.Setenv("CFL_API_TOKEN", token)

	configRoot := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configRoot)

	cfg := &config.Config{
		Current: "integration",
		Profiles: []config.Profile{
			{
				Name:     "integration",
				Domain:   profile.Domain,
				User:     profile.User,
				SpaceKey: spaceKey,
			},
		},
	}
	configPath, err := config.Path()
	if err != nil {
		t.Fatalf("config.Path: %v", err)
	}
	if err := cfg.SaveTo(configPath); err != nil {
		t.Fatalf("cfg.SaveTo: %v", err)
	}

	outDir := filepath.Join(t.TempDir(), "migrate-export")
	args := []string{"--profile", "integration", "migrate", "export", "--out", outDir}
	if spaceID != "" {
		args = append(args, "--space-id", spaceID)
	}
	if spaceKey != "" {
		args = append(args, "--space-key", spaceKey)
	}
	if rootPageID = strings.TrimSpace(rootPageID); rootPageID != "" {
		args = append(args, "--root-page-id", rootPageID)
	}

	rootCmd := cmd.NewRootCmd()
	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stdout)
	rootCmd.SetArgs(args)

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("cfl migrate export failed: %v\noutput=%s", err, stdout.String())
	}

	return outDir, stdout.String()
}

func countMarkdownFiles(t *testing.T, root string) int {
	t.Helper()

	markdownCount := 0
	walkErr := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.EqualFold(filepath.Ext(path), ".md") {
			markdownCount++
		}
		return nil
	})
	if walkErr != nil {
		t.Fatalf("WalkDir: %v", walkErr)
	}
	return markdownCount
}
