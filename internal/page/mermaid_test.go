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
	input := "before\n\n```mermaid\ngraph TD\nA-->B\n```\n\nafter\n"

	got, err := ConvertMarkdownToStorageWithMermaid(context.Background(), path, input)
	if err != nil {
		t.Fatalf("ConvertMarkdownToStorageWithMermaid() error = %v", err)
	}

	if !strings.Contains(got, "<p>before</p>") {
		t.Fatalf("converted storage = %q, want paragraph for 'before'", got)
	}
	if !strings.Contains(got, `<ac:image><ri:attachment ri:filename="mermaid-`) {
		t.Fatalf("converted storage = %q, want mermaid attachment image macro", got)
	}
	if !strings.Contains(got, "<p>after</p>") {
		t.Fatalf("converted storage = %q, want paragraph for 'after'", got)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	foundSVG := false
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "mermaid-") && strings.HasSuffix(entry.Name(), ".svg") {
			foundSVG = true
			break
		}
	}
	if !foundSVG {
		t.Fatal("expected generated mermaid-*.svg file")
	}
}
