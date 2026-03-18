package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/takymt/cflcli/internal/page"
)

func TestCollectMarkdownFiles_EmptyDir(t *testing.T) {
	t.Parallel()

	root := filepath.Join(t.TempDir(), "docs")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}

	files, skipped, err := collectMarkdownFiles(root, 500)
	if err != nil {
		t.Fatalf("collectMarkdownFiles() error = %v", err)
	}
	if len(files) != 0 {
		t.Fatalf("len(files) = %d, want 0", len(files))
	}
	if len(skipped) != 0 {
		t.Fatalf("len(skipped) = %d, want 0", len(skipped))
	}
}

func TestCollectMarkdownFiles_ValidAndInvalid(t *testing.T) {
	t.Parallel()

	root := filepath.Join(t.TempDir(), "docs")
	mustMkdirAll(t, filepath.Join(root, "nested"))
	writeTestFile(t, filepath.Join(root, "guide-a.md"), []byte("---\nspace-key: TEST\npage-id: 400\nparent-id: 200\n---\n# A\n"))
	writeTestFile(t, filepath.Join(root, "nested", "guide-b.md"), []byte("---\nspace-key: TEST\npage-id: 401\nparent-id: 200\n---\n# B\n"))
	writeTestFile(t, filepath.Join(root, "draft.md"), []byte("# no frontmatter\n"))
	writeTestFile(t, filepath.Join(root, "notes.txt"), []byte("ignored\n"))

	files, skipped, err := collectMarkdownFiles(root, 500)
	if err != nil {
		t.Fatalf("collectMarkdownFiles() error = %v", err)
	}

	if len(files) != 2 {
		t.Fatalf("len(files) = %d, want 2", len(files))
	}
	if files[0].Display != filepath.Join("docs", "guide-a.md") {
		t.Fatalf("files[0].Display = %q, want %q", files[0].Display, filepath.Join("docs", "guide-a.md"))
	}
	if files[1].Display != filepath.Join("docs", "nested", "guide-b.md") {
		t.Fatalf("files[1].Display = %q, want %q", files[1].Display, filepath.Join("docs", "nested", "guide-b.md"))
	}

	if len(skipped) != 1 {
		t.Fatalf("len(skipped) = %d, want 1", len(skipped))
	}
	if skipped[0].Display != filepath.Join("docs", "draft.md") {
		t.Fatalf("skipped[0].Display = %q, want %q", skipped[0].Display, filepath.Join("docs", "draft.md"))
	}
	if skipped[0].Reason != "no valid frontmatter" {
		t.Fatalf("skipped[0].Reason = %q, want %q", skipped[0].Reason, "no valid frontmatter")
	}
}

func TestCollectMarkdownFiles_NestedDirs(t *testing.T) {
	t.Parallel()

	root := filepath.Join(t.TempDir(), "docs")
	mustMkdirAll(t, filepath.Join(root, "product", "api"))
	writeTestFile(t, filepath.Join(root, "product", "guide.md"), []byte("---\nspace-key: TEST\npage-id: 400\nparent-id: 200\n---\nbody\n"))
	writeTestFile(t, filepath.Join(root, "product", "api", "reference.md"), []byte("---\nspace-key: TEST\npage-id: 401\nparent-id: 200\n---\nbody\n"))

	files, skipped, err := collectMarkdownFiles(root, 500)
	if err != nil {
		t.Fatalf("collectMarkdownFiles() error = %v", err)
	}
	if len(skipped) != 0 {
		t.Fatalf("len(skipped) = %d, want 0", len(skipped))
	}
	if len(files) != 2 {
		t.Fatalf("len(files) = %d, want 2", len(files))
	}
	if files[0].Display != filepath.Join("docs", "product", "api", "reference.md") {
		t.Fatalf("files[0].Display = %q, want %q", files[0].Display, filepath.Join("docs", "product", "api", "reference.md"))
	}
	if files[1].Display != filepath.Join("docs", "product", "guide.md") {
		t.Fatalf("files[1].Display = %q, want %q", files[1].Display, filepath.Join("docs", "product", "guide.md"))
	}
}

func TestCollectMarkdownFiles_MaxFilesExceeded(t *testing.T) {
	t.Parallel()

	root := filepath.Join(t.TempDir(), "docs")
	mustMkdirAll(t, root)
	writeTestFile(t, filepath.Join(root, "one.md"), []byte("---\nspace-key: TEST\npage-id: 400\nparent-id: 200\n---\n"))
	writeTestFile(t, filepath.Join(root, "two.md"), []byte("---\nspace-key: TEST\npage-id: 401\nparent-id: 200\n---\n"))
	writeTestFile(t, filepath.Join(root, "three.md"), []byte("---\nspace-key: TEST\npage-id: 402\nparent-id: 200\n---\n"))

	_, _, err := collectMarkdownFiles(root, 2)
	if err == nil {
		t.Fatal("collectMarkdownFiles() error = nil, want error")
	}
	if got, want := err.Error(), "directory contains more than 2 markdown files; use a more specific path"; got != want {
		t.Fatalf("collectMarkdownFiles() error = %q, want %q", got, want)
	}
}

func TestRunPageSync_Directory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	root := filepath.Join(dir, "docs")
	mustMkdirAll(t, filepath.Join(root, "nested"))
	writeTestFile(t, filepath.Join(root, "guide-a.md"), []byte("---\nspace-key: TEST\npage-id: 400\nparent-id: 200\n---\n# A\n"))
	writeTestFile(t, filepath.Join(root, "nested", "guide-b.md"), []byte("---\nspace-key: TEST\npage-id: 401\nparent-id: 200\n---\n# B\n"))

	client := newFakeClient()
	client.pages["400"] = &page.Page{ID: "400", URL: "https://example.test/pages/400"}
	client.pages["401"] = &page.Page{ID: "401", URL: "https://example.test/pages/401"}

	var stdout bytes.Buffer
	app := New(client, &stdout)
	exit := app.Run(context.Background(), []string{"page", "sync", "docs"}, dir)
	if exit != 0 {
		t.Fatalf("Run() exit = %d, want 0", exit)
	}

	output := stdout.String()
	if !strings.Contains(output, "Synced: docs/guide-a.md -> https://example.test/pages/400") {
		t.Fatalf("output = %q, want synced guide-a", output)
	}
	if !strings.Contains(output, "Synced: docs/nested/guide-b.md -> https://example.test/pages/401") {
		t.Fatalf("output = %q, want synced guide-b", output)
	}
	if !strings.Contains(output, "Synced 2/2 files") {
		t.Fatalf("output = %q, want summary", output)
	}

	if gotPage := client.pageByID("400"); gotPage == nil || !strings.Contains(gotPage.Body, "<h1>A</h1>") {
		t.Fatalf("page 400 body = %q, want heading", pageBody(client.pageByID("400")))
	}
	if gotPage := client.pageByID("401"); gotPage == nil || !strings.Contains(gotPage.Body, "<h1>B</h1>") {
		t.Fatalf("page 401 body = %q, want heading", pageBody(client.pageByID("401")))
	}
}

func TestRunPageSync_DirectoryPartialFailure(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	root := filepath.Join(dir, "docs")
	mustMkdirAll(t, root)
	writeTestFile(t, filepath.Join(root, "guide-a.md"), []byte("---\nspace-key: TEST\npage-id: 400\nparent-id: 200\n---\n# A\n"))
	writeTestFile(t, filepath.Join(root, "guide-b.md"), []byte("---\nspace-key: TEST\npage-id: 401\nparent-id: 200\n---\n# B\n"))
	writeTestFile(t, filepath.Join(root, "draft.md"), []byte("# invalid\n"))

	client := newFakeClient()
	client.pages["400"] = &page.Page{ID: "400", URL: "https://example.test/pages/400"}

	var stdout bytes.Buffer
	app := New(client, &stdout)
	exit := app.Run(context.Background(), []string{"page", "sync", "docs"}, dir)
	if exit != 1 {
		t.Fatalf("Run() exit = %d, want 1", exit)
	}

	output := stdout.String()
	if !strings.Contains(output, "Synced: docs/guide-a.md -> https://example.test/pages/400") {
		t.Fatalf("output = %q, want synced guide-a", output)
	}
	if !strings.Contains(output, "Failed: docs/guide-b.md: not found") {
		t.Fatalf("output = %q, want failed guide-b", output)
	}
	if !strings.Contains(output, "Skipped (no valid frontmatter): docs/draft.md") {
		t.Fatalf("output = %q, want skipped draft", output)
	}
	if !strings.Contains(output, "Synced 1/2 files (1 skipped)") {
		t.Fatalf("output = %q, want summary", output)
	}

	if gotPage := client.pageByID("400"); gotPage == nil || !strings.Contains(gotPage.Body, "<h1>A</h1>") {
		t.Fatalf("page 400 body = %q, want heading", pageBody(client.pageByID("400")))
	}
}

func TestRunPageSync_DirectoryWatchRejected(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	root := filepath.Join(dir, "docs")
	mustMkdirAll(t, root)

	var stdout bytes.Buffer
	app := New(newFakeClient(), &stdout)
	exit := app.Run(context.Background(), []string{"page", "sync", "docs", "--watch"}, dir)
	if exit != 1 {
		t.Fatalf("Run() exit = %d, want 1", exit)
	}
	if !strings.Contains(stdout.String(), "--watch is only supported for a single markdown file") {
		t.Fatalf("output = %q, want watch rejection", stdout.String())
	}
}

func mustMkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
}

func writeTestFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}

func pageBody(p *page.Page) string {
	if p == nil {
		return ""
	}
	return p.Body
}
