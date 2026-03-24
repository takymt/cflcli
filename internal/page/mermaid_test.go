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

	result, err := ConvertMarkdownToStorageWithMermaid(context.Background(), path, input)
	if err != nil {
		t.Fatalf("ConvertMarkdownToStorageWithMermaid() error = %v", err)
	}
	got := result.Storage
	generatedPaths := result.Generated
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
		svgPath, ok := generatedPaths[filename]
		if !ok {
			t.Fatalf("generatedPaths missing %s", filename)
		}
		if _, err := os.Stat(svgPath); err != nil {
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

	result, err := ConvertMarkdownToStorageWithMermaid(context.Background(), path, input)
	if err != nil {
		t.Fatalf("ConvertMarkdownToStorageWithMermaid() error = %v", err)
	}
	got := result.Storage
	generatedPaths := result.Generated
	if len(generatedPaths) != 1 {
		t.Fatalf("generatedPaths = %d, want 1", len(generatedPaths))
	}

	want := `<ac:image ac:align="right" ac:layout="align-end" ac:width="900"><ri:attachment ri:filename="mermaid-1.svg" /></ac:image>`
	if !strings.Contains(got, want) {
		t.Fatalf("converted storage = %q, want %q", got, want)
	}
}

func TestConvertMarkdownToStorageWithMermaid_TildeFence(t *testing.T) {
	t.Parallel()

	if _, err := exec.LookPath("mmdc"); err != nil {
		t.Skip("mmdc is required for mermaid conversion test")
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "guide.md")
	input := "~~~mermaid\ngraph TD\nA-->B\n~~~\n"

	result, err := ConvertMarkdownToStorageWithMermaid(context.Background(), path, input)
	if err != nil {
		t.Fatalf("ConvertMarkdownToStorageWithMermaid() error = %v", err)
	}
	if len(result.Generated) != 1 {
		t.Fatalf("generatedPaths = %d, want 1", len(result.Generated))
	}
	if !strings.Contains(result.Storage, `<ac:image ac:align="center" ac:layout="center"><ri:attachment ri:filename="mermaid-1.svg" /></ac:image>`) {
		t.Fatalf("converted storage = %q, want mermaid image macro", result.Storage)
	}
}

func TestConvertMarkdownToStorageWithMermaid_UsesHashCache(t *testing.T) {
	t.Parallel()

	if _, err := exec.LookPath("mmdc"); err != nil {
		t.Skip("mmdc is required for mermaid conversion test")
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "guide.md")
	input := "```mermaid\ngraph TD\nA-->B\n```\n"

	first, err := ConvertMarkdownToStorageWithMermaid(context.Background(), path, input)
	if err != nil {
		t.Fatalf("first ConvertMarkdownToStorageWithMermaid() error = %v", err)
	}
	if len(first.Generated) != 1 {
		t.Fatalf("first generatedPaths = %d, want 1", len(first.Generated))
	}
	if err := first.SaveCache(); err != nil {
		t.Fatalf("first SaveCache() error = %v", err)
	}

	second, err := ConvertMarkdownToStorageWithMermaid(context.Background(), path, input)
	if err != nil {
		t.Fatalf("second ConvertMarkdownToStorageWithMermaid() error = %v", err)
	}
	if len(second.Generated) != 0 {
		t.Fatalf("second generatedPaths = %d, want 0", len(second.Generated))
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

func TestConvertMarkdownToStorageWithMermaid_MixedFenceEscapesMermaid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "backticks escape tilde mermaid",
			input: "```md\n~~~mermaid\ngraph TD\nA-->B\n~~~\n```\n",
			want: []string{
				`<ac:parameter ac:name="language">md</ac:parameter>`,
				`~~~mermaid`,
				`graph TD`,
				`A-->B`,
				`~~~`,
			},
		},
		{
			name:  "tildes escape backtick mermaid",
			input: "~~~md\n```mermaid\ngraph TD\nA-->B\n```\n~~~\n",
			want: []string{
				`<ac:parameter ac:name="language">md</ac:parameter>`,
				"```mermaid",
				`graph TD`,
				`A-->B`,
				"```",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			path := filepath.Join(dir, "guide.md")
			result, err := ConvertMarkdownToStorageWithMermaid(context.Background(), path, tt.input)
			if err != nil {
				t.Fatalf("ConvertMarkdownToStorageWithMermaid() error = %v", err)
			}
			if len(result.Generated) != 0 {
				t.Fatalf("generatedPaths = %d, want 0", len(result.Generated))
			}
			if strings.Contains(result.Storage, "<ac:image") {
				t.Fatalf("converted storage = %q, want literal fenced content", result.Storage)
			}
			for _, want := range tt.want {
				if !strings.Contains(result.Storage, want) {
					t.Fatalf("converted storage = %q, missing %q", result.Storage, want)
				}
			}
		})
	}
}
