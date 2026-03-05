package page

import (
	"strings"
	"testing"
)

func TestConvertMarkdownToStorage(t *testing.T) {
	t.Parallel()

	input := strings.Join([]string{
		"# Heading 1",
		"",
		"## Heading 2",
		"",
		"Paragraph text.",
		"",
		"- one",
		"- two",
		"",
		"1. first",
		"2. second",
		"",
		"```go",
		"fmt.Println(\"hello\")",
		"```",
		"",
	}, "\n")

	got, err := ConvertMarkdownToStorage(input)
	if err != nil {
		t.Fatalf("ConvertMarkdownToStorage() error = %v", err)
	}

	for _, want := range []string{
		"<h1>Heading 1</h1>",
		"<h2>Heading 2</h2>",
		"<p>Paragraph text.</p>",
		"<ul><li><p>one</p></li><li><p>two</p></li></ul>",
		"<ol><li><p>first</p></li><li><p>second</p></li></ol>",
		`ac:name="code"`,
		"<![CDATA[fmt.Println(\"hello\")",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("ConvertMarkdownToStorage() = %q, missing %q", got, want)
		}
	}
}

func TestConvertMarkdownToStorageEmptyBody(t *testing.T) {
	t.Parallel()

	got, err := ConvertMarkdownToStorage("")
	if err != nil {
		t.Fatalf("ConvertMarkdownToStorage() error = %v", err)
	}

	if got != "" {
		t.Fatalf("ConvertMarkdownToStorage() = %q, want empty string", got)
	}
}
