package migrate

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestStorageToMarkdown_ConvertsMermaidAndAttachments(t *testing.T) {
	storage := strings.Join([]string{
		`<p>Intro</p>`,
		`<ac:image ac:alt="Logo"><ri:attachment ri:filename="logo.png" /></ac:image>`,
		`<ac:structured-macro ac:name="code"><ac:parameter ac:name="language">mermaid</ac:parameter><ac:plain-text-body><![CDATA[flowchart TD`,
		`A --> B`,
		`]]></ac:plain-text-body></ac:structured-macro>`,
		`<ac:structured-macro ac:name="toc"><ac:parameter ac:name="maxLevel">2</ac:parameter></ac:structured-macro>`,
	}, "\n")

	markdown, attachments, err := StorageToMarkdown(storage, func(filename string) string {
		return filepath.ToSlash(filepath.Join("attachments", "_migrate", "123", filename))
	})
	if err != nil {
		t.Fatalf("StorageToMarkdown: %v", err)
	}

	if len(attachments) != 1 || attachments[0] != "logo.png" {
		t.Fatalf("attachments=%v want [logo.png]", attachments)
	}
	if !strings.Contains(markdown, `![Logo](attachments/_migrate/123/logo.png)`) {
		t.Fatalf("missing converted attachment markdown: %q", markdown)
	}
	if !strings.Contains(markdown, "```mermaid") || !strings.Contains(markdown, "flowchart TD") {
		t.Fatalf("missing mermaid code fence conversion: %q", markdown)
	}
	if !strings.Contains(markdown, `cfl:migrate-unsupported-macro`) || !strings.Contains(markdown, `name="toc"`) {
		t.Fatalf("missing unsupported macro comment: %q", markdown)
	}
}

func TestStorageToMarkdown_DeduplicatesAttachmentReferences(t *testing.T) {
	storage := strings.Join([]string{
		`<ac:image><ri:attachment ri:filename="same.png" /></ac:image>`,
		`<ri:attachment ri:filename="same.png" />`,
	}, "\n")

	_, attachments, err := StorageToMarkdown(storage, func(filename string) string {
		return filepath.ToSlash(filepath.Join("attachments", "_migrate", "123", filename))
	})
	if err != nil {
		t.Fatalf("StorageToMarkdown: %v", err)
	}

	if len(attachments) != 1 || attachments[0] != "same.png" {
		t.Fatalf("attachments=%v want [same.png]", attachments)
	}
}

func TestStorageToMarkdown_ConvertsImageURLTag(t *testing.T) {
	storage := `<ac:image ac:alt="diagram"><ri:url ri:value="attachments/_migrate/123/diagram.svg" /></ac:image>`

	markdown, attachments, err := StorageToMarkdown(storage, nil)
	if err != nil {
		t.Fatalf("StorageToMarkdown: %v", err)
	}
	if len(attachments) != 0 {
		t.Fatalf("attachments=%v want empty", attachments)
	}
	if !strings.Contains(markdown, `![diagram](attachments/_migrate/123/diagram.svg)`) {
		t.Fatalf("markdown=%q", markdown)
	}
}

func TestStorageToMarkdown_ConvertsCommonHTMLToMarkdownSyntax(t *testing.T) {
	storage := strings.Join([]string{
		`<h2>見出し</h2>`,
		`<p><strong>太字</strong> と <em>斜体</em> と <code>code</code> と <span style="text-decoration: line-through;">取消</span></p>`,
		`<p><a href="https://example.com/docs">リンク</a></p>`,
		`<blockquote><p>引用</p><p>ネスト前</p></blockquote>`,
		`<ul><li>one</li><li>two<ul><li>nested</li></ul></li></ul>`,
		`<ol><li>first</li><li>second</li></ol>`,
		`<hr />`,
		`<ac:task-list><ac:task><ac:task-status>incomplete</ac:task-status><ac:task-body>タスク1</ac:task-body></ac:task><ac:task><ac:task-status>complete</ac:task-status><ac:task-body>タスク2</ac:task-body></ac:task></ac:task-list>`,
		`<ac:emoticon ac:name="smile" />`,
	}, "\n")

	markdown, _, err := StorageToMarkdown(storage, nil)
	if err != nil {
		t.Fatalf("StorageToMarkdown: %v", err)
	}

	for _, want := range []string{
		`## 見出し`,
		`**太字**`,
		`*斜体*`,
		"`code`",
		`~~取消~~`,
		`[リンク](https://example.com/docs)`,
		`> 引用`,
		`- one`,
		`- two`,
		`- nested`,
		`1. first`,
		`2. second`,
		`---`,
		`- [ ] タスク1`,
		`- [x] タスク2`,
		`:smile:`,
	} {
		if !strings.Contains(markdown, want) {
			t.Fatalf("markdown missing %q\nmarkdown=%q", want, markdown)
		}
	}
}

func TestStorageToMarkdown_ConvertsNonMermaidCodeMacroToFence(t *testing.T) {
	storage := `<ac:structured-macro ac:name="code"><ac:parameter ac:name="language">js</ac:parameter><ac:plain-text-body><![CDATA[const x = 1;]]></ac:plain-text-body></ac:structured-macro>`

	markdown, _, err := StorageToMarkdown(storage, nil)
	if err != nil {
		t.Fatalf("StorageToMarkdown: %v", err)
	}

	if !strings.Contains(markdown, "```js") || !strings.Contains(markdown, "const x = 1;") {
		t.Fatalf("markdown=%q", markdown)
	}
	if strings.Contains(markdown, "cfl:migrate-unsupported-macro") {
		t.Fatalf("code macro should not fallback to unsupported comment: %q", markdown)
	}
}
