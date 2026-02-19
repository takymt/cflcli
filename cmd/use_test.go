package cmd

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/takymt/cflcli/internal/config"
)

func TestRunUseInteractive(t *testing.T) {
	testCases := []struct {
		name         string
		current      string
		profiles     []config.Profile
		input        string
		wantErr      string
		wantContains string
		wantCurrent  string
		checkConfig  bool
	}{
		{
			name:    "select profile by number",
			current: "work",
			profiles: []config.Profile{
				{Name: "default", Domain: "default.atlassian.net", User: "default@example.com", SpaceKey: "DEF"},
				{Name: "work", Domain: "work.atlassian.net", User: "work@example.com", SpaceKey: "WORK"},
				{Name: "personal", Domain: "personal.atlassian.net", User: "me@example.com", SpaceKey: "HOME"},
			},
			input:        "3\n",
			wantContains: `Switched to profile "personal".`,
			wantCurrent:  "personal",
			checkConfig:  true,
		},
		{
			name:         "no profiles configured",
			input:        "",
			wantContains: "No profiles configured. Run 'cfl config init' to create one.",
		},
		{
			name:    "esc cancels selection",
			current: "work",
			profiles: []config.Profile{
				{Name: "work", Domain: "work.atlassian.net", User: "work@example.com", SpaceKey: "WORK"},
			},
			input:        "\x1b\n",
			wantContains: "selection cancelled",
			wantCurrent:  "work",
			checkConfig:  true,
		},
		{
			name:    "invalid non numeric selection",
			current: "work",
			profiles: []config.Profile{
				{Name: "work", Domain: "work.atlassian.net", User: "work@example.com", SpaceKey: "WORK"},
			},
			input:       "abc\n",
			wantErr:     "invalid selection",
			wantCurrent: "work",
			checkConfig: true,
		},
		{
			name:    "out of range selection",
			current: "work",
			profiles: []config.Profile{
				{Name: "work", Domain: "work.atlassian.net", User: "work@example.com", SpaceKey: "WORK"},
			},
			input:       "9\n",
			wantErr:     "must be between",
			wantCurrent: "work",
			checkConfig: true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			configPath := filepath.Join(dir, "config.toml")

			if len(tc.profiles) > 0 {
				cfg := &config.Config{
					Current:  tc.current,
					Profiles: tc.profiles,
				}
				if err := cfg.SaveTo(configPath); err != nil {
					t.Fatalf("save config failed: %v", err)
				}
			}

			opts := &useOptions{configPath: configPath}
			out := &bytes.Buffer{}
			err := runUseInteractive(strings.NewReader(tc.input), out, opts)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tc.wantContains != "" && !strings.Contains(out.String(), tc.wantContains) {
				t.Fatalf("expected output containing %q, got: %s", tc.wantContains, out.String())
			}

			if tc.checkConfig {
				loaded, loadErr := config.LoadFrom(configPath)
				if loadErr != nil {
					t.Fatalf("load config failed: %v", loadErr)
				}
				if loaded.Current != tc.wantCurrent {
					t.Fatalf("expected current %q, got %q", tc.wantCurrent, loaded.Current)
				}
			}
		})
	}
}
