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

	t.Run("markdown code fence without language defaults to text", func(t *testing.T) {
		got, err := ToStorage([]byte("```\nconst x = 1;\n```"), "markdown")
		if err != nil {
			t.Fatalf("ToStorage: %v", err)
		}
		if !strings.Contains(got, `<ac:structured-macro ac:name="code">`) {
			t.Fatalf("missing code macro: %q", got)
		}
		if !strings.Contains(got, `<ac:parameter ac:name="language">text</ac:parameter>`) {
			t.Fatalf("missing default text language: %q", got)
		}
		if strings.Contains(got, "const x = 1;\n]]>") {
			t.Fatalf("unexpected trailing newline in code block: %q", got)
		}
	})

	t.Run("url only line inside code fence stays code", func(t *testing.T) {
		got, err := ToStorage([]byte("```txt\nhttps://zenn.dev/zenn/articles/markdown-guide\n```"), "markdown")
		if err != nil {
			t.Fatalf("ToStorage: %v", err)
		}
		if strings.Contains(got, "<a href=") {
			t.Fatalf("unexpected anchor tag in code fence: %q", got)
		}
		if !strings.Contains(got, `<![CDATA[https://zenn.dev/zenn/articles/markdown-guide]]>`) {
			t.Fatalf("url in code fence must stay literal: %q", got)
		}
	})

	t.Run("emoji shortcodes convert to confluence emoticons", func(t *testing.T) {
		got, err := ToStorage([]byte(":smile: :thumbsup: :unknown_emoji:"), "markdown")
		if err != nil {
			t.Fatalf("ToStorage: %v", err)
		}
		if !strings.Contains(got, `<ac:emoticon ac:name="smile" />`) {
			t.Fatalf("missing smile emoji: %q", got)
		}
		if !strings.Contains(got, `<ac:emoticon ac:name="thumbsup" />`) {
			t.Fatalf("missing thumbsup emoji: %q", got)
		}
		if !strings.Contains(got, `<ac:emoticon ac:name="unknown_emoji" />`) {
			t.Fatalf("unknown emoji must be converted as-is: %q", got)
		}
	})

	t.Run("emoji shortcode in code block stays literal", func(t *testing.T) {
		got, err := ToStorage([]byte("```\n:smile:\n```"), "markdown")
		if err != nil {
			t.Fatalf("ToStorage: %v", err)
		}
		if !strings.Contains(got, `<![CDATA[:smile:]]>`) {
			t.Fatalf("emoji in code block was transformed unexpectedly: %q", got)
		}
	})

	t.Run("details block converts to collapsed expand macro", func(t *testing.T) {
		input := strings.Join([]string{
			":::details 折りたたみタイトル",
			"折りたたみ本文1",
			"折りたたみ本文2",
			":::",
		}, "\n")
		got, err := ToStorage([]byte(input), "markdown")
		if err != nil {
			t.Fatalf("ToStorage: %v", err)
		}
		if !strings.Contains(got, `<ac:structured-macro ac:name="expand">`) {
			t.Fatalf("missing expand macro: %q", got)
		}
		if !strings.Contains(got, `<ac:parameter ac:name="title">折りたたみタイトル</ac:parameter>`) {
			t.Fatalf("missing expand title: %q", got)
		}
		if !strings.Contains(got, `<ac:parameter ac:name="expanded">false</ac:parameter>`) {
			t.Fatalf("expand default state must be collapsed: %q", got)
		}
		if !strings.Contains(got, "<p>折りたたみ本文1\n折りたたみ本文2</p>") {
			t.Fatalf("missing expand body: %q", got)
		}
	})

	t.Run("admonition blocks convert to confluence macros", func(t *testing.T) {
		input := strings.Join([]string{
			":::info",
			"info",
			":::",
			"",
			":::memo",
			"memo",
			":::",
			"",
			":::success",
			"success",
			":::",
			"",
			":::warn",
			"warn",
			":::",
			"",
			":::error",
			"error",
			":::",
		}, "\n")
		got, err := ToStorage([]byte(input), "markdown")
		if err != nil {
			t.Fatalf("ToStorage: %v", err)
		}

		if !strings.Contains(got, `<ac:structured-macro ac:name="info"><ac:rich-text-body><p>info</p>`) {
			t.Fatalf("missing info macro: %q", got)
		}
		if !strings.Contains(got, `<ac:adf-extension><ac:adf-node type="panel"><ac:adf-attribute key="panel-type">note</ac:adf-attribute><ac:adf-content><p>memo</p>`) {
			t.Fatalf("missing note panel adf-extension for memo: %q", got)
		}
		if !strings.Contains(got, `<ac:structured-macro ac:name="tip"><ac:rich-text-body><p>success</p>`) {
			t.Fatalf("missing tip macro for success: %q", got)
		}
		if !strings.Contains(got, `<ac:structured-macro ac:name="note"><ac:rich-text-body><p>warn</p>`) {
			t.Fatalf("missing note macro for warn: %q", got)
		}
		if strings.Count(got, `<ac:structured-macro ac:name="warning">`) != 1 {
			t.Fatalf("expected one warning macro for error: %q", got)
		}
		if !strings.Contains(got, `<ac:structured-macro ac:name="warning"><ac:rich-text-body><p>error</p>`) {
			t.Fatalf("missing warning macro for error: %q", got)
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
