package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunShowsCommandUsageForInputErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		args         []string
		wantError    string
		wantUsageFor string
	}{
		{
			name:         "page new missing required flags",
			args:         []string{"page", "new"},
			wantError:    `required flag(s) "space-key", "title" not set`,
			wantUsageFor: "cfl page new",
		},
		{
			name:         "page new unexpected arg",
			args:         []string{"page", "new", "one.md", "--space-key", "TEST", "--title", "One"},
			wantError:    `unknown command "one.md" for "cfl page new"`,
			wantUsageFor: "cfl page new",
		},
		{
			name:         "page new missing title flag",
			args:         []string{"page", "new", "--space-key", "TEST"},
			wantError:    `required flag(s) "title" not set`,
			wantUsageFor: "cfl page new",
		},
		{
			name:         "page new missing space key flag",
			args:         []string{"page", "new", "--title", "One"},
			wantError:    `required flag(s) "space-key" not set`,
			wantUsageFor: "cfl page new",
		},
		{
			name:         "page new unknown flag",
			args:         []string{"page", "new", "--space-key", "TEST", "--title", "One", "--bogus"},
			wantError:    "unknown flag: --bogus",
			wantUsageFor: "cfl page new",
		},
		{
			name:         "page new removed slug flag",
			args:         []string{"page", "new", "--space-key", "TEST", "--title", "One", "--slug", "one"},
			wantError:    "unknown flag: --slug",
			wantUsageFor: "cfl page new",
		},
		{
			name:         "page sync missing arg",
			args:         []string{"page", "sync"},
			wantError:    "accepts 1 arg(s), received 0",
			wantUsageFor: "cfl page sync <path>",
		},
		{
			name:         "page sync extra arg",
			args:         []string{"page", "sync", "one.md", "two.md"},
			wantError:    "accepts 1 arg(s), received 2",
			wantUsageFor: "cfl page sync <path>",
		},
		{
			name:         "page sync unknown flag",
			args:         []string{"page", "sync", "one.md", "--bogus"},
			wantError:    "unknown flag: --bogus",
			wantUsageFor: "cfl page sync <path>",
		},
		{
			name:         "attachment put missing arg",
			args:         []string{"attachment", "put", "--page-id", "400"},
			wantError:    "accepts 1 arg(s), received 0",
			wantUsageFor: "cfl attachment put <file>",
		},
		{
			name:         "attachment put extra arg",
			args:         []string{"attachment", "put", "--page-id", "400", "a.svg", "b.svg"},
			wantError:    "accepts 1 arg(s), received 2",
			wantUsageFor: "cfl attachment put <file>",
		},
		{
			name:         "attachment put missing page id",
			args:         []string{"attachment", "put", "a.svg"},
			wantError:    `required flag(s) "page-id" not set`,
			wantUsageFor: "cfl attachment put <file>",
		},
		{
			name:         "attachment put unknown flag",
			args:         []string{"attachment", "put", "--page-id", "400", "a.svg", "--bogus"},
			wantError:    "unknown flag: --bogus",
			wantUsageFor: "cfl attachment put <file>",
		},
		{
			name:         "attachment delete missing arg",
			args:         []string{"attachment", "delete", "--page-id", "400"},
			wantError:    "accepts 1 arg(s), received 0",
			wantUsageFor: "cfl attachment delete <filename>",
		},
		{
			name:         "attachment delete extra arg",
			args:         []string{"attachment", "delete", "--page-id", "400", "a.svg", "b.svg"},
			wantError:    "accepts 1 arg(s), received 2",
			wantUsageFor: "cfl attachment delete <filename>",
		},
		{
			name:         "attachment delete missing page id",
			args:         []string{"attachment", "delete", "a.svg"},
			wantError:    `required flag(s) "page-id" not set`,
			wantUsageFor: "cfl attachment delete <filename>",
		},
		{
			name:         "attachment delete unknown flag",
			args:         []string{"attachment", "delete", "--page-id", "400", "a.svg", "--bogus"},
			wantError:    "unknown flag: --bogus",
			wantUsageFor: "cfl attachment delete <filename>",
		},
		{
			name:         "auth unknown flag",
			args:         []string{"auth", "--bogus"},
			wantError:    "unknown flag: --bogus",
			wantUsageFor: "cfl auth [flags]",
		},
		{
			name:         "auth unknown command",
			args:         []string{"auth", "bogus"},
			wantError:    `unknown command "bogus" for "cfl auth"`,
			wantUsageFor: "cfl auth [flags]",
		},
		{
			name:         "auth login extra arg",
			args:         []string{"auth", "login", "extra"},
			wantError:    `unknown command "extra" for "cfl auth login"`,
			wantUsageFor: "cfl auth login",
		},
		{
			name:         "auth login unknown flag",
			args:         []string{"auth", "login", "--bogus"},
			wantError:    "unknown flag: --bogus",
			wantUsageFor: "cfl auth login",
		},
		{
			name:         "auth logout extra arg",
			args:         []string{"auth", "logout", "extra"},
			wantError:    `unknown command "extra" for "cfl auth logout"`,
			wantUsageFor: "cfl auth logout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			var stdout bytes.Buffer
			app := New(newFakeClient(), &stdout)

			exit := app.Run(context.Background(), tt.args, dir)
			if exit != 1 {
				t.Fatalf("Run() exit = %d, want 1", exit)
			}

			out := stdout.String()
			if !strings.Contains(out, tt.wantError) {
				t.Fatalf("output = %q, want error %q", out, tt.wantError)
			}
			if !strings.Contains(out, "Usage:") {
				t.Fatalf("output = %q, want usage", out)
			}
			if !strings.Contains(out, tt.wantUsageFor) {
				t.Fatalf("output = %q, want usage for %q", out, tt.wantUsageFor)
			}
		})
	}
}

func TestRunDoesNotShowUsageForRuntimeErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       []string
		setup      func(t *testing.T, dir string, app *App)
		wantOutput string
	}{
		{
			name: "page new existing file",
			args: []string{"page", "new", "--title", "Guide", "--space-key", "TEST"},
			setup: func(t *testing.T, dir string, app *App) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(dir, "Guide.md"), []byte("existing"), 0o644); err != nil {
					t.Fatalf("WriteFile() error = %v", err)
				}
			},
			wantOutput: "already exists",
		},
		{
			name:       "page new empty title",
			args:       []string{"page", "new", "--title", "   ", "--space-key", "TEST"},
			wantOutput: "title must not be empty",
		},
		{
			name:       "page sync missing file",
			args:       []string{"page", "sync", "missing.md"},
			wantOutput: "missing.md",
		},
		{
			name:       "attachment put missing file",
			args:       []string{"attachment", "put", "--page-id", "400", "missing.svg"},
			wantOutput: "missing.svg",
		},
		{
			name: "auth login validation failure",
			args: []string{
				"auth", "login",
				"--domain", "example.atlassian.net",
				"--email", "user@example.com",
				"--api-token", "secret",
			},
			setup: func(t *testing.T, dir string, app *App) {
				t.Helper()
				app.authValidator = &recordingValidator{err: errors.New("validation failed")}
			},
			wantOutput: "validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			var stdout bytes.Buffer
			app := New(newFakeClient(), &stdout)
			if tt.setup != nil {
				tt.setup(t, dir, app)
			}

			exit := app.Run(context.Background(), tt.args, dir)
			if exit != 1 {
				t.Fatalf("Run() exit = %d, want 1", exit)
			}

			out := stdout.String()
			if !strings.Contains(out, tt.wantOutput) {
				t.Fatalf("output = %q, want runtime error %q", out, tt.wantOutput)
			}
			if strings.Contains(out, "Usage:") {
				t.Fatalf("output = %q, must not include usage for runtime errors", out)
			}
		})
	}
}
