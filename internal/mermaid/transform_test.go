package mermaid

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

type fakeRenderer struct {
	renderErr error
	closed    bool
	renders   []string
}

func (f *fakeRenderer) Render(_ context.Context, source string) ([]byte, error) {
	if f.renderErr != nil {
		return nil, f.renderErr
	}
	f.renders = append(f.renders, source)
	return []byte(fmt.Sprintf("<svg>%s</svg>", source)), nil
}

func (f *fakeRenderer) Close() error {
	f.closed = true
	return nil
}

func TestRenderMarkdownFences_NoMermaid(t *testing.T) {
	t.Parallel()

	factoryCalled := false
	input := []byte("# title\n\n```go\nfmt.Println(\"hi\")\n```\n")

	got, cleanup, err := RenderMarkdownFences(context.Background(), input, t.TempDir(), func() (SVGRenderer, error) {
		factoryCalled = true
		return &fakeRenderer{}, nil
	})
	if err != nil {
		t.Fatalf("RenderMarkdownFences: %v", err)
	}
	if cleanup != nil {
		t.Fatalf("cleanup must be nil when no mermaid block exists")
	}
	if factoryCalled {
		t.Fatalf("renderer factory must not be called when no mermaid block exists")
	}
	if !bytes.Equal(got, input) {
		t.Fatalf("output changed unexpectedly:\n%s", got)
	}
}

func TestRenderMarkdownFences_RewritesMermaidBlocksToImages(t *testing.T) {
	t.Parallel()

	renderer := &fakeRenderer{}
	workDir := t.TempDir()
	input := []byte(strings.Join([]string{
		"# Diagrams",
		"",
		"```mermaid",
		"flowchart TD",
		"  A --> B",
		"```",
		"",
		"```mermaid",
		"flowchart TD",
		"  B --> C",
		"```",
	}, "\n"))

	got, cleanup, err := RenderMarkdownFences(context.Background(), input, workDir, func() (SVGRenderer, error) {
		return renderer, nil
	})
	if err != nil {
		t.Fatalf("RenderMarkdownFences: %v", err)
	}
	if cleanup == nil {
		t.Fatalf("cleanup must not be nil when mermaid blocks are rendered")
	}
	t.Cleanup(func() {
		_ = cleanup()
	})

	if !renderer.closed {
		t.Fatalf("renderer must be closed")
	}

	out := string(got)
	pathPattern := regexp.MustCompile(`!\[mermaid-\d+\]\(<([^>]+)\>\)`)
	matches := pathPattern.FindAllStringSubmatch(out, -1)
	if len(matches) != 2 {
		t.Fatalf("expected 2 rendered image references, got %d in %q", len(matches), out)
	}

	for i, match := range matches {
		if len(match) != 2 {
			t.Fatalf("invalid match %v", match)
		}
		path := filepath.Join(workDir, filepath.FromSlash(match[1]))
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Fatalf("ReadFile(%q): %v", path, readErr)
		}
		if !strings.Contains(string(content), "<svg>") {
			t.Fatalf("rendered file does not contain svg: %q", content)
		}
		wantToken := fmt.Sprintf("mermaid-%03d.svg", i+1)
		if !strings.Contains(path, wantToken) {
			t.Fatalf("generated filename %q does not contain %q", path, wantToken)
		}
	}
}

func TestRenderMarkdownFences_RenderError(t *testing.T) {
	t.Parallel()

	input := []byte(strings.Join([]string{
		"```mermaid",
		"flowchart TD",
		"  A --> B",
		"```",
	}, "\n"))

	_, cleanup, err := RenderMarkdownFences(context.Background(), input, t.TempDir(), func() (SVGRenderer, error) {
		return &fakeRenderer{renderErr: fmt.Errorf("boom")}, nil
	})
	if err == nil || !strings.Contains(err.Error(), "render mermaid block 1") {
		t.Fatalf("unexpected error: %v", err)
	}
	if cleanup != nil {
		t.Fatalf("cleanup must be nil on render failure")
	}
}
