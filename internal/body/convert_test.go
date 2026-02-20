package body

import (
	"strings"
	"testing"
)

func TestNormalizeFormat(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "markdown", input: "markdown", want: "markdown"},
		{name: "storage", input: "storage", want: "storage"},
		{name: "trim and lower", input: "  MARKDOWN  ", want: "markdown"},
		{name: "invalid", input: "wiki", wantErr: true},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := NormalizeFormat(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("NormalizeFormat: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}

func TestToStorage(t *testing.T) {
	t.Parallel()

	t.Run("markdown", func(t *testing.T) {
		got, err := ToStorage([]byte("hello **world**"), "markdown")
		if err != nil {
			t.Fatalf("ToStorage: %v", err)
		}
		if !strings.Contains(got, "<p>") || !strings.Contains(got, "<strong>world</strong>") {
			t.Fatalf("unexpected converted value: %q", got)
		}
	})

	t.Run("markdown strikethrough", func(t *testing.T) {
		got, err := ToStorage([]byte("~~hoge~~"), "markdown")
		if err != nil {
			t.Fatalf("ToStorage: %v", err)
		}
		if strings.Contains(got, "~~hoge~~") {
			t.Fatalf("unexpected literal strike syntax: %q", got)
		}
		if !strings.Contains(got, "<del>hoge</del>") {
			t.Fatalf("missing converted strike tag: %q", got)
		}
	})

	t.Run("markdown code fence converts to confluence code macro", func(t *testing.T) {
		got, err := ToStorage([]byte("```go\nfmt.Println(\"hi\")\n```"), "markdown")
		if err != nil {
			t.Fatalf("ToStorage: %v", err)
		}
		if !strings.Contains(got, `<ac:structured-macro ac:name="code">`) {
			t.Fatalf("missing code macro: %q", got)
		}
		if !strings.Contains(got, `<ac:parameter ac:name="language">go</ac:parameter>`) {
			t.Fatalf("missing code language: %q", got)
		}
		if !strings.Contains(got, `<![CDATA[fmt.Println("hi")`) {
			t.Fatalf("missing code body: %q", got)
		}
	})

	t.Run("nested list blank line is tightened", func(t *testing.T) {
		got, err := ToStorage([]byte("- parent\n\n  - child\n"), "markdown")
		if err != nil {
			t.Fatalf("ToStorage: %v", err)
		}
		if strings.Contains(got, "<p>parent</p>") {
			t.Fatalf("unexpected loose parent paragraph: %q", got)
		}
		if !strings.Contains(got, "<li>child</li>") {
			t.Fatalf("missing child list item: %q", got)
		}
		if strings.Contains(got, "<li>parent\n<ul>") {
			t.Fatalf("unexpected soft-break-prone list markup: %q", got)
		}
	})

	t.Run("same level list blank lines are tightened", func(t *testing.T) {
		got, err := ToStorage([]byte("- foo\n\n- bar\n"), "markdown")
		if err != nil {
			t.Fatalf("ToStorage: %v", err)
		}
		if strings.Contains(got, "<p>foo</p>") || strings.Contains(got, "<p>bar</p>") {
			t.Fatalf("unexpected loose list paragraphs: %q", got)
		}
		if !strings.Contains(got, "<li>foo</li>") || !strings.Contains(got, "<li>bar</li>") {
			t.Fatalf("missing list items: %q", got)
		}
	})

	t.Run("storage passthrough", func(t *testing.T) {
		got, err := ToStorage([]byte("<p>Hello</p>"), "storage")
		if err != nil {
			t.Fatalf("ToStorage: %v", err)
		}
		if got != "<p>Hello</p>" {
			t.Fatalf("got %q want %q", got, "<p>Hello</p>")
		}
	})
}
