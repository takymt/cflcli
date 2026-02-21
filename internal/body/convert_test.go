package body

import (
	"os"
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
		if strings.Contains(got, "fmt.Println(\"hi\")\n]]>") {
			t.Fatalf("unexpected trailing newline in code block: %q", got)
		}
	})

	t.Run("details block without title uses default title", func(t *testing.T) {
		input := strings.Join([]string{
			":::details",
			"body",
			":::",
		}, "\n")
		got, err := ToStorage([]byte(input), "markdown")
		if err != nil {
			t.Fatalf("ToStorage: %v", err)
		}
		if !strings.Contains(got, `<ac:structured-macro ac:name="expand">`) {
			t.Fatalf("missing expand macro: %q", got)
		}
		if !strings.Contains(got, `<ac:parameter ac:name="title">Details</ac:parameter>`) {
			t.Fatalf("missing default expand title: %q", got)
		}
		if !strings.Contains(got, `<ac:parameter ac:name="expanded">false</ac:parameter>`) {
			t.Fatalf("expand default state must be collapsed: %q", got)
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
}

func TestToStorage_MarkdownToStorageFixture(t *testing.T) {
	t.Parallel()

	fixture, err := os.ReadFile("testdata/markdown_to_storage_fixture.md")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	got, err := ToStorage(fixture, "markdown")
	if err != nil {
		t.Fatalf("ToStorage: %v", err)
	}

	wantContains := []string{
		"<h1>見出し1</h1>",
		`<span style="text-decoration: line-through;">打ち消し線</span>`,
		"<a href=\"https://developer.atlassian.com/cloud/confluence/\">アンカーテキスト</a>",
		"<ac:image ac:alt=\"alt-text\"><ri:url ri:value=\"https://developer.atlassian.com/favicon.ico\" /></ac:image>",
		"<ac:task-list>",
		"<ac:task-status>incomplete</ac:task-status>",
		"<ac:task-status>complete</ac:task-status>",
		`<a href="https://zenn.dev/zenn/articles/markdown-guide" data-card-appearance="inline">https://zenn.dev/zenn/articles/markdown-guide</a>`,
		`<ac:structured-macro ac:name="code">`,
		`<ac:parameter ac:name="language">js</ac:parameter>`,
		`<ac:structured-macro ac:name="expand">`,
		`<ac:adf-extension><ac:adf-node type="panel"><ac:adf-attribute key="panel-type">note</ac:adf-attribute><ac:adf-content><p>メモ</p>`,
		`<ac:structured-macro ac:name="warning"><ac:rich-text-body><p>エラー</p>`,
		`<ac:emoticon ac:name="smile" />`,
		"<u>underline via raw html</u>",
	}

	for _, token := range wantContains {
		if !strings.Contains(got, token) {
			t.Fatalf("output missing %q in %q", token, got)
		}
	}

	if !strings.Contains(got, "[×] これはタスクリストにしない") {
		t.Fatalf("non-task [×] item must remain literal: %q", got)
	}
	if strings.Contains(got, "<img ") {
		t.Fatalf("raw image tag remains: %q", got)
	}
}
