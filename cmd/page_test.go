package cmd

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/takymt/cflcli/internal/client"
	"github.com/takymt/cflcli/internal/config"
)

func setupPageListServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	originalHTTPClient := client.DefaultHTTPClient
	client.DefaultHTTPClient = srv.Client()
	t.Cleanup(func() { client.DefaultHTTPClient = originalHTTPClient })

	return srv
}

func setOutputMode(t *testing.T, mode string) {
	t.Helper()

	originalOutput := OutputFlag()
	SetOutputFlag(mode)
	t.Cleanup(func() { SetOutputFlag(originalOutput) })
}

func newPageListConfig(domain, spaceKey string) *config.Config {
	return &config.Config{
		Current: "work",
		Profiles: []config.Profile{
			{Name: "work", Domain: domain, User: "user@example.com", SpaceKey: spaceKey},
		},
	}
}

func writeTempBodyFile(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "body.xhtml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write body file: %v", err)
	}
	return path
}

func writeRepoConfig(t *testing.T, dir, content string) string {
	t.Helper()

	path := filepath.Join(dir, "cfl.toml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write repo config: %v", err)
	}
	return path
}

func chdir(t *testing.T, dir string) {
	t.Helper()

	original, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir(%q): %v", dir, err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(original)
	})
}
