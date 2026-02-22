package config

import (
	"path/filepath"
	"testing"
)

func TestConfig_SaveToLoadFrom_RoundTrip(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Current: "work",
		Profiles: []Profile{
			{
				Name:       "work",
				Domain:     "example.atlassian.net",
				User:       "user@example.com",
				SpaceKey:   "DOC",
				AssetsRoot: "/tmp/assets",
				Output:     "json",
			},
		},
	}

	path := filepath.Join(t.TempDir(), "config.toml")
	if err := cfg.SaveTo(path); err != nil {
		t.Fatalf("SaveTo: %v", err)
	}

	loaded, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if loaded.Current != "work" || len(loaded.Profiles) != 1 || loaded.Profiles[0].Name != "work" {
		t.Fatalf("unexpected loaded config: %+v", loaded)
	}
	if loaded.Profiles[0].AssetsRoot != "/tmp/assets" {
		t.Fatalf("assets_root=%q", loaded.Profiles[0].AssetsRoot)
	}
}

func TestConfig_LoadFrom_NotFound(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "missing.toml")
	cfg, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if cfg == nil || len(cfg.Profiles) != 0 || cfg.Current != "" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestConfig_ProfileOperations(t *testing.T) {
	t.Parallel()

	cfg := &Config{}

	work := &Profile{Name: "work", Domain: "example.atlassian.net", User: "user@example.com"}
	if err := cfg.AddProfile(work); err != nil {
		t.Fatalf("AddProfile: %v", err)
	}
	if cfg.Current != "work" {
		t.Fatalf("current=%q", cfg.Current)
	}

	if err := cfg.AddProfile(work); err == nil {
		t.Fatalf("expected duplicate profile error")
	}

	dev := &Profile{Name: "dev", Domain: "dev.atlassian.net", User: "dev@example.com"}
	if err := cfg.AddProfile(dev); err != nil {
		t.Fatalf("AddProfile dev: %v", err)
	}

	if err := cfg.SetCurrent("dev"); err != nil {
		t.Fatalf("SetCurrent: %v", err)
	}
	if cfg.CurrentProfile() == nil || cfg.CurrentProfile().Name != "dev" {
		t.Fatalf("unexpected current profile: %+v", cfg.CurrentProfile())
	}

	if err := cfg.SetCurrent("missing"); err == nil {
		t.Fatalf("expected missing profile error")
	}

	if err := cfg.DeleteProfile("dev"); err != nil {
		t.Fatalf("DeleteProfile: %v", err)
	}
	if cfg.Current != "" {
		t.Fatalf("expected current profile to be cleared, got %q", cfg.Current)
	}
	if err := cfg.DeleteProfile("missing"); err == nil {
		t.Fatalf("expected missing profile error")
	}
}

func TestDir_UsesXDGConfigHome(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg-home")
	got, err := Dir()
	if err != nil {
		t.Fatalf("Dir: %v", err)
	}
	if got != "/tmp/xdg-home/cflcli" {
		t.Fatalf("got %q want %q", got, "/tmp/xdg-home/cflcli")
	}
}
