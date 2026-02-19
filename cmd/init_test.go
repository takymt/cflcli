package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/takymt/cflcli/internal/config"
)

func TestInit_Uninitialized(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")

	opts := &initOptions{configPath: configPath}
	in := strings.NewReader("example.atlassian.net\nuser@example.com\nDEV\njson\n")
	out := &bytes.Buffer{}

	if err := runInit(in, out, opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cfg, err := config.LoadFrom(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	if cfg.Current != "default" {
		t.Fatalf("expected current profile %q, got %q", "default", cfg.Current)
	}
	if len(cfg.Profiles) != 1 {
		t.Fatalf("expected 1 profile, got %d", len(cfg.Profiles))
	}

	p := cfg.Profiles[0]
	if p.Name != "default" {
		t.Fatalf("expected profile name %q, got %q", "default", p.Name)
	}
	if p.Domain != "example.atlassian.net" {
		t.Fatalf("expected domain %q, got %q", "example.atlassian.net", p.Domain)
	}
	if p.User != "user@example.com" {
		t.Fatalf("expected user %q, got %q", "user@example.com", p.User)
	}
	if p.SpaceKey != "DEV" {
		t.Fatalf("expected space key %q, got %q", "DEV", p.SpaceKey)
	}
	if p.Output != "json" {
		t.Fatalf("expected output %q, got %q", "json", p.Output)
	}

	if !strings.Contains(out.String(), "Initialization completed.") {
		t.Fatalf("unexpected output: %s", out.String())
	}
}

func TestInit_AlreadyInitialized(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")

	cfg := &config.Config{
		Current: "work",
		Profiles: []config.Profile{
			{Name: "work", Domain: "work.atlassian.net", User: "work@example.com", SpaceKey: "WORK", Output: "table"},
		},
	}
	if err := cfg.SaveTo(configPath); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}
	before, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}

	opts := &initOptions{configPath: configPath}
	out := &bytes.Buffer{}
	if err := runInit(strings.NewReader(""), out, opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	after, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("expected config file to remain unchanged")
	}
	if !strings.Contains(out.String(), "already initialized") {
		t.Fatalf("unexpected output: %s", out.String())
	}
}
