package cmd

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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
	renderFn func(context.Context, string) ([]byte, error)
}

func (f *fakeMermaidRenderer) Render(ctx context.Context, source string) ([]byte, error) {
	if f.renderFn == nil {
		return []byte("<svg></svg>"), nil
	}
	return f.renderFn(ctx, source)
}

func (f *fakeMermaidRenderer) Close() error {
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

func TestLoadPageStorageBody_MarkdownMermaidProducesAttachmentAsset(t *testing.T) {
	bodyFile := writeTempBodyFile(t, strings.Join([]string{
		"# Diagram",
		"",
		"```mermaid",
		"flowchart TD",
		"  A --> B",
		"```",
	}, "\n"))

	setMermaidRendererFactory(t, func() (mermaid.SVGRenderer, error) {
		return &fakeMermaidRenderer{
			renderFn: func(_ context.Context, source string) ([]byte, error) {
				if !strings.Contains(source, "flowchart TD") {
					t.Fatalf("unexpected mermaid source: %q", source)
				}
				return []byte(`<svg xmlns="http://www.w3.org/2000/svg"><text>diagram</text></svg>`), nil
			},
		}, nil
	})

	input, err := loadPageStorageBody(bodyFile, "markdown", "", false)
	if err != nil {
		t.Fatalf("loadPageStorageBody: %v", err)
	}
	t.Cleanup(func() {
		_ = input.Cleanup()
	})

	if !strings.Contains(input.StorageBody, `<ri:attachment ri:filename="cfl-mermaid-001.svg" />`) {
		t.Fatalf("storage body must reference rendered mermaid attachment: %q", input.StorageBody)
	}
	if len(input.LocalImageAssets) != 1 {
		t.Fatalf("LocalImageAssets len=%d want 1", len(input.LocalImageAssets))
	}
	if input.LocalImageAssets[0].Filename != "cfl-mermaid-001.svg" {
		t.Fatalf("asset filename=%q want %q", input.LocalImageAssets[0].Filename, "cfl-mermaid-001.svg")
	}

	svgContent, err := os.ReadFile(input.LocalImageAssets[0].SourcePath)
	if err != nil {
		t.Fatalf("ReadFile(svg): %v", err)
	}
	if !strings.Contains(string(svgContent), "<svg") {
		t.Fatalf("rendered svg content is invalid: %q", svgContent)
	}
}

func TestLoadPageStorageBody_MarkdownMermaidRenderError(t *testing.T) {
	bodyFile := writeTempBodyFile(t, strings.Join([]string{
		"```mermaid",
		"flowchart TD",
		"  A --> B",
		"```",
	}, "\n"))

	setMermaidRendererFactory(t, func() (mermaid.SVGRenderer, error) {
		return &fakeMermaidRenderer{
			renderFn: func(context.Context, string) ([]byte, error) {
				return nil, fmt.Errorf("boom")
			},
		}, nil
	})

	_, err := loadPageStorageBody(bodyFile, "markdown", "", false)
	if err == nil || !strings.Contains(err.Error(), "render mermaid block 1") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadPageStorageBody_MarkdownNoRenderMermaidKeepsCodeBlock(t *testing.T) {
	bodyFile := writeTempBodyFile(t, strings.Join([]string{
		"# Diagram",
		"",
		"```mermaid",
		"flowchart TD",
		"  A --> B",
		"```",
	}, "\n"))

	setMermaidRendererFactory(t, func() (mermaid.SVGRenderer, error) {
		return &fakeMermaidRenderer{
			renderFn: func(context.Context, string) ([]byte, error) {
				return nil, fmt.Errorf("renderer must not be called when --no-render-mermaid is enabled")
			},
		}, nil
	})

	input, err := loadPageStorageBody(bodyFile, "markdown", "", true)
	if err != nil {
		t.Fatalf("loadPageStorageBody: %v", err)
	}

	if len(input.LocalImageAssets) != 0 {
		t.Fatalf("LocalImageAssets len=%d want 0", len(input.LocalImageAssets))
	}
	if !strings.Contains(input.StorageBody, `<ac:structured-macro ac:name="code">`) {
		t.Fatalf("storage body must keep code macro: %q", input.StorageBody)
	}
	if !strings.Contains(input.StorageBody, `<ac:parameter ac:name="language">mermaid</ac:parameter>`) {
		t.Fatalf("storage body must keep mermaid code language: %q", input.StorageBody)
	}
}
