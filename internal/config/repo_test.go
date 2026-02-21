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

	cfg, _, err := DiscoverRepoConfig(bodyFile)
	if err != nil {
		t.Fatalf("DiscoverRepoConfig: %v", err)
	}
	if cfg == nil {
		t.Fatalf("expected repo config")
	}
	if cfg.Domain != "nested.atlassian.net" {
		t.Fatalf("domain=%q", cfg.Domain)
	}
}
