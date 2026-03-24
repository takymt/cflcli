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

func TestRunPageSync_MissingRelativeAttachmentDoesNotUpdateBody(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "guide.md")
	content := "---\ntitle: guide\nspace-key: TEST\npage-id: 400\nparent-id: 200\n---\n![Missing](./missing.png)\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	client := newFakeClient()
	client.pages["400"] = &page.Page{
		ID:    "400",
		Title: "guide",
		Body:  "<p>old body</p>",
		URL:   "https://example.test/pages/400",
	}

	var stdout bytes.Buffer
	app := New(client, &stdout)
	exit := app.Run(context.Background(), []string{"page", "sync", "guide.md"}, dir)
	if exit != 1 {
		t.Fatalf("Run() exit = %d, want 1", exit)
	}

	if !strings.Contains(stdout.String(), `missing.png`) {
		t.Fatalf("Run() output = %q, want missing attachment message", stdout.String())
	}
	if len(client.putAttachmentCalls) != 0 {
		t.Fatalf("putAttachmentCalls = %d, want 0", len(client.putAttachmentCalls))
	}

	p := client.pageByID("400")
	if p == nil {
		t.Fatal("expected page 400 to exist")
	}
	if p.Body != "<p>old body</p>" {
		t.Fatalf("page body updated unexpectedly: %q", p.Body)
	}
}

func TestRunPageSync_AbsolutePathWarningIsPrinted(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "guide.md")
	content := "---\ntitle: guide\nspace-key: TEST\npage-id: 400\nparent-id: 200\n---\n![Diagram](/tmp/diagram.png)\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	client := newFakeClient()
	client.pages["400"] = &page.Page{ID: "400", Title: "guide", URL: "https://example.test/pages/400"}

	var stdout bytes.Buffer
	app := New(client, &stdout)
	exit := app.Run(context.Background(), []string{"page", "sync", "guide.md"}, dir)
	if exit != 0 {
		t.Fatalf("Run() exit = %d, want 0 (output: %q)", exit, stdout.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "absolute local path") || !strings.Contains(output, "/tmp/diagram.png") {
		t.Fatalf("Run() output = %q, want absolute path warning", output)
	}
	if len(client.putAttachmentCalls) != 0 {
		t.Fatalf("putAttachmentCalls = %d, want 0", len(client.putAttachmentCalls))
	}

	p := client.pageByID("400")
	if p == nil {
		t.Fatal("expected page 400 to exist")
	}
	if strings.Contains(p.Body, `ri:filename="diagram.png"`) {
		t.Fatalf("page body = %q, must not include attachment macro for absolute path", p.Body)
	}
}
