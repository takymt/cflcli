package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/takymt/cflcli/internal/auth"
	"gopkg.in/yaml.v3"
)

func TestRunAuthAliasPromptsAndSavesCredentials(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)

	var stdout bytes.Buffer
	app := New(newFakeClient(), &stdout)
	app.authPrompter = &fakePrompter{
		promptValues: []string{
			"example.atlassian.net",
			"user@example.com",
		},
		secretValues: []string{"secret"},
	}
	validator := &recordingValidator{}
	app.authValidator = validator

	exit := app.Run(context.Background(), []string{"auth"}, configHome)
	if exit != 0 {
		t.Fatalf("Run(auth) exit = %d, want 0", exit)
	}

	if len(validator.calls) != 1 {
		t.Fatalf("validator calls = %d, want 1", len(validator.calls))
	}
	if validator.calls[0].Domain != "example.atlassian.net" {
		t.Fatalf("validated domain = %q, want %q", validator.calls[0].Domain, "example.atlassian.net")
	}

	path := filepath.Join(configHome, "cflcli", "config.yml")
	config := readYAMLMap(t, path)
	if got := config["domain"]; got != "example.atlassian.net" {
		t.Fatalf("saved domain = %#v, want %q", got, "example.atlassian.net")
	}
	if got := config["email"]; got != "user@example.com" {
		t.Fatalf("saved email = %#v, want %q", got, "user@example.com")
	}
	if got := config["api_token"]; got != "secret" {
		t.Fatalf("saved api_token = %#v, want %q", got, "secret")
	}
	if !strings.Contains(stdout.String(), "Saved credentials to "+path) {
		t.Fatalf("output = %q, want save path", stdout.String())
	}
}

func TestRunAuthLoginValidatesWithEnvOverridesAndSavesConfigValues(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("CONFLUENCE_EMAIL", "env@example.com")

	var stdout bytes.Buffer
	app := New(newFakeClient(), &stdout)
	app.authPrompter = &fakePrompter{
		promptValues: []string{
			"config.atlassian.net",
			"config@example.com",
		},
		secretValues: []string{"config-token"},
	}
	validator := &recordingValidator{}
	app.authValidator = validator

	exit := app.Run(context.Background(), []string{"auth", "login"}, configHome)
	if exit != 0 {
		t.Fatalf("Run(auth login) exit = %d, want 0", exit)
	}

	if len(validator.calls) != 1 {
		t.Fatalf("validator calls = %d, want 1", len(validator.calls))
	}
	got := validator.calls[0]
	if got.Domain != "config.atlassian.net" {
		t.Fatalf("validated domain = %q, want %q", got.Domain, "config.atlassian.net")
	}
	if got.Email != "env@example.com" {
		t.Fatalf("validated email = %q, want %q", got.Email, "env@example.com")
	}
	if got.APIToken != "config-token" {
		t.Fatalf("validated api token = %q, want %q", got.APIToken, "config-token")
	}

	path := filepath.Join(configHome, "cflcli", "config.yml")
	config := readYAMLMap(t, path)
	if got := config["email"]; got != "config@example.com" {
		t.Fatalf("saved email = %#v, want %q", got, "config@example.com")
	}
}

func TestRunAuthLoginNoValidateSkipsValidator(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)

	var stdout bytes.Buffer
	app := New(newFakeClient(), &stdout)
	validator := &recordingValidator{}
	app.authValidator = validator

	exit := app.Run(context.Background(), []string{
		"auth", "login",
		"--domain", "example.atlassian.net",
		"--email", "user@example.com",
		"--api-token", "secret",
		"--no-validate",
	}, configHome)
	if exit != 0 {
		t.Fatalf("Run(auth login) exit = %d, want 0", exit)
	}
	if len(validator.calls) != 0 {
		t.Fatalf("validator calls = %d, want 0", len(validator.calls))
	}
}

func TestRunAuthLoginValidationFailureDoesNotOverwriteConfig(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)

	path := filepath.Join(configHome, "cflcli", "config.yml")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, []byte("domain: old.atlassian.net\nemail: old@example.com\napi_token: old-token\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	var stdout bytes.Buffer
	app := New(newFakeClient(), &stdout)
	app.authValidator = &recordingValidator{
		err: errors.New("401 Unauthorized: invalid token"),
	}

	exit := app.Run(context.Background(), []string{
		"auth", "login",
		"--domain", "new.atlassian.net",
		"--email", "new@example.com",
		"--api-token", "new-token",
	}, configHome)
	if exit != 1 {
		t.Fatalf("Run(auth login) exit = %d, want 1", exit)
	}

	config := readYAMLMap(t, path)
	if got := config["domain"]; got != "old.atlassian.net" {
		t.Fatalf("domain after failed login = %#v, want %q", got, "old.atlassian.net")
	}
	if got := config["email"]; got != "old@example.com" {
		t.Fatalf("email after failed login = %#v, want %q", got, "old@example.com")
	}
	if got := config["api_token"]; got != "old-token" {
		t.Fatalf("api_token after failed login = %#v, want %q", got, "old-token")
	}
}

func TestRunAuthLogoutDeletesCredentialKeysAndSucceedsWithoutFile(t *testing.T) {
	t.Run("removes auth keys only", func(t *testing.T) {
		configHome := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", configHome)

		path := filepath.Join(configHome, "cflcli", "config.yml")
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatalf("MkdirAll() error = %v", err)
		}
		if err := os.WriteFile(path, []byte("other: keep\ndomain: d\nemail: e\napi_token: t\n"), 0o600); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		var stdout bytes.Buffer
		app := New(newFakeClient(), &stdout)
		exit := app.Run(context.Background(), []string{"auth", "logout"}, configHome)
		if exit != 0 {
			t.Fatalf("Run(auth logout) exit = %d, want 0", exit)
		}

		config := readYAMLMap(t, path)
		if got := config["other"]; got != "keep" {
			t.Fatalf("other = %#v, want %q", got, "keep")
		}
		if _, ok := config["domain"]; ok {
			t.Fatalf("domain key must be deleted, config = %#v", config)
		}
		if !strings.Contains(stdout.String(), "Cleared credentials in "+path) {
			t.Fatalf("output = %q, want clear path", stdout.String())
		}
	})

	t.Run("missing file is success", func(t *testing.T) {
		configHome := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", configHome)

		var stdout bytes.Buffer
		app := New(newFakeClient(), &stdout)
		exit := app.Run(context.Background(), []string{"auth", "logout"}, configHome)
		if exit != 0 {
			t.Fatalf("Run(auth logout) exit = %d, want 0", exit)
		}
	})
}

type fakePrompter struct {
	promptValues []string
	secretValues []string
	promptErr    error
	secretErr    error
}

func (f *fakePrompter) Prompt(label string) (string, error) {
	if f.promptErr != nil {
		return "", f.promptErr
	}
	if len(f.promptValues) == 0 {
		return "", nil
	}
	value := f.promptValues[0]
	f.promptValues = f.promptValues[1:]
	return value, nil
}

func (f *fakePrompter) PromptSecret(label string) (string, error) {
	if f.secretErr != nil {
		return "", f.secretErr
	}
	if len(f.secretValues) == 0 {
		return "", nil
	}
	value := f.secretValues[0]
	f.secretValues = f.secretValues[1:]
	return value, nil
}

type recordingValidator struct {
	calls []auth.Credentials
	err   error
}

func (v *recordingValidator) Validate(ctx context.Context, creds auth.Credentials) error {
	v.calls = append(v.calls, creds)
	return v.err
}

func readYAMLMap(t *testing.T, path string) map[string]any {
	t.Helper()

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	if strings.TrimSpace(string(raw)) == "" {
		return map[string]any{}
	}

	var config map[string]any
	if err := yaml.Unmarshal(raw, &config); err != nil {
		t.Fatalf("yaml.Unmarshal(%q) error = %v", path, err)
	}
	if config == nil {
		return map[string]any{}
	}
	return config
}
