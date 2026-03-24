package page

import (
	"strings"
	"testing"
)

func TestConvertMarkdownToStorage_CatalogSupporting(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		contains []string
	}{
		{
			name:  "headings",
			input: "# H1\n## H2\n### H3\n#### H4\n##### H5\n###### H6\n",
			contains: []string{
				"<h1>H1</h1>",
				"<h2>H2</h2>",
				"<h3>H3</h3>",
				"<h4>H4</h4>",
				"<h5>H5</h5>",
				"<h6>H6</h6>",
			},
		},
		{
			name:  "inline styles",
			input: "- **bold**\n- _italic_\n- ~~strike~~\n- this is `code`\n- <sub>sub</sub>\n- <sup>sup</sup>\n- <ins>ins</ins>\n",
			contains: []string{
				"<strong>bold</strong>",
				"<em>italic</em>",
				`text-decoration: line-through`,
				"<code>code</code>",
				"<sub>sub</sub>",
				"<sup>sup</sup>",
				"<ins>ins</ins>",
			},
		},
		{
			name:  "unordered and ordered lists nested",
			input: "- one\n  - child\n1. first\n2. second\n   1. second-first\n",
			contains: []string{
				"<ul>",
				"<ol>",
				"<li><p>child</p></li>",
				"<li><p>second-first</p></li>",
			},
		},
		{
			name:  "links and autolinks",
			input: "[Anchor text](https://developer.atlassian.com/cloud/confluence/)\nhttps://zenn.dev/zenn/articles/markdown-guide\n",
			contains: []string{
				`<a href="https://developer.atlassian.com/cloud/confluence/">Anchor text</a>`,
				`<a href="https://zenn.dev/zenn/articles/markdown-guide" data-card-appearance="inline">https://zenn.dev/zenn/articles/markdown-guide</a>`,
			},
		},
		{
			name:  "images",
			input: "![alt-text](https://developer.atlassian.com/favicon.ico)\n![local-image](./assets/diagram.png)\n",
			contains: []string{
				`<ac:image ac:alt="alt-text"><ri:url ri:value="https://developer.atlassian.com/favicon.ico" /></ac:image>`,
				`<ac:image ac:alt="local-image"><ri:attachment ri:filename="diagram.png" /></ac:image>`,
			},
		},
		{
			name:  "relative markdown links become attachments",
			input: "[Spec](./docs/spec.pdf)\n[Sibling](../files/sibling.txt)\n",
			contains: []string{
				`<ac:link><ri:attachment ri:filename="spec.pdf" /><ac:plain-text-link-body><![CDATA[Spec]]></ac:plain-text-link-body></ac:link>`,
				`<ac:link><ri:attachment ri:filename="sibling.txt" /><ac:plain-text-link-body><![CDATA[Sibling]]></ac:plain-text-link-body></ac:link>`,
			},
		},
		{
			name:  "emoji shortcode",
			input: ":smile: :warning:\n",
			contains: []string{
				`<ac:emoticon ac:name="smile" />`,
				`<ac:emoticon ac:name="warning" />`,
			},
		},
		{
			name:  "color span",
			input: `<span style="color: rgb(255,0,0);">red text</span>` + "\n",
			contains: []string{
				`<span style="color: rgb(255,0,0);">red text</span>`,
			},
		},
		{
			name: "text align paragraphs",
			input: "" +
				`<p style="text-align: left;">left aligned text</p>` + "\n" +
				`<p style="text-align: center;">centered text</p>` + "\n" +
				`<p style="text-align: right;">right aligned text</p>` + "\n",
			contains: []string{
				`<p style="text-align: left;">left aligned text</p>`,
				`<p style="text-align: center;">centered text</p>`,
				`<p style="text-align: right;">right aligned text</p>`,
			},
		},
		{
			name: "alerts mapping with emoji in body",
			input: "" +
				":::info\ninformation :information:\n:::\n\n" +
				":::note\nnote\n:::\n\n" +
				":::success\nsuccess\n:::\n\n" +
				":::warning\nwarning\n:::\n\n" +
				":::error\nerror :warning:\n:::\n",
			contains: []string{
				`<ac:structured-macro ac:name="info"><ac:rich-text-body>`,
				`<ac:adf-extension><ac:adf-node type="panel"><ac:adf-attribute key="panel-type">note</ac:adf-attribute>`,
				`<ac:structured-macro ac:name="tip"><ac:rich-text-body>`,
				`<ac:structured-macro ac:name="note"><ac:rich-text-body>`,
				`<ac:structured-macro ac:name="warning"><ac:rich-text-body>`,
				`<ac:emoticon ac:name="information" />`,
				`<ac:emoticon ac:name="warning" />`,
			},
		},
		{
			name:  "task list",
			input: "- [ ] Task 1\n- [x] Task 2\n",
			contains: []string{
				"<ac:task-list>",
				"<ac:task-status>incomplete</ac:task-status>",
				"<ac:task-status>complete</ac:task-status>",
			},
		},
		{
			name:  "blockquote and rule",
			input: "> Quote\n>> Nested quote\n\n---\n",
			contains: []string{
				"<blockquote>",
				"<p>Quote</p>",
				"<p>Nested quote</p>",
				"<hr />",
			},
		},
		{
			name:  "line breaks in paragraph",
			input: "This example\nWill span two lines\n",
			contains: []string{
				"<p>This example<br />Will span two lines</p>",
			},
		},
		{
			name:  "table",
			input: "| Head | Head |\n| ---- | ---- |\n| Text | Text |\n",
			contains: []string{
				"<table><tbody>",
				"<th>Head</th>",
				"<td>Text</td>",
			},
		},
		{
			name:  "fenced code block",
			input: "```javascript\nconst codeBlock = \"this is code block\";\n[Spec](./docs/spec.pdf)\n![img](./assets/diagram.png)\n```\n",
			contains: []string{
				`ac:name="code"`,
				`<ac:parameter ac:name="language">javascript</ac:parameter>`,
				`<![CDATA[const codeBlock = "this is code block";`,
				`[Spec](./docs/spec.pdf)`,
				`![img](./assets/diagram.png)`,
			},
		},
		{
			name:  "tilde fenced code block",
			input: "~~~javascript\nconst codeBlock = \"this is tilde code block\";\n~~~\n",
			contains: []string{
				`ac:name="code"`,
				`<ac:parameter ac:name="language">javascript</ac:parameter>`,
				`<![CDATA[const codeBlock = "this is tilde code block";`,
			},
		},
		{
			name:  "details expand",
			input: "<details><summary>title</summary>\n- Collapsed body line 1\n- Collapsed body line 2\n</details>\n",
			contains: []string{
				`<ac:structured-macro ac:name="expand">`,
				`<ac:parameter ac:name="title">title</ac:parameter>`,
				`Collapsed body line 1`,
			},
		},
		{
			name:  "inline comments ignored",
			input: "<!-- TODO: add details about this section -->\nVisible text\n",
			contains: []string{
				"<p>Visible text</p>",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ConvertMarkdownToStorage(tt.input)
			if err != nil {
				t.Fatalf("ConvertMarkdownToStorage() error = %v", err)
			}
			for _, want := range tt.contains {
				if !strings.Contains(got, want) {
					t.Fatalf("ConvertMarkdownToStorage() = %q, missing %q", got, want)
				}
			}
		})
	}
}

func TestConvertMarkdownToStorage_EmptyBody(t *testing.T) {
	t.Parallel()

	got, err := ConvertMarkdownToStorage("")
	if err != nil {
		t.Fatalf("ConvertMarkdownToStorage() error = %v", err)
	}

	if got != "" {
		t.Fatalf("ConvertMarkdownToStorage() = %q, want empty string", got)
	}
}

func TestConvertMarkdownToStorage_MixedFenceEscapes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "backticks escape tilde fence",
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
			name:  "tildes escape backtick fence",
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

			got, err := ConvertMarkdownToStorage(tt.input)
			if err != nil {
				t.Fatalf("ConvertMarkdownToStorage() error = %v", err)
			}
			if strings.Contains(got, "<ac:image") {
				t.Fatalf("ConvertMarkdownToStorage() = %q, want literal fenced content", got)
			}
			for _, want := range tt.want {
				if !strings.Contains(got, want) {
					t.Fatalf("ConvertMarkdownToStorage() = %q, missing %q", got, want)
				}
			}
		})
	}
}

func TestConvertMarkdownToStorage_MultiLineComments(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		contains []string
		excludes []string
	}{
		{
			name:  "comment between paragraphs",
			input: "Before\n<!--\nThis is a\nmulti-line comment\n-->\nAfter\n",
			contains: []string{
				"<p>Before</p>",
				"<p>After</p>",
			},
			excludes: []string{
				"This is a",
				"multi-line comment",
				"&lt;!--",
				"--&gt;",
			},
		},
		{
			name:  "adjacent comments at document boundaries",
			input: "<!--\nleading comment\n-->\nVisible text\n<!--\ntrailing comment\n-->\n",
			contains: []string{
				"<p>Visible text</p>",
			},
			excludes: []string{
				"leading comment",
				"trailing comment",
				"&lt;!--",
				"--&gt;",
			},
		},
		{
			name:  "comments inside fenced code remain literal",
			input: "```md\n<!--\ncomment in code\n-->\n```\n",
			contains: []string{
				`ac:name="code"`,
				`<![CDATA[<!--`,
				"comment in code",
				"-->",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ConvertMarkdownToStorage(tt.input)
			if err != nil {
				t.Fatalf("ConvertMarkdownToStorage() error = %v", err)
			}
			for _, want := range tt.contains {
				if !strings.Contains(got, want) {
					t.Fatalf("ConvertMarkdownToStorage() = %q, missing %q", got, want)
				}
			}
			for _, notWant := range tt.excludes {
				if strings.Contains(got, notWant) {
					t.Fatalf("ConvertMarkdownToStorage() = %q, unexpected %q", got, notWant)
				}
			}
		})
	}
}

func TestParseHeading(t *testing.T) {
	t.Parallel()

	level, text, ok := parseHeading("### title")
	if !ok || level != 3 || text != "title" {
		t.Fatalf("parseHeading() = (%d, %q, %v), want (3, %q, true)", level, text, ok, "title")
	}
}
