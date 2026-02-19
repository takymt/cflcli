package e2e

import (
	"strings"
	"testing"
)

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
