package cmd

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/takymt/cflcli/internal/client"
	"github.com/takymt/cflcli/internal/config"
	"github.com/takymt/cflcli/internal/mermaid"
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

type fakeMermaidRenderer struct {
	renderFn    func(context.Context, string) ([]byte, error)
	closeCalled bool
}

func (f *fakeMermaidRenderer) Render(ctx context.Context, source string) ([]byte, error) {
	if f.renderFn == nil {
		return []byte("<svg></svg>"), nil
	}
	return f.renderFn(ctx, source)
}

func (f *fakeMermaidRenderer) Close() error {
	f.closeCalled = true
	return nil
}

func setMermaidRendererFactory(t *testing.T, factory func() (mermaid.SVGRenderer, error)) {
	t.Helper()

	original := newMermaidRenderer
	newMermaidRenderer = factory
	t.Cleanup(func() {
		newMermaidRenderer = original
	})
}
