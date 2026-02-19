package test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/takymt/cflcli/internal/config"
)

var cliBinaryPath string
var cliRepoRoot string

func TestMain(m *testing.M) {
	cliRepoRoot = findRepoRoot()

	tmpDir, err := os.MkdirTemp("", "cflcli-test-*")
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "create temp dir failed: %v\n", err)
		os.Exit(1)
	}

	cliBinaryPath = filepath.Join(tmpDir, "cfl")
	build := exec.CommandContext(context.Background(), "go", "build", "-o", cliBinaryPath, ".")
	build.Dir = cliRepoRoot
	build.Env = os.Environ()
	if out, buildErr := build.CombinedOutput(); buildErr != nil {
		_ = os.RemoveAll(tmpDir)
		_, _ = fmt.Fprintf(os.Stderr, "build failed: %v\n%s", buildErr, string(out))
		os.Exit(1)
	}

	code := m.Run()
	_ = os.RemoveAll(tmpDir)
	os.Exit(code)
}

func findRepoRoot() string {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return "."
	}
	return filepath.Dir(filepath.Dir(currentFile))
}

func runCLI(t *testing.T, xdgConfigHome string, stdin string, args ...string) (string, error) {
	t.Helper()

	cmd := exec.CommandContext(context.Background(), cliBinaryPath, args...)
	cmd.Dir = cliRepoRoot
	cmd.Env = withEnv(os.Environ(), "XDG_CONFIG_HOME", xdgConfigHome)
	cmd.Stdin = strings.NewReader(stdin)

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	err := cmd.Run()
	return out.String(), err
}

func withEnv(base []string, key string, value string) []string {
	prefix := key + "="
	result := make([]string, 0, len(base)+1)
	for _, item := range base {
		if strings.HasPrefix(item, prefix) {
			continue
		}
		result = append(result, item)
	}
	return append(result, prefix+value)
}

func configPath(xdgConfigHome string) string {
	return filepath.Join(xdgConfigHome, "cflcli", "config.toml")
}

func loadConfig(t *testing.T, xdgConfigHome string) *config.Config {
	t.Helper()
	cfg, err := config.LoadFrom(configPath(xdgConfigHome))
	if err != nil {
		t.Fatalf("load config failed: %v", err)
	}
	return cfg
}

func createProfile(t *testing.T, xdgConfigHome string, name string, domain string, user string, spaceKey string, output string) {
	t.Helper()
	out, err := runCLI(t, xdgConfigHome, "", "config", "init", name, "--domain", domain, "--user", user, "--space-key", spaceKey, "--profile-output", output)
	if err != nil {
		t.Fatalf("create profile %q failed: %v\n%s", name, err, out)
	}
}

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

func TestConfigInitUsesDefaultProfileValues(t *testing.T) {
	xdgConfigHome := t.TempDir()

	createProfile(t, xdgConfigHome, "default", "default.atlassian.net", "default@example.com", "DEF", "json")

	out, err := runCLI(t, xdgConfigHome, "\n\n\n\n", "config", "init", "work")
	if err != nil {
		t.Fatalf("unexpected error: %v\n%s", err, out)
	}
	if strings.Contains(out, "Profile Name: ") {
		t.Fatalf("did not expect profile name prompt with positional name: %s", out)
	}
	if !strings.Contains(out, "Output (json|table) [json]: ") {
		t.Fatalf("expected output prompt with default value, got: %s", out)
	}

	cfg := loadConfig(t, xdgConfigHome)
	p := cfg.FindProfile("work")
	if p == nil {
		t.Fatal("expected work profile to exist")
	}
	if p.Domain != "default.atlassian.net" || p.User != "default@example.com" || p.SpaceKey != "DEF" || p.Output != "json" {
		t.Fatalf("unexpected work profile values: %+v", *p)
	}
}

func TestConfigInitRejectsDuplicateProfileEarly(t *testing.T) {
	testCases := []struct {
		name   string
		args   []string
		stdin  string
		wantIn string
	}{
		{
			name:   "interactive name input",
			args:   []string{"config", "init"},
			stdin:  "dup\nSHOULD-NOT-BE-READ\n",
			wantIn: `profile "dup" already exists`,
		},
		{
			name:   "positional name",
			args:   []string{"config", "init", "dup"},
			stdin:  "SHOULD-NOT-BE-READ\n",
			wantIn: `profile "dup" already exists`,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			xdgConfigHome := t.TempDir()
			createProfile(t, xdgConfigHome, "default", "default.atlassian.net", "default@example.com", "DEF", "table")
			createProfile(t, xdgConfigHome, "dup", "dup.atlassian.net", "dup@example.com", "DUP", "json")

			out, err := runCLI(t, xdgConfigHome, tc.stdin, tc.args...)
			if err == nil {
				t.Fatalf("expected duplicate error, got success: %s", out)
			}
			if !strings.Contains(out, tc.wantIn) {
				t.Fatalf("expected output containing %q, got: %s", tc.wantIn, out)
			}
			if strings.Contains(out, "Domain: ") {
				t.Fatalf("expected early stop before Domain prompt, got: %s", out)
			}
		})
	}
}

func TestConfigInitRejectsInvalidOutput(t *testing.T) {
	xdgConfigHome := t.TempDir()

	out, err := runCLI(
		t,
		xdgConfigHome,
		"",
		"config",
		"init",
		"work",
		"--domain",
		"work.atlassian.net",
		"--user",
		"work@example.com",
		"--space-key",
		"WORK",
		"--profile-output",
		"yaml",
	)
	if err == nil {
		t.Fatalf("expected invalid output error, got success: %s", out)
	}
	if !strings.Contains(out, "output must be one of: json, table") {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestConfigEditCommand(t *testing.T) {
	t.Run("updates existing profile and keeps empty fields", func(t *testing.T) {
		xdgConfigHome := t.TempDir()

		createProfile(t, xdgConfigHome, "default", "default.atlassian.net", "default@example.com", "DEF", "json")
		createProfile(t, xdgConfigHome, "work", "work.atlassian.net", "work@example.com", "WORK", "table")

		out, err := runCLI(t, xdgConfigHome, "\nnew@example.com\n\njson\n", "config", "edit", "work")
		if err != nil {
			t.Fatalf("unexpected error: %v\n%s", err, out)
		}
		if !strings.Contains(out, `Profile "work" updated successfully.`) {
			t.Fatalf("unexpected output: %s", out)
		}

		cfg := loadConfig(t, xdgConfigHome)
		p := cfg.FindProfile("work")
		if p == nil {
			t.Fatal("expected work profile to exist")
		}
		if p.Domain != "work.atlassian.net" || p.User != "new@example.com" || p.SpaceKey != "WORK" || p.Output != "json" {
			t.Fatalf("unexpected work profile values after edit: %+v", *p)
		}
	})

	t.Run("returns not found for unknown profile", func(t *testing.T) {
		xdgConfigHome := t.TempDir()

		out, err := runCLI(t, xdgConfigHome, "", "config", "edit", "missing")
		if err == nil {
			t.Fatalf("expected error, got success: %s", out)
		}
		if !strings.Contains(out, `profile "missing" not found`) {
			t.Fatalf("unexpected output: %s", out)
		}
	})
}

func TestUseCommandsSwitchCurrentProfile(t *testing.T) {
	xdgConfigHome := t.TempDir()

	createProfile(t, xdgConfigHome, "default", "default.atlassian.net", "default@example.com", "DEF", "table")
	createProfile(t, xdgConfigHome, "work", "work.atlassian.net", "work@example.com", "WORK", "table")
	createProfile(t, xdgConfigHome, "personal", "personal.atlassian.net", "personal@example.com", "HOME", "json")

	commands := []struct {
		name        string
		args        []string
		wantCurrent string
		wantOutput  string
	}{
		{
			name:        "root use command",
			args:        []string{"use", "work"},
			wantCurrent: "work",
			wantOutput:  `Switched to profile "work".`,
		},
		{
			name:        "config use alias command",
			args:        []string{"config", "use", "personal"},
			wantCurrent: "personal",
			wantOutput:  `Switched to profile "personal".`,
		},
	}

	for _, cmdCase := range commands {
		cmdCase := cmdCase
		t.Run(cmdCase.name, func(t *testing.T) {
			out, err := runCLI(t, xdgConfigHome, "", cmdCase.args...)
			if err != nil {
				t.Fatalf("unexpected error: %v\n%s", err, out)
			}
			if !strings.Contains(out, cmdCase.wantOutput) {
				t.Fatalf("unexpected output: %s", out)
			}

			cfg := loadConfig(t, xdgConfigHome)
			if cfg.Current != cmdCase.wantCurrent {
				t.Fatalf("expected current %q, got %q", cmdCase.wantCurrent, cfg.Current)
			}
		})
	}
}

func TestConfigDeleteCommand(t *testing.T) {
	testCases := []struct {
		name             string
		setup            func(t *testing.T, xdgConfigHome string)
		args             []string
		wantErr          bool
		wantOutput       []string
		wantCurrent      string
		expectProfile    string
		expectProfileNil string
	}{
		{
			name: "delete non current profile",
			setup: func(t *testing.T, xdgConfigHome string) {
				createProfile(t, xdgConfigHome, "default", "default.atlassian.net", "default@example.com", "DEF", "table")
				createProfile(t, xdgConfigHome, "work", "work.atlassian.net", "work@example.com", "WORK", "table")
				createProfile(t, xdgConfigHome, "personal", "personal.atlassian.net", "personal@example.com", "HOME", "json")
			},
			args:             []string{"config", "delete", "personal"},
			wantErr:          false,
			wantOutput:       []string{`Profile "personal" deleted.`},
			wantCurrent:      "default",
			expectProfile:    "default",
			expectProfileNil: "personal",
		},
		{
			name: "delete current profile without force",
			setup: func(t *testing.T, xdgConfigHome string) {
				createProfile(t, xdgConfigHome, "default", "default.atlassian.net", "default@example.com", "DEF", "table")
				createProfile(t, xdgConfigHome, "work", "work.atlassian.net", "work@example.com", "WORK", "table")
				out, err := runCLI(t, xdgConfigHome, "", "use", "work")
				if err != nil {
					t.Fatalf("switch profile failed: %v\n%s", err, out)
				}
			},
			args:             []string{"config", "delete", "work"},
			wantErr:          true,
			wantOutput:       []string{`cannot delete current profile "work" without --force`},
			wantCurrent:      "work",
			expectProfile:    "work",
			expectProfileNil: "",
		},
		{
			name: "delete current profile with force switches to default",
			setup: func(t *testing.T, xdgConfigHome string) {
				createProfile(t, xdgConfigHome, "default", "default.atlassian.net", "default@example.com", "DEF", "table")
				createProfile(t, xdgConfigHome, "work", "work.atlassian.net", "work@example.com", "WORK", "table")
				out, err := runCLI(t, xdgConfigHome, "", "use", "work")
				if err != nil {
					t.Fatalf("switch profile failed: %v\n%s", err, out)
				}
			},
			args:             []string{"config", "delete", "work", "--force"},
			wantErr:          false,
			wantOutput:       []string{`Profile "work" deleted.`, `Current profile switched to "default".`},
			wantCurrent:      "default",
			expectProfile:    "default",
			expectProfileNil: "work",
		},
		{
			name: "delete current profile with force but default missing",
			setup: func(t *testing.T, xdgConfigHome string) {
				createProfile(t, xdgConfigHome, "work", "work.atlassian.net", "work@example.com", "WORK", "table")
				createProfile(t, xdgConfigHome, "personal", "personal.atlassian.net", "personal@example.com", "HOME", "json")
			},
			args:             []string{"config", "delete", "work", "--force"},
			wantErr:          true,
			wantOutput:       []string{`profile "default" not found`},
			wantCurrent:      "work",
			expectProfile:    "work",
			expectProfileNil: "",
		},
		{
			name: "delete default current profile with force",
			setup: func(t *testing.T, xdgConfigHome string) {
				createProfile(t, xdgConfigHome, "default", "default.atlassian.net", "default@example.com", "DEF", "table")
			},
			args:             []string{"config", "delete", "default", "--force"},
			wantErr:          true,
			wantOutput:       []string{`cannot delete current profile "default" with --force`},
			wantCurrent:      "default",
			expectProfile:    "default",
			expectProfileNil: "",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			xdgConfigHome := t.TempDir()
			tc.setup(t, xdgConfigHome)

			out, err := runCLI(t, xdgConfigHome, "", tc.args...)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error, got success: %s", out)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v\n%s", err, out)
			}
			for _, expected := range tc.wantOutput {
				if !strings.Contains(out, expected) {
					t.Fatalf("expected output containing %q, got: %s", expected, out)
				}
			}

			cfg := loadConfig(t, xdgConfigHome)
			if cfg.Current != tc.wantCurrent {
				t.Fatalf("expected current %q, got %q", tc.wantCurrent, cfg.Current)
			}
			if tc.expectProfile != "" && cfg.FindProfile(tc.expectProfile) == nil {
				t.Fatalf("expected profile %q to exist", tc.expectProfile)
			}
			if tc.expectProfileNil != "" && cfg.FindProfile(tc.expectProfileNil) != nil {
				t.Fatalf("expected profile %q to be deleted", tc.expectProfileNil)
			}
		})
	}
}
