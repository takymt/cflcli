package cmd

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/takymt/cflcli/internal/config"
)

func TestUse_WithName(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")

	cfg := &config.Config{
		Current: "work",
		Profiles: []config.Profile{
			{Name: "work", Domain: "work.atlassian.net", User: "work@example.com", SpaceKey: "WORK"},
			{Name: "personal", Domain: "personal.atlassian.net", User: "me@example.com", SpaceKey: "HOME"},
		},
	}
	if err := cfg.SaveTo(configPath); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	opts := &useOptions{configPath: configPath}
	out := &bytes.Buffer{}
	if err := runUse(out, "personal", opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, `Switched to profile "personal".`) {
		t.Errorf("unexpected output: %s", output)
	}

	loaded, err := config.LoadFrom(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	if loaded.Current != "personal" {
		t.Errorf("expected current %q, got %q", "personal", loaded.Current)
	}
}

func TestUse_WithName_NotFound(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")

	cfg := &config.Config{
		Current: "work",
		Profiles: []config.Profile{
			{Name: "work", Domain: "work.atlassian.net", User: "work@example.com", SpaceKey: "WORK"},
		},
	}
	if err := cfg.SaveTo(configPath); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	opts := &useOptions{configPath: configPath}
	out := &bytes.Buffer{}
	err := runUse(out, "nonexistent", opts)
	if err == nil {
		t.Fatal("expected error for nonexistent profile, got nil")
	}
}

func TestUseInteractive_Select(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")

	cfg := &config.Config{
		Current: "work",
		Profiles: []config.Profile{
			{Name: "work", Domain: "work.atlassian.net", User: "work@example.com", SpaceKey: "WORK"},
			{Name: "personal", Domain: "personal.atlassian.net", User: "me@example.com", SpaceKey: "HOME"},
		},
	}
	if err := cfg.SaveTo(configPath); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	opts := &useOptions{configPath: configPath}
	in := strings.NewReader("2\n")
	out := &bytes.Buffer{}
	if err := runUseInteractive(in, out, opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "1) work (current)") {
		t.Errorf("expected current marker on work: %s", output)
	}
	if !strings.Contains(output, "2) personal") {
		t.Errorf("expected personal in list: %s", output)
	}
	if !strings.Contains(output, `Switched to profile "personal".`) {
		t.Errorf("unexpected output: %s", output)
	}

	loaded, err := config.LoadFrom(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	if loaded.Current != "personal" {
		t.Errorf("expected current %q, got %q", "personal", loaded.Current)
	}
}

func TestUseInteractive_NoProfiles(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")

	opts := &useOptions{configPath: configPath}
	in := strings.NewReader("")
	out := &bytes.Buffer{}
	if err := runUseInteractive(in, out, opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "No profiles configured") {
		t.Errorf("expected no profiles message, got: %s", output)
	}
}

func TestUseInteractive_EOF(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")

	cfg := &config.Config{
		Current: "work",
		Profiles: []config.Profile{
			{Name: "work", Domain: "work.atlassian.net", User: "work@example.com", SpaceKey: "WORK"},
		},
	}
	if err := cfg.SaveTo(configPath); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	opts := &useOptions{configPath: configPath}
	in := strings.NewReader("") // EOF
	out := &bytes.Buffer{}
	err := runUseInteractive(in, out, opts)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "read input") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestUseInteractive_ESC(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")

	cfg := &config.Config{
		Current: "work",
		Profiles: []config.Profile{
			{Name: "work", Domain: "work.atlassian.net", User: "work@example.com", SpaceKey: "WORK"},
		},
	}
	if err := cfg.SaveTo(configPath); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	opts := &useOptions{configPath: configPath}
	in := strings.NewReader("\x1b\n")
	out := &bytes.Buffer{}
	if err := runUseInteractive(in, out, opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "selection cancelled") {
		t.Errorf("expected cancelled message, got: %s", output)
	}
}

func TestUseInteractive_InvalidInput(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")

	cfg := &config.Config{
		Current: "work",
		Profiles: []config.Profile{
			{Name: "work", Domain: "work.atlassian.net", User: "work@example.com", SpaceKey: "WORK"},
		},
	}
	if err := cfg.SaveTo(configPath); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	opts := &useOptions{configPath: configPath}
	in := strings.NewReader("abc\n")
	out := &bytes.Buffer{}
	err := runUseInteractive(in, out, opts)
	if err == nil {
		t.Fatal("expected error for invalid input, got nil")
	}
	if !strings.Contains(err.Error(), "invalid selection") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestUseInteractive_OutOfRange(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")

	cfg := &config.Config{
		Current: "work",
		Profiles: []config.Profile{
			{Name: "work", Domain: "work.atlassian.net", User: "work@example.com", SpaceKey: "WORK"},
		},
	}
	if err := cfg.SaveTo(configPath); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	opts := &useOptions{configPath: configPath}
	in := strings.NewReader("99\n")
	out := &bytes.Buffer{}
	err := runUseInteractive(in, out, opts)
	if err == nil {
		t.Fatal("expected error for out of range, got nil")
	}
	if !strings.Contains(err.Error(), "must be between") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestUseInteractive_EmptyInput(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")

	cfg := &config.Config{
		Current: "work",
		Profiles: []config.Profile{
			{Name: "work", Domain: "work.atlassian.net", User: "work@example.com", SpaceKey: "WORK"},
		},
	}
	if err := cfg.SaveTo(configPath); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	opts := &useOptions{configPath: configPath}
	in := strings.NewReader("\n")
	out := &bytes.Buffer{}
	err := runUseInteractive(in, out, opts)
	if err == nil {
		t.Fatal("expected error for empty input, got nil")
	}
	if !strings.Contains(err.Error(), "invalid selection") {
		t.Errorf("unexpected error message: %v", err)
	}
}
