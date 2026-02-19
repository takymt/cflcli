package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/takymt/cflcli/internal/config"
)

func TestRunUseInteractive(t *testing.T) {
	configPath := createUseTestConfig(t, &config.Config{
		Current: "work",
		Profiles: []config.Profile{
			{Name: "work", Domain: "example.atlassian.net", User: "work@example.com"},
			{Name: "dev", Domain: "dev.atlassian.net", User: "dev@example.com"},
		},
	})

	var out bytes.Buffer
	err := runUseInteractive(strings.NewReader("2\n"), &out, &useOptions{configPath: configPath})
	if err != nil {
		t.Fatalf("runUseInteractive: %v", err)
	}
	if !strings.Contains(out.String(), `Switched to profile "dev".`) {
		t.Fatalf("unexpected output: %q", out.String())
	}

	cfg, err := config.LoadFrom(configPath)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if cfg.Current != "dev" {
		t.Fatalf("current=%q want %q", cfg.Current, "dev")
	}
}

func TestRunUseInteractive_Cancelled(t *testing.T) {
	configPath := createUseTestConfig(t, &config.Config{
		Current: "work",
		Profiles: []config.Profile{
			{Name: "work", Domain: "example.atlassian.net", User: "work@example.com"},
		},
	})

	var out bytes.Buffer
	err := runUseInteractive(strings.NewReader("\x1b\n"), &out, &useOptions{configPath: configPath})
	if err != nil {
		t.Fatalf("runUseInteractive: %v", err)
	}
	if !strings.Contains(out.String(), "selection cancelled") {
		t.Fatalf("unexpected output: %q", out.String())
	}
}

func TestRunUseInteractive_InvalidSelection(t *testing.T) {
	configPath := createUseTestConfig(t, &config.Config{
		Current: "work",
		Profiles: []config.Profile{
			{Name: "work", Domain: "example.atlassian.net", User: "work@example.com"},
		},
	})

	var out bytes.Buffer
	err := runUseInteractive(strings.NewReader("9\n"), &out, &useOptions{configPath: configPath})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "invalid selection") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func createUseTestConfig(t *testing.T, cfg *config.Config) string {
	t.Helper()

	configPath := t.TempDir() + "/config.toml"
	if err := cfg.SaveTo(configPath); err != nil {
		t.Fatalf("SaveTo: %v", err)
	}
	return configPath
}
