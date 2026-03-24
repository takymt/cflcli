package page

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveRelativeReferences_RelativeMarkdownLinkToWebUILinkCard(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	currentPath := filepath.Join(dir, "guide.md")
	targetPath := filepath.Join(dir, "child.md")
	target := "---\ntitle: child\nspace-key: TEST\npage-id: 123\nparent-id: 200\n---\n# child\n"
	if err := os.WriteFile(targetPath, []byte(target), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	got, err := resolveRelativeReferences(currentPath, "[Child](./child.md)\n", "https://example.atlassian.net")
	if err != nil {
		t.Fatalf("resolveRelativeReferences() error = %v", err)
	}

	wantMarkdown := "[Child](https://example.atlassian.net/wiki/spaces/TEST/pages/123)\n"
	if got.Markdown != wantMarkdown {
		t.Fatalf("Markdown = %q, want %q", got.Markdown, wantMarkdown)
	}
	if len(got.AttachmentSources) != 0 {
		t.Fatalf("AttachmentSources = %#v, want empty", got.AttachmentSources)
	}
	if len(got.Warnings) != 0 {
		t.Fatalf("Warnings = %#v, want empty", got.Warnings)
	}
}

func TestResolveRelativeReferences_RelativeMarkdownLinkPreservesQueryAndFragment(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	currentPath := filepath.Join(dir, "guide.md")
	targetPath := filepath.Join(dir, "child.md")
	target := "---\ntitle: child\nspace-key: TEST\npage-id: 123\nparent-id: 200\n---\n# child\n"
	if err := os.WriteFile(targetPath, []byte(target), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	got, err := resolveRelativeReferences(currentPath, "[Child](./child.md?mode=preview#section)\n", "https://example.atlassian.net")
	if err != nil {
		t.Fatalf("resolveRelativeReferences() error = %v", err)
	}

	wantMarkdown := "[Child](https://example.atlassian.net/wiki/spaces/TEST/pages/123?mode=preview#section)\n"
	if got.Markdown != wantMarkdown {
		t.Fatalf("Markdown = %q, want %q", got.Markdown, wantMarkdown)
	}
	if len(got.AttachmentSources) != 0 {
		t.Fatalf("AttachmentSources = %#v, want empty", got.AttachmentSources)
	}
}

func TestResolveRelativeReferences_RelativeMarkdownLinkWithInvalidFrontmatterFallsBackToAttachment(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	currentPath := filepath.Join(dir, "guide.md")
	targetPath := filepath.Join(dir, "child.md")
	target := "---\ntitle: child\nspace-key: [TEST\npage-id: 123\nparent-id: 200\n---\n# child\n"
	if err := os.WriteFile(targetPath, []byte(target), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	got, err := resolveRelativeReferences(currentPath, "[Child](./child.md)\n", "https://example.atlassian.net")
	if err != nil {
		t.Fatalf("resolveRelativeReferences() error = %v", err)
	}

	wantMarkdown := "[Child](./child.md)\n"
	if got.Markdown != wantMarkdown {
		t.Fatalf("Markdown = %q, want %q", got.Markdown, wantMarkdown)
	}
	wantSource := filepath.Join(dir, "child.md")
	if got.AttachmentSources["child.md"] != wantSource {
		t.Fatalf("AttachmentSources[child.md] = %q, want %q", got.AttachmentSources["child.md"], wantSource)
	}
	if len(got.Warnings) != 0 {
		t.Fatalf("Warnings = %#v, want empty", got.Warnings)
	}
}

func TestResolveRelativeReferences_RelativeMarkdownLinkWithoutFrontmatterFallsBackToAttachment(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	currentPath := filepath.Join(dir, "guide.md")
	targetPath := filepath.Join(dir, "child.md")
	if err := os.WriteFile(targetPath, []byte("# child\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	got, err := resolveRelativeReferences(currentPath, "[Child](./child.md)\n", "https://example.atlassian.net")
	if err != nil {
		t.Fatalf("resolveRelativeReferences() error = %v", err)
	}

	wantMarkdown := "[Child](./child.md)\n"
	if got.Markdown != wantMarkdown {
		t.Fatalf("Markdown = %q, want %q", got.Markdown, wantMarkdown)
	}
	wantSource := filepath.Join(dir, "child.md")
	if got.AttachmentSources["child.md"] != wantSource {
		t.Fatalf("AttachmentSources[child.md] = %q, want %q", got.AttachmentSources["child.md"], wantSource)
	}
}

func TestResolveRelativeReferences_RelativeImageInParentDirectoryAddsAttachmentSource(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	currentPath := filepath.Join(dir, "docs", "guide.md")
	if err := os.MkdirAll(filepath.Join(dir, "docs"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	got, err := resolveRelativeReferences(currentPath, "![Diagram](../assets/diagram.png)\n", "https://example.atlassian.net")
	if err != nil {
		t.Fatalf("resolveRelativeReferences() error = %v", err)
	}

	wantSource := filepath.Join(dir, "assets", "diagram.png")
	if got.AttachmentSources["diagram.png"] != wantSource {
		t.Fatalf("AttachmentSources[diagram.png] = %q, want %q", got.AttachmentSources["diagram.png"], wantSource)
	}
	wantMarkdown := "![Diagram](../assets/diagram.png)\n"
	if got.Markdown != wantMarkdown {
		t.Fatalf("Markdown = %q, want %q", got.Markdown, wantMarkdown)
	}
}

func TestResolveRelativeReferences_AbsolutePathsProduceWarningsAndSkipAttachmentSources(t *testing.T) {
	t.Parallel()

	currentPath := filepath.Join(t.TempDir(), "guide.md")
	input := "![Diagram](/tmp/diagram.png)\n[Spec](/tmp/spec.pdf)\n"

	got, err := resolveRelativeReferences(currentPath, input, "https://example.atlassian.net")
	if err != nil {
		t.Fatalf("resolveRelativeReferences() error = %v", err)
	}

	if got.Markdown != input {
		t.Fatalf("Markdown = %q, want %q", got.Markdown, input)
	}
	if len(got.AttachmentSources) != 0 {
		t.Fatalf("AttachmentSources = %#v, want empty", got.AttachmentSources)
	}
	if len(got.Warnings) != 2 {
		t.Fatalf("Warnings = %#v, want 2 warnings", got.Warnings)
	}
}

func TestResolveRelativeReferences_AttachmentFilenameCollisionReturnsError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	currentPath := filepath.Join(dir, "docs", "guide.md")
	if err := os.MkdirAll(filepath.Join(dir, "docs"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	_, err := resolveRelativeReferences(currentPath, "![A](./a/foo.png)\n![B](./b/foo.png)\n", "https://example.atlassian.net")
	if err == nil {
		t.Fatal("resolveRelativeReferences() error = nil, want collision error")
	}
}

func TestResolveRelativeReferences_SameBasenameForSameResolvedFileIsAllowed(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	currentPath := filepath.Join(dir, "docs", "guide.md")
	if err := os.MkdirAll(filepath.Join(dir, "docs"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	got, err := resolveRelativeReferences(currentPath, "![A](./foo.png)\n![B](../docs/foo.png)\n", "https://example.atlassian.net")
	if err != nil {
		t.Fatalf("resolveRelativeReferences() error = %v", err)
	}

	wantSource := filepath.Join(dir, "docs", "foo.png")
	if got.AttachmentSources["foo.png"] != wantSource {
		t.Fatalf("AttachmentSources[foo.png] = %q, want %q", got.AttachmentSources["foo.png"], wantSource)
	}
}
