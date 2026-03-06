package page

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestConvertMarkdownToStorageWithMermaid(t *testing.T) {
	t.Parallel()

	if _, err := exec.LookPath("mmdc"); err != nil {
		t.Skip("mmdc is required for mermaid conversion test")
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "guide.md")
	input := "before\n\n```mermaid\ngraph TD\nA-->B\n```\n\nmiddle\n\n```mermaid\ngraph TD\nB-->C\n```\n\nafter\n"

	got, generatedPaths, err := ConvertMarkdownToStorageWithMermaid(context.Background(), path, input, "https://example.test")
	if err != nil {
		t.Fatalf("ConvertMarkdownToStorageWithMermaid() error = %v", err)
	}
	if len(generatedPaths) != 2 {
		t.Fatalf("generatedPaths = %d, want 2", len(generatedPaths))
	}

	if !strings.Contains(got, "<p>before</p>") {
		t.Fatalf("converted storage = %q, want paragraph for 'before'", got)
	}
	if !strings.Contains(got, `<ac:image ac:align="center" ac:layout="center"><ri:attachment ri:filename="mermaid-1.svg" /></ac:image>`) {
		t.Fatalf("converted storage = %q, want mermaid-1 attachment image macro", got)
	}
	if !strings.Contains(got, `<ac:image ac:align="center" ac:layout="center"><ri:attachment ri:filename="mermaid-2.svg" /></ac:image>`) {
		t.Fatalf("converted storage = %q, want mermaid-2 attachment image macro", got)
	}
	if !strings.Contains(got, "<p>middle</p>") {
		t.Fatalf("converted storage = %q, want paragraph for 'middle'", got)
	}
	if !strings.Contains(got, "<p>after</p>") {
		t.Fatalf("converted storage = %q, want paragraph for 'after'", got)
	}

	for _, filename := range []string{"mermaid-1.svg", "mermaid-2.svg"} {
		if _, err := os.Stat(filepath.Join(dir, filename)); err != nil {
			t.Fatalf("expected generated %s: %v", filename, err)
		}
	}
}

func TestConvertMarkdownToStorageWithMermaid_WithWidthAndAlignOptions(t *testing.T) {
	t.Parallel()

	if _, err := exec.LookPath("mmdc"); err != nil {
		t.Skip("mmdc is required for mermaid conversion test")
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "guide.md")
	input := "```mermaid width=900 align=right\ngraph TD\nA-->B\n```\n"

	got, generatedPaths, err := ConvertMarkdownToStorageWithMermaid(context.Background(), path, input, "https://example.test")
	if err != nil {
		t.Fatalf("ConvertMarkdownToStorageWithMermaid() error = %v", err)
	}
	if len(generatedPaths) != 1 {
		t.Fatalf("generatedPaths = %d, want 1", len(generatedPaths))
	}

	want := `<ac:image ac:align="right" ac:layout="align-end" ac:width="900"><ri:attachment ri:filename="mermaid-1.svg" /></ac:image>`
	if !strings.Contains(got, want) {
		t.Fatalf("converted storage = %q, want %q", got, want)
	}
}

func TestConvertMarkdownToStorageWithMermaid_OversizeReturnsError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "guide.md")
	oversized := strings.Repeat("A", maxMermaidBlockChars+1)
	input := "```mermaid\n" + oversized + "\n```\n"

	_, _, err := ConvertMarkdownToStorageWithMermaid(context.Background(), path, input, "https://example.test")
	if err == nil {
		t.Fatal("ConvertMarkdownToStorageWithMermaid() error = nil, want oversize error")
	}
	if !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("error = %q, want exceeds message", err.Error())
	}
}

func TestConvertMarkdownToStorageWithMermaid_RelativeMarkdownLinkToWebUILinkCard(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	currentPath := filepath.Join(dir, "guide.md")
	targetPath := filepath.Join(dir, "child.md")
	target := "---\nspace-key: TEST\npage-id: 123\nparent-id: 200\n---\n# child\n"
	if err := os.WriteFile(targetPath, []byte(target), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	input := "[Child](./child.md)\n"
	got, generatedPaths, err := ConvertMarkdownToStorageWithMermaid(context.Background(), currentPath, input, "https://example.atlassian.net")
	if err != nil {
		t.Fatalf("ConvertMarkdownToStorageWithMermaid() error = %v", err)
	}
	if len(generatedPaths) != 0 {
		t.Fatalf("generatedPaths = %d, want 0", len(generatedPaths))
	}
	want := `<a href="https://example.atlassian.net/wiki/spaces/TEST/pages/123" data-card-appearance="inline">Child</a>`
	if !strings.Contains(got, want) {
		t.Fatalf("converted storage = %q, want %q", got, want)
	}
}
