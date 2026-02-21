package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverRepoConfig_FromFilePathFindsNearestParent(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	topCfgPath := filepath.Join(root, "cfl.toml")
	if err := os.WriteFile(topCfgPath, []byte(`domain = "top.atlassian.net"`), 0o600); err != nil {
		t.Fatalf("WriteFile(top): %v", err)
	}

	nested := filepath.Join(root, "docs")
	nestedCfgPath := filepath.Join(nested, "cfl.toml")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("MkdirAll(nested): %v", err)
	}
	if err := os.WriteFile(nestedCfgPath, []byte(`domain = "nested.atlassian.net"`), 0o600); err != nil {
		t.Fatalf("WriteFile(nested): %v", err)
	}

	bodyFile := filepath.Join(nested, "pages", "a.md")
	if err := os.MkdirAll(filepath.Dir(bodyFile), 0o755); err != nil {
		t.Fatalf("MkdirAll(body dir): %v", err)
	}
	if err := os.WriteFile(bodyFile, []byte("# doc"), 0o600); err != nil {
		t.Fatalf("WriteFile(body): %v", err)
	}

	cfg, path, err := DiscoverRepoConfig(bodyFile)
	if err != nil {
		t.Fatalf("DiscoverRepoConfig: %v", err)
	}
	if cfg == nil {
		t.Fatalf("expected repo config")
	}
	assertSameFilePath(t, path, nestedCfgPath)
	if cfg.Domain != "nested.atlassian.net" {
		t.Fatalf("domain=%q", cfg.Domain)
	}
}

func TestDiscoverRepoConfig_FromCWD(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	cfgPath := filepath.Join(root, "cfl.toml")
	if err := os.WriteFile(cfgPath, []byte("space_id = \"SPACE-1\"\ncontent_root = \"docs\""), 0o600); err != nil {
		t.Fatalf("WriteFile(cfl.toml): %v", err)
	}
	workDir := filepath.Join(root, "sub", "dir")
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(workDir): %v", err)
	}

	oldCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(workDir); err != nil {
		t.Fatalf("Chdir(workDir): %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldCwd)
	})

	cfg, path, err := DiscoverRepoConfig("")
	if err != nil {
		t.Fatalf("DiscoverRepoConfig: %v", err)
	}
	if cfg == nil {
		t.Fatalf("expected repo config")
	}
	assertSameFilePath(t, path, cfgPath)
	if cfg.SpaceID != "SPACE-1" || cfg.ContentRoot != "docs" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestDiscoverRepoConfig_NotFound(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	cfg, path, err := DiscoverRepoConfig(root)
	if err != nil {
		t.Fatalf("DiscoverRepoConfig: %v", err)
	}
	if cfg != nil || path != "" {
		t.Fatalf("expected not found, got cfg=%+v path=%q", cfg, path)
	}
}

func assertSameFilePath(t *testing.T, got, want string) {
	t.Helper()

	gotInfo, err := os.Stat(got)
	if err != nil {
		t.Fatalf("Stat(got): %v", err)
	}
	wantInfo, err := os.Stat(want)
	if err != nil {
		t.Fatalf("Stat(want): %v", err)
	}
	if !os.SameFile(gotInfo, wantInfo) {
		t.Fatalf("path=%q want %q", got, want)
	}
}
