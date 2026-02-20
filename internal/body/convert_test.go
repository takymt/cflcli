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
		if !strings.Contains(got, `<span style="text-decoration: line-through;">hoge</span>`) {
			t.Fatalf("missing converted strike span: %q", got)
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

	t.Run("markdown link converts to storage anchor tag", func(t *testing.T) {
		got, err := ToStorage([]byte(`[アンカーテキスト](https://developer.atlassian.com/cloud/confluence/)`), "markdown")
		if err != nil {
			t.Fatalf("ToStorage: %v", err)
		}
		if !strings.Contains(got, `<a href="https://developer.atlassian.com/cloud/confluence/">アンカーテキスト</a>`) {
			t.Fatalf("missing storage anchor tag: %q", got)
		}
		if strings.Contains(got, "<ac:link>") {
			t.Fatalf("unexpected ac:link macro for normal markdown link: %q", got)
		}
	})

	t.Run("markdown image converts to confluence image", func(t *testing.T) {
		got, err := ToStorage([]byte(`![alt-text](https://developer.atlassian.com/favicon.ico)`), "markdown")
		if err != nil {
			t.Fatalf("ToStorage: %v", err)
		}
		if !strings.Contains(got, `<ac:image`) {
			t.Fatalf("missing image tag: %q", got)
		}
		if !strings.Contains(got, `ac:alt="alt-text"`) {
			t.Fatalf("missing image alt: %q", got)
		}
		if !strings.Contains(got, `ri:url ri:value="https://developer.atlassian.com/favicon.ico"`) {
			t.Fatalf("missing image url: %q", got)
		}
		if strings.Contains(got, "<img ") {
			t.Fatalf("raw img tag remains: %q", got)
		}
	})

	t.Run("markdown task list converts to confluence task list", func(t *testing.T) {
		input := strings.Join([]string{
			"- [ ] タスク1",
			"- [x] タスク2",
			"- [×] これはタスクリストにしない",
		}, "\n")
		got, err := ToStorage([]byte(input), "markdown")
		if err != nil {
			t.Fatalf("ToStorage: %v", err)
		}
		if !strings.Contains(got, "<ac:task-list>") {
			t.Fatalf("missing task list macro: %q", got)
		}
		if strings.Count(got, "<ac:task-status>") != 2 {
			t.Fatalf("unexpected task count: %q", got)
		}
		if !strings.Contains(got, "<ac:task-status>incomplete</ac:task-status>") {
			t.Fatalf("missing incomplete task status: %q", got)
		}
		if !strings.Contains(got, "<ac:task-status>complete</ac:task-status>") {
			t.Fatalf("missing complete task status: %q", got)
		}
		if !strings.Contains(got, "これはタスクリストにしない") {
			t.Fatalf("non task item was removed: %q", got)
		}
	})

	t.Run("url only line converts to block link card", func(t *testing.T) {
		got, err := ToStorage([]byte("https://zenn.dev/zenn/articles/markdown-guide"), "markdown")
		if err != nil {
			t.Fatalf("ToStorage: %v", err)
		}
		if !strings.Contains(got, `ac:card-appearance="block"`) {
			t.Fatalf("missing block card appearance: %q", got)
		}
		if !strings.Contains(got, `ri:url ri:value="https://zenn.dev/zenn/articles/markdown-guide"`) {
			t.Fatalf("missing block card url: %q", got)
		}
		if strings.Contains(got, "<ac:plain-text-link-body>") {
			t.Fatalf("unexpected plain-text link body for block card: %q", got)
		}
		if strings.Contains(got, "<a href=") {
			t.Fatalf("unexpected plain anchor for URL-only line: %q", got)
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
		if !strings.Contains(got, `<ac:emoticon ac:name="thumbs-up" />`) {
			t.Fatalf("missing thumbs-up emoji: %q", got)
		}
		if !strings.Contains(got, ":unknown_emoji:") {
			t.Fatalf("unknown emoji should remain literal: %q", got)
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

	t.Run("nested quote depth is preserved", func(t *testing.T) {
		got, err := ToStorage([]byte("> 親引用\n>> 子引用"), "markdown")
		if err != nil {
			t.Fatalf("ToStorage: %v", err)
		}
		if strings.Count(got, "<blockquote>") < 2 {
			t.Fatalf("missing blockquote: %q", got)
		}
		if !strings.Contains(got, "<blockquote>\n<p>子引用</p>\n</blockquote>") {
			t.Fatalf("nested blockquote was not preserved: %q", got)
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

	t.Run("markdown core syntax", func(t *testing.T) {
		input := strings.Join([]string{
			"# 見出し1",
			"## 見出し2",
			"### 見出し3",
			"#### 見出し4",
			"",
			"- foo",
			"  - nested",
			"- bar",
			"",
			"1. first",
			"2. second",
			"",
			"> quote",
			">> nested quote",
			"",
			"---",
			"",
			"*italic*",
			"**bold**",
			"~~strike~~",
			"inline `code`",
			`\*escaped\*`,
			"<u>underline</u>",
		}, "\n")

		got, err := ToStorage([]byte(input), "markdown")
		if err != nil {
			t.Fatalf("ToStorage: %v", err)
		}

		wantContains := []string{
			"<h1>見出し1</h1>",
			"<h2>見出し2</h2>",
			"<h3>見出し3</h3>",
			"<h4>見出し4</h4>",
			"<ul>",
			"<ol>",
			"<blockquote>",
			"<hr />",
			"<em>italic</em>",
			"<strong>bold</strong>",
			`<span style="text-decoration: line-through;">strike</span>`,
			"<code>code</code>",
			"*escaped*",
			"<u>underline</u>",
		}
		for _, token := range wantContains {
			if !strings.Contains(got, token) {
				t.Fatalf("output missing %q in %q", token, got)
			}
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
		"<h2>見出し2</h2>",
		"<h3>見出し3</h3>",
		"<h4>見出し4</h4>",
		"<ul>",
		"<ol>",
		"<a href=\"https://developer.atlassian.com/cloud/confluence/\">アンカーテキスト</a>",
		"<ac:image ac:alt=\"alt-text\"><ri:url ri:value=\"https://developer.atlassian.com/favicon.ico\" /></ac:image>",
		"<ac:task-list>",
		"<ac:task-status>incomplete</ac:task-status>",
		"<ac:task-status>complete</ac:task-status>",
		"<blockquote>",
		"<hr />",
		"<em>イタリック</em>",
		"<strong>太字</strong>",
		`<span style="text-decoration: line-through;">打ち消し線</span>`,
		"<code>code</code>",
		`ac:card-appearance="block"`,
		`<ac:emoticon ac:name="smile" />`,
		`<ac:emoticon ac:name="thumbs-up" />`,
		`<ac:structured-macro ac:name="code">`,
		`<ac:parameter ac:name="language">js</ac:parameter>`,
		`<ac:structured-macro ac:name="expand">`,
		`<ac:parameter ac:name="expanded">false</ac:parameter>`,
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
