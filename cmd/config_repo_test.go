package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/takymt/cflcli/internal/config"
)

func TestRunConfigShow_DisplaysSourcesWithRepoOverrides(t *testing.T) {
	configPath := createUseTestConfig(t, &config.Config{
		Current: "work",
		Profiles: []config.Profile{
			{
				Name:     "work",
				Domain:   "profile.example.atlassian.net",
				User:     "user@example.com",
				SpaceKey: "PROFILE",
				Output:   "table",
			},
		},
	})

	repoDir := t.TempDir()
	writeRepoConfig(t, repoDir, `domain = "repo.example.atlassian.net"`+"\n"+`space_id = "123456"`)
	chdir(t, repoDir)

	var out bytes.Buffer
	if err := runConfigShow(&out, &configShowOptions{configPath: configPath}); err != nil {
		t.Fatalf("runConfigShow: %v", err)
	}
	got := out.String()

	if !strings.Contains(got, "Domain:") || !strings.Contains(got, "(source: repo)") {
		t.Fatalf("domain source not shown: %q", got)
	}
	if !strings.Contains(got, "Space:") || !strings.Contains(got, "space_id=123456") || !strings.Contains(got, "(source: repo)") {
		t.Fatalf("space source not shown: %q", got)
	}
	if !strings.Contains(got, "Repo Config:") || !strings.Contains(got, "cfl.toml") {
		t.Fatalf("repo config path not shown: %q", got)
	}
}

func TestRunConfigRepoInit_CreatesRepoConfigFromCurrentProfile(t *testing.T) {
	configPath := createUseTestConfig(t, &config.Config{
		Current: "work",
		Profiles: []config.Profile{
			{
				Name:     "work",
				Domain:   "example.atlassian.net",
				User:     "user@example.com",
				SpaceKey: "WORK",
			},
		},
	})

	repoDir := t.TempDir()
	chdir(t, repoDir)

	var out bytes.Buffer
	if err := runConfigRepoInit(&out, &configRepoInitOptions{configPath: configPath}); err != nil {
		t.Fatalf("runConfigRepoInit: %v", err)
	}

	repoPath := filepath.Join(repoDir, "cfl.toml")
	body, err := os.ReadFile(repoPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	raw := string(body)
	if !strings.Contains(raw, `domain = "example.atlassian.net"`) {
		t.Fatalf("missing domain in repo config: %q", raw)
	}
	if !strings.Contains(raw, `space_key = "WORK"`) {
		t.Fatalf("missing space_key in repo config: %q", raw)
	}
	if !strings.Contains(out.String(), `Repo config "cfl.toml" created.`) {
		t.Fatalf("unexpected output: %q", out.String())
	}
}
