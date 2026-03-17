package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func TestXDGConfigStoreSaveCreatesConfigYMLAndOverwritesCredentialKeys(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	store := NewXDGConfigStore()
	path, err := store.Path()
	if err != nil {
		t.Fatalf("Path() error = %v", err)
	}
	if filepath.Base(path) != "config.yml" {
		t.Fatalf("config path = %q, want config.yml", path)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, []byte("other: keep\nemail: old@example.com\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := store.Save(Credentials{
		Domain:   "https://example.atlassian.net/wiki",
		Email:    "user@example.com",
		APIToken: "secret",
	}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	var config map[string]any
	if err := yaml.Unmarshal(raw, &config); err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}

	if got := config["other"]; got != "keep" {
		t.Fatalf("other = %#v, want %q", got, "keep")
	}
	if got := config["domain"]; got != "https://example.atlassian.net/wiki" {
		t.Fatalf("domain = %#v, want %q", got, "https://example.atlassian.net/wiki")
	}
	if got := config["email"]; got != "user@example.com" {
		t.Fatalf("email = %#v, want %q", got, "user@example.com")
	}
	if got := config["api_token"]; got != "secret" {
		t.Fatalf("api_token = %#v, want %q", got, "secret")
	}
}

func TestXDGConfigStoreClearDeletesOnlyCredentialKeys(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	store := NewXDGConfigStore()
	path, err := store.Path()
	if err != nil {
		t.Fatalf("Path() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, []byte("other: keep\ndomain: d\nemail: e\napi_token: t\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := store.Clear(); err != nil {
		t.Fatalf("Clear() error = %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	var config map[string]any
	if err := yaml.Unmarshal(raw, &config); err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}

	if got := config["other"]; got != "keep" {
		t.Fatalf("other = %#v, want %q", got, "keep")
	}
	if _, ok := config["domain"]; ok {
		t.Fatalf("domain key must be deleted, config = %#v", config)
	}
	if _, ok := config["email"]; ok {
		t.Fatalf("email key must be deleted, config = %#v", config)
	}
	if _, ok := config["api_token"]; ok {
		t.Fatalf("api_token key must be deleted, config = %#v", config)
	}
}

func TestXDGConfigStoreClearMissingFileIsSuccess(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	store := NewXDGConfigStore()
	if err := store.Clear(); err != nil {
		t.Fatalf("Clear() error = %v, want nil", err)
	}
}

func TestResolveRuntimeCredentialsUsesEnvPerKey(t *testing.T) {
	t.Setenv("CONFLUENCE_DOMAIN", "env.atlassian.net")
	t.Setenv("CONFLUENCE_EMAIL", "")
	t.Setenv("ATLASSIAN_EMAIL", "env@example.com")
	t.Setenv("CONFLUENCE_API_TOKEN", "")
	t.Setenv("ATLASSIAN_API_TOKEN", "")

	got, err := ResolveRuntimeCredentials(fakeStore{
		loadResult: Credentials{
			Domain:   "config.atlassian.net",
			Email:    "config@example.com",
			APIToken: "config-token",
		},
	})
	if err != nil {
		t.Fatalf("ResolveRuntimeCredentials() error = %v", err)
	}

	if got.Domain != "env.atlassian.net" {
		t.Fatalf("Domain = %q, want %q", got.Domain, "env.atlassian.net")
	}
	if got.Email != "env@example.com" {
		t.Fatalf("Email = %q, want %q", got.Email, "env@example.com")
	}
	if got.APIToken != "config-token" {
		t.Fatalf("APIToken = %q, want %q", got.APIToken, "config-token")
	}
}

func TestResolveValidationCredentialsUsesEnvOverConfigInput(t *testing.T) {
	t.Setenv("CONFLUENCE_DOMAIN", "")
	t.Setenv("CONFLUENCE_EMAIL", "env@example.com")
	t.Setenv("CONFLUENCE_API_TOKEN", "")
	t.Setenv("ATLASSIAN_EMAIL", "")
	t.Setenv("ATLASSIAN_API_TOKEN", "")

	got, err := ResolveValidationCredentials(Credentials{
		Domain:   "config.atlassian.net",
		Email:    "config@example.com",
		APIToken: "config-token",
	})
	if err != nil {
		t.Fatalf("ResolveValidationCredentials() error = %v", err)
	}

	if got.Domain != "config.atlassian.net" {
		t.Fatalf("Domain = %q, want %q", got.Domain, "config.atlassian.net")
	}
	if got.Email != "env@example.com" {
		t.Fatalf("Email = %q, want %q", got.Email, "env@example.com")
	}
	if got.APIToken != "config-token" {
		t.Fatalf("APIToken = %q, want %q", got.APIToken, "config-token")
	}
}

func TestHTTPValidatorUsesFiveSecondTimeout(t *testing.T) {
	validator := NewHTTPValidator(nil)
	if validator.client.Timeout != 5*time.Second {
		t.Fatalf("client timeout = %v, want %v", validator.client.Timeout, 5*time.Second)
	}
}

func TestHTTPValidatorReturnsServerStatusAndMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wiki/api/v2/spaces" {
			t.Fatalf("request path = %q, want %q", r.URL.Path, "/wiki/api/v2/spaces")
		}
		if r.URL.RawQuery != "limit=1" {
			t.Fatalf("request query = %q, want %q", r.URL.RawQuery, "limit=1")
		}
		user, pass, ok := r.BasicAuth()
		if !ok {
			t.Fatal("BasicAuth() ok = false, want true")
		}
		if user != "user@example.com" || pass != "secret" {
			t.Fatalf("BasicAuth() = %q/%q, want user@example.com/secret", user, pass)
		}
		http.Error(w, "invalid token", http.StatusUnauthorized)
	}))
	defer server.Close()

	validator := NewHTTPValidator(server.Client())
	validator.buildURL = func(domain string) string {
		return server.URL + "/wiki/api/v2/spaces?limit=1"
	}

	err := validator.Validate(context.Background(), Credentials{
		Domain:   "example.atlassian.net",
		Email:    "user@example.com",
		APIToken: "secret",
	})
	if err == nil {
		t.Fatal("Validate() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "401 Unauthorized") {
		t.Fatalf("Validate() error = %q, want status code", err.Error())
	}
	if !strings.Contains(err.Error(), "invalid token") {
		t.Fatalf("Validate() error = %q, want server message", err.Error())
	}
}

type fakeStore struct {
	loadResult Credentials
}

func (f fakeStore) Path() (string, error) {
	return "", nil
}

func (f fakeStore) Load() (Credentials, error) {
	return f.loadResult, nil
}

func (f fakeStore) Save(Credentials) error {
	return nil
}

func (f fakeStore) Clear() error {
	return nil
}
