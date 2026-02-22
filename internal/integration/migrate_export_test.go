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

	spaceID := strings.TrimSpace(os.Getenv("CFL_IT_SPACE_ID"))
	spaceKey := strings.TrimSpace(os.Getenv("CFL_IT_SPACE_KEY"))
	rootPageID := strings.TrimSpace(os.Getenv("CFL_IT_MIGRATE_ROOT_PAGE_ID"))

	if spaceID == "" && spaceKey == "" {
		t.Skip("set CFL_IT_SPACE_ID or CFL_IT_SPACE_KEY for migrate export integration test")
	}
	if spaceID != "" && spaceKey != "" {
		t.Fatal("CFL_IT_SPACE_ID and CFL_IT_SPACE_KEY are mutually exclusive")
	}

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
	if rootPageID != "" {
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

	markdownCount := 0
	walkErr := filepath.WalkDir(outDir, func(path string, d os.DirEntry, err error) error {
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
	if markdownCount == 0 {
		t.Fatalf("no markdown files exported in %s", outDir)
	}
}
