package e2e

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestInitCommand(t *testing.T) {
	t.Run("creates default profile on first run", func(t *testing.T) {
		xdgConfigHome := t.TempDir()

		out, err := runCLI(t, xdgConfigHome, "default.atlassian.net\ndefault@example.com\nDEF\njson\n", "init")
		if err != nil {
			t.Fatalf("unexpected error: %v\n%s", err, out)
		}
		if !strings.Contains(out, "Initialization completed.") {
			t.Fatalf("expected initialization message, got: %s", out)
		}

		cfg := loadConfig(t, xdgConfigHome)
		if cfg.Current != "default" {
			t.Fatalf("expected current %q, got %q", "default", cfg.Current)
		}
		p := cfg.FindProfile("default")
		if p == nil {
			t.Fatal("expected default profile to exist")
		}
		if p.Domain != "default.atlassian.net" || p.User != "default@example.com" || p.SpaceKey != "DEF" || p.Output != "json" {
			t.Fatalf("unexpected default profile: %+v", *p)
		}
	})

	t.Run("keeps existing config unchanged when already initialized", func(t *testing.T) {
		xdgConfigHome := t.TempDir()
		createProfile(t, xdgConfigHome, "default", "default.atlassian.net", "default@example.com", "DEF", "table")

		before, err := os.ReadFile(configPath(xdgConfigHome))
		if err != nil {
			t.Fatalf("read before config failed: %v", err)
		}

		out, err := runCLI(t, xdgConfigHome, "", "init")
		if err != nil {
			t.Fatalf("unexpected error: %v\n%s", err, out)
		}
		if !strings.Contains(out, "already initialized; no changes made.") {
			t.Fatalf("expected already initialized message, got: %s", out)
		}

		after, err := os.ReadFile(configPath(xdgConfigHome))
		if err != nil {
			t.Fatalf("read after config failed: %v", err)
		}
		if !bytes.Equal(before, after) {
			t.Fatal("expected config file to remain unchanged")
		}
	})
}
