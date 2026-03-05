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

	got, err := ConvertMarkdownToStorageWithMermaid(context.Background(), path, input)
	if err != nil {
		t.Fatalf("ConvertMarkdownToStorageWithMermaid() error = %v", err)
	}

	if !strings.Contains(got, "<p>before</p>") {
		t.Fatalf("converted storage = %q, want paragraph for 'before'", got)
	}
	if !strings.Contains(got, `<ac:image><ri:attachment ri:filename="mermaid-1.svg" /></ac:image>`) {
		t.Fatalf("converted storage = %q, want mermaid-1 attachment image macro", got)
	}
	if !strings.Contains(got, `<ac:image><ri:attachment ri:filename="mermaid-2.svg" /></ac:image>`) {
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

func TestConvertMarkdownToStorageWithMermaid_OversizeReturnsError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "guide.md")
	oversized := strings.Repeat("A", maxMermaidBlockChars+1)
	input := "```mermaid\n" + oversized + "\n```\n"

	_, err := ConvertMarkdownToStorageWithMermaid(context.Background(), path, input)
	if err == nil {
		t.Fatal("ConvertMarkdownToStorageWithMermaid() error = nil, want oversize error")
	}
	if !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("error = %q, want exceeds message", err.Error())
	}
}
