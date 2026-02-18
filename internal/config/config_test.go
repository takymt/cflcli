package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFrom_NonExistentFile(t *testing.T) {
	cfg, err := LoadFrom("/tmp/cflcli-test-nonexistent/config.toml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Current != "" {
		t.Errorf("expected empty current, got %q", cfg.Current)
	}
	if len(cfg.Profiles) != 0 {
		t.Errorf("expected 0 profiles, got %d", len(cfg.Profiles))
	}
}

func TestSaveToAndLoadFrom(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	cfg := &Config{
		Current: "work",
		Profiles: []Profile{
			{
				Name:     "work",
				Domain:   "example.atlassian.net",
				User:     "user@example.com",
				SpaceKey: "DEV",
			},
		},
	}

	if err := cfg.SaveTo(path); err != nil {
		t.Fatalf("SaveTo failed: %v", err)
	}

	// Verify file permissions
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}
	perm := info.Mode().Perm()
	if perm != 0o600 {
		t.Errorf("expected file permission 0600, got %o", perm)
	}

	loaded, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom failed: %v", err)
	}

	if loaded.Current != "work" {
		t.Errorf("expected current %q, got %q", "work", loaded.Current)
	}
	if len(loaded.Profiles) != 1 {
		t.Fatalf("expected 1 profile, got %d", len(loaded.Profiles))
	}
	p := loaded.Profiles[0]
	if p.Name != "work" {
		t.Errorf("expected name %q, got %q", "work", p.Name)
	}
	if p.Domain != "example.atlassian.net" {
		t.Errorf("expected domain %q, got %q", "example.atlassian.net", p.Domain)
	}
	if p.User != "user@example.com" {
		t.Errorf("expected user %q, got %q", "user@example.com", p.User)
	}
	if p.SpaceKey != "DEV" {
		t.Errorf("expected space_key %q, got %q", "DEV", p.SpaceKey)
	}
}

func TestAddProfile(t *testing.T) {
	cfg := &Config{}

	p1 := &Profile{Name: "first", Domain: "a.atlassian.net", User: "a@a.com", SpaceKey: "A"}
	if err := cfg.AddProfile(p1); err != nil {
		t.Fatalf("AddProfile failed: %v", err)
	}
	// First profile should be set as current
	if cfg.Current != "first" {
		t.Errorf("expected current %q, got %q", "first", cfg.Current)
	}

	p2 := &Profile{Name: "second", Domain: "b.atlassian.net", User: "b@b.com", SpaceKey: "B"}
	if err := cfg.AddProfile(p2); err != nil {
		t.Fatalf("AddProfile failed: %v", err)
	}
	// Current should not change
	if cfg.Current != "first" {
		t.Errorf("expected current %q, got %q", "first", cfg.Current)
	}
	if len(cfg.Profiles) != 2 {
		t.Fatalf("expected 2 profiles, got %d", len(cfg.Profiles))
	}
}

func TestAddProfile_Duplicate(t *testing.T) {
	cfg := &Config{}

	p := &Profile{Name: "dup", Domain: "a.atlassian.net", User: "a@a.com", SpaceKey: "A"}
	if err := cfg.AddProfile(p); err != nil {
		t.Fatalf("AddProfile failed: %v", err)
	}
	if err := cfg.AddProfile(p); err == nil {
		t.Error("expected error for duplicate profile, got nil")
	}
}

func TestFindProfile(t *testing.T) {
	cfg := &Config{
		Profiles: []Profile{
			{Name: "one"},
			{Name: "two"},
		},
	}

	if p := cfg.FindProfile("one"); p == nil {
		t.Error("expected to find profile 'one'")
	}
	if p := cfg.FindProfile("nonexistent"); p != nil {
		t.Error("expected nil for nonexistent profile")
	}
}

func TestSetCurrent(t *testing.T) {
	cfg := &Config{
		Current: "first",
		Profiles: []Profile{
			{Name: "first"},
			{Name: "second"},
		},
	}

	if err := cfg.SetCurrent("second"); err != nil {
		t.Fatalf("SetCurrent failed: %v", err)
	}
	if cfg.Current != "second" {
		t.Errorf("expected current %q, got %q", "second", cfg.Current)
	}
}

func TestSetCurrent_NotFound(t *testing.T) {
	cfg := &Config{
		Current: "first",
		Profiles: []Profile{
			{Name: "first"},
		},
	}

	err := cfg.SetCurrent("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent profile, got nil")
	}
	if cfg.Current != "first" {
		t.Errorf("current should not have changed, got %q", cfg.Current)
	}
}

func TestDeleteProfile(t *testing.T) {
	cfg := &Config{
		Current: "first",
		Profiles: []Profile{
			{Name: "first", Domain: "a.atlassian.net", User: "a@a.com", SpaceKey: "A"},
			{Name: "second", Domain: "b.atlassian.net", User: "b@b.com", SpaceKey: "B"},
		},
	}

	// Delete non-current profile succeeds
	if err := cfg.DeleteProfile("second"); err != nil {
		t.Fatalf("DeleteProfile failed: %v", err)
	}
	if len(cfg.Profiles) != 1 {
		t.Fatalf("expected 1 profile, got %d", len(cfg.Profiles))
	}
	if cfg.Profiles[0].Name != "first" {
		t.Errorf("expected remaining profile %q, got %q", "first", cfg.Profiles[0].Name)
	}
}

func TestDeleteProfile_CurrentProfile(t *testing.T) {
	cfg := &Config{
		Current: "active",
		Profiles: []Profile{
			{Name: "active"},
			{Name: "other"},
		},
	}

	if err := cfg.DeleteProfile("active"); err != nil {
		t.Fatalf("DeleteProfile failed: %v", err)
	}
	if len(cfg.Profiles) != 1 {
		t.Fatalf("expected 1 profile, got %d", len(cfg.Profiles))
	}
	if cfg.Profiles[0].Name != "other" {
		t.Errorf("expected remaining profile %q, got %q", "other", cfg.Profiles[0].Name)
	}
	if cfg.Current != "" {
		t.Errorf("expected current to be cleared, got %q", cfg.Current)
	}
}

func TestDeleteProfile_NotFound(t *testing.T) {
	cfg := &Config{
		Current: "first",
		Profiles: []Profile{
			{Name: "first"},
		},
	}

	err := cfg.DeleteProfile("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent profile, got nil")
	}
}

func TestCurrentProfile(t *testing.T) {
	cfg := &Config{
		Current: "active",
		Profiles: []Profile{
			{Name: "active"},
			{Name: "inactive"},
		},
	}

	p := cfg.CurrentProfile()
	if p == nil {
		t.Fatal("expected current profile, got nil")
	}
	if p.Name != "active" {
		t.Errorf("expected %q, got %q", "active", p.Name)
	}

	cfg.Current = ""
	if p := cfg.CurrentProfile(); p != nil {
		t.Error("expected nil when current is empty")
	}

	cfg.Current = "missing"
	if p := cfg.CurrentProfile(); p != nil {
		t.Error("expected nil when current profile doesn't exist")
	}
}
