package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/takymt/cflcli/internal/page"
)

func TestRunPageNew(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		args         []string
		wantPath     string
		wantBody     string
		wantParentID string
		wantDraft    bool
		wantTitle    string
		wantOutput   string
	}{
		{
			name:         "explicit parent defaults draft false",
			args:         []string{"page", "new", "--title", "Guide", "--space-key", "TEST", "--parent-id", "200"},
			wantPath:     "Guide.md",
			wantBody:     "---\ntitle: Guide\nspace-key: TEST\npage-id: 401\nparent-id: 200\ndraft: false\n---\n",
			wantParentID: "200",
			wantDraft:    false,
			wantTitle:    "Guide",
			wantOutput:   "https://example.test/pages/401",
		},
		{
			name:         "sanitizes title for default filename",
			args:         []string{"page", "new", "--title", "Architecture: Overview / Draft?", "--space-key", "TEST", "--parent-id", "200"},
			wantPath:     "Architecture Overview Draft.md",
			wantBody:     "---\ntitle: Architecture: Overview / Draft?\nspace-key: TEST\npage-id: 401\nparent-id: 200\ndraft: false\n---\n",
			wantParentID: "200",
			wantDraft:    false,
			wantTitle:    "Architecture: Overview / Draft?",
			wantOutput:   "https://example.test/pages/401",
		},
		{
			name:         "writes draft true when requested",
			args:         []string{"page", "new", "--title", "Draft Guide", "--space-key", "TEST", "--parent-id", "200", "--draft"},
			wantPath:     "Draft Guide.md",
			wantBody:     "---\ntitle: Draft Guide\nspace-key: TEST\npage-id: 401\nparent-id: 200\ndraft: true\n---\n",
			wantParentID: "200",
			wantDraft:    true,
			wantTitle:    "Draft Guide",
			wantOutput:   "https://example.test/pages/401",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			client := newFakeClient()
			assertRunPageNewSuccess(t, dir, client, tt.args, tt.wantPath, tt.wantBody, tt.wantParentID, tt.wantDraft, tt.wantTitle, tt.wantOutput)
		})
	}
}

func TestRunPageNewErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       []string
		wantOutput string
	}{
		{
			name:       "default filename reserved on windows",
			args:       []string{"page", "new", "--title", "CON", "--space-key", "TEST", "--parent-id", "200"},
			wantOutput: "pass --path",
		},
		{
			name:       "path must be markdown",
			args:       []string{"page", "new", "--title", "Guide", "--path", "guide.txt", "--space-key", "TEST", "--parent-id", "200"},
			wantOutput: "ending in .md",
		},
		{
			name:       "path parent must exist",
			args:       []string{"page", "new", "--title", "Guide", "--path", "missing/guide.md", "--space-key", "TEST", "--parent-id", "200"},
			wantOutput: "parent directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assertRunPageNewError(t, t.TempDir(), newFakeClient(), tt.args, tt.wantOutput)
		})
	}
}

func TestRunPageNew_ResolvedRootParent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	client := newFakeClient()
	client.spaceRoots["100"] = "300"
	client.spaceKeyToID["TEST"] = "100"

	assertRunPageNewSuccess(
		t,
		dir,
		client,
		[]string{"page", "new", "--title", "Guide", "--space-key", "TEST"},
		"Guide.md",
		"---\ntitle: Guide\nspace-key: TEST\npage-id: 401\nparent-id: 300\ndraft: false\n---\n",
		"300",
		false,
		"Guide",
		"https://example.test/pages/401",
	)
}

func TestRunPageNew_ExplicitPath(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "docs"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	client := newFakeClient()

	assertRunPageNewSuccess(
		t,
		dir,
		client,
		[]string{"page", "new", "--title", "Guide", "--path", "docs/guide.md", "--space-key", "TEST", "--parent-id", "200"},
		"docs/guide.md",
		"---\ntitle: Guide\nspace-key: TEST\npage-id: 401\nparent-id: 200\ndraft: false\n---\n",
		"200",
		false,
		"Guide",
		"https://example.test/pages/401",
	)
}

func TestRunPageNew_ExistingFileReturnsError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Guide.md"), []byte("existing"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	assertRunPageNewError(
		t,
		dir,
		newFakeClient(),
		[]string{"page", "new", "--title", "Guide", "--space-key", "TEST", "--parent-id", "200"},
		"already exists",
	)
}

func TestRunPageNew_PathMustNotBeDirectory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "guide.md"), 0o755); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}

	assertRunPageNewError(
		t,
		dir,
		newFakeClient(),
		[]string{"page", "new", "--title", "Guide", "--path", "guide.md", "--space-key", "TEST", "--parent-id", "200"},
		"is a directory",
	)
}

func assertRunPageNewSuccess(t *testing.T, dir string, client *fakeClient, args []string, wantPath string, wantBody string, wantParentID string, wantDraft bool, wantTitle string, wantOutput string) {
	t.Helper()

	var stdout bytes.Buffer
	app := New(client, &stdout)

	exit := app.Run(context.Background(), args, dir)
	if exit != 0 {
		t.Fatalf("Run() exit = %d, want 0", exit)
	}

	if !strings.Contains(stdout.String(), wantOutput) {
		t.Fatalf("Run() output = %q, want substring %q", stdout.String(), wantOutput)
	}

	if len(client.pages) != 1 {
		t.Fatalf("page count = %d, want 1", len(client.pages))
	}

	got, err := os.ReadFile(filepath.Join(dir, wantPath))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if string(got) != wantBody {
		t.Fatalf("created file = %q, want %q", string(got), wantBody)
	}

	if gotPage := client.pageByID("401"); gotPage == nil {
		t.Fatal("expected created page to exist")
	} else {
		if gotPage.ParentID != wantParentID {
			t.Fatalf("created page parent = %q, want %q", gotPage.ParentID, wantParentID)
		}
		if gotPage.Title != wantTitle {
			t.Fatalf("created page title = %q, want %q", gotPage.Title, wantTitle)
		}
	}
	if gotDraft := client.createDraftByID("401"); gotDraft != wantDraft {
		t.Fatalf("created page draft = %t, want %t", gotDraft, wantDraft)
	}
}

func assertRunPageNewError(t *testing.T, dir string, client *fakeClient, args []string, wantOutput string) {
	t.Helper()

	var stdout bytes.Buffer
	app := New(client, &stdout)

	exit := app.Run(context.Background(), args, dir)
	if exit != 1 {
		t.Fatalf("Run() exit = %d, want 1", exit)
	}

	if !strings.Contains(stdout.String(), wantOutput) {
		t.Fatalf("Run() output = %q, want substring %q", stdout.String(), wantOutput)
	}

	if len(client.pages) != 0 {
		t.Fatalf("page count = %d, want 0", len(client.pages))
	}
}

func TestRunPageSyncErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		fileBody   string
		wantOutput string
	}{
		{
			name:       "missing frontmatter",
			fileBody:   "# no frontmatter\n",
			wantOutput: "frontmatter",
		},
		{
			name:       "missing required key",
			fileBody:   "---\ntitle: guide\nspace-key: TEST\npage-id: 400\n---\n",
			wantOutput: "required",
		},
		{
			name:       "malformed frontmatter",
			fileBody:   "---\ntitle: guide\nspace-key: [TEST\npage-id: 400\nparent-id: 200\n---\n",
			wantOutput: "malformed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			path := filepath.Join(dir, "guide.md")
			if err := os.WriteFile(path, []byte(tt.fileBody), 0o644); err != nil {
				t.Fatalf("WriteFile() error = %v", err)
			}

			assertRunPageSyncError(t, dir, newFakeClient(), "guide.md", tt.wantOutput)
		})
	}
}

func TestRunPageSync_ValidFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "guide.md")
	if err := os.WriteFile(path, []byte("---\ntitle: Guide Title\nspace-key: TEST\npage-id: 400\nparent-id: 200\n---\n# Title\n\nParagraph.\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	client := newFakeClient()
	client.pages["400"] = &page.Page{ID: "400", URL: "https://example.test/pages/400"}

	assertRunPageSyncSuccess(t, dir, client, "guide.md", "Guide Title", "https://example.test/pages/400", "<h1>Title</h1>", "<p>Paragraph.</p>")
}

func TestRunPageSync_EmptyBody(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "guide.md")
	if err := os.WriteFile(path, []byte("---\ntitle: Empty Body\nspace-key: TEST\npage-id: 400\nparent-id: 200\n---\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	client := newFakeClient()
	client.pages["400"] = &page.Page{ID: "400", URL: "https://example.test/pages/400"}

	assertRunPageSyncSuccess(t, dir, client, "guide.md", "Empty Body", "https://example.test/pages/400", "")
}

func TestRunPageSync_TitleComesFromFrontmatter(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "renamed-guide.md")
	if err := os.WriteFile(path, []byte("---\ntitle: Guide From Frontmatter\nspace-key: TEST\npage-id: 400\nparent-id: 200\ndraft: false\n---\nBody\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	client := newFakeClient()
	client.pages["400"] = &page.Page{ID: "400", Title: "old-title", URL: "https://example.test/pages/400"}

	assertRunPageSyncSuccess(t, dir, client, "renamed-guide.md", "Guide From Frontmatter", "https://example.test/pages/400", "<p>Body</p>")
}

func TestRunPageSync_DraftComesFromFrontmatter(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "draft-guide.md")
	if err := os.WriteFile(path, []byte("---\ntitle: Draft Guide\nspace-key: TEST\npage-id: 400\nparent-id: 200\ndraft: true\n---\nBody\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	client := newFakeClient()
	client.pages["400"] = &page.Page{ID: "400", Title: "old-title", URL: "https://example.test/pages/400"}

	assertRunPageSyncSuccess(t, dir, client, "draft-guide.md", "Draft Guide", "https://example.test/pages/400", "<p>Body</p>")

	if gotDraft := client.updateDraftByID("400"); !gotDraft {
		t.Fatal("updated page draft = false, want true")
	}
}

func assertRunPageSyncSuccess(t *testing.T, dir string, client *fakeClient, filename string, wantPageTitle string, wantOutput string, wantPageBody ...string) {
	t.Helper()

	var stdout bytes.Buffer
	app := New(client, &stdout)
	exit := app.Run(context.Background(), []string{"page", "sync", filename}, dir)
	if exit != 0 {
		t.Fatalf("Run() exit = %d, want 0", exit)
	}

	if !strings.Contains(stdout.String(), wantOutput) {
		t.Fatalf("Run() output = %q, want substring %q", stdout.String(), wantOutput)
	}

	if gotPage := client.pageByID("400"); gotPage == nil {
		t.Fatal("expected page 400 to exist")
	} else {
		if gotPage.Title != wantPageTitle {
			t.Fatalf("updated page title = %q, want %q", gotPage.Title, wantPageTitle)
		}
		for _, want := range wantPageBody {
			if !strings.Contains(gotPage.Body, want) {
				t.Fatalf("updated page body = %q, want substring %q", gotPage.Body, want)
			}
		}
	}
}

func assertRunPageSyncError(t *testing.T, dir string, client *fakeClient, filename string, wantOutput string) {
	t.Helper()

	var stdout bytes.Buffer
	app := New(client, &stdout)
	exit := app.Run(context.Background(), []string{"page", "sync", filename}, dir)
	if exit != 1 {
		t.Fatalf("Run() exit = %d, want 1", exit)
	}

	if !strings.Contains(stdout.String(), wantOutput) {
		t.Fatalf("Run() output = %q, want substring %q", stdout.String(), wantOutput)
	}
}

func TestRunAttachmentPutDelete(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "diagram.svg")
	if err := os.WriteFile(filePath, []byte("<svg></svg>"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	client := newFakeClient()
	var stdout bytes.Buffer
	app := New(client, &stdout)

	exit := app.Run(context.Background(), []string{"attachment", "put", "--page-id", "400", filePath}, dir)
	if exit != 0 {
		t.Fatalf("Run(attachment put) exit = %d, want 0", exit)
	}
	if len(client.putAttachmentCalls) != 1 {
		t.Fatalf("putAttachmentCalls = %d, want 1", len(client.putAttachmentCalls))
	}

	exit = app.Run(context.Background(), []string{"attachment", "delete", "--page-id", "400", "diagram.svg"}, dir)
	if exit != 0 {
		t.Fatalf("Run(attachment delete) exit = %d, want 0", exit)
	}
	if len(client.deleteAttachmentCalls) != 1 {
		t.Fatalf("deleteAttachmentCalls = %d, want 1", len(client.deleteAttachmentCalls))
	}
}

func TestRunPageSync_AttachmentFailureDoesNotUpdateBody(t *testing.T) {
	t.Parallel()

	if _, err := exec.LookPath("mmdc"); err != nil {
		t.Skip("mmdc is required for mermaid attachment sync test")
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "guide.md")
	content := "---\ntitle: guide\nspace-key: TEST\npage-id: 400\nparent-id: 200\n---\n```mermaid\ngraph TD\nA-->B\n```\n"
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
	client.failPutAttachmentNames["mermaid-1.svg"] = true

	var stdout bytes.Buffer
	app := New(client, &stdout)
	exit := app.Run(context.Background(), []string{"page", "sync", "guide.md"}, dir)
	if exit != 1 {
		t.Fatalf("Run() exit = %d, want 1", exit)
	}

	p := client.pageByID("400")
	if p == nil {
		t.Fatal("expected page 400 to exist")
	}
	if p.Body != "<p>old body</p>" {
		t.Fatalf("page body updated unexpectedly: %q", p.Body)
	}
}

func TestRunPageSync_MermaidRetryAfterUploadFailure(t *testing.T) {
	t.Parallel()

	if _, err := exec.LookPath("mmdc"); err != nil {
		t.Skip("mmdc is required for mermaid attachment retry test")
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "guide.md")
	content := "---\ntitle: guide\nspace-key: TEST\npage-id: 400\nparent-id: 200\n---\n```mermaid\ngraph TD\nA-->B\n```\n"
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
	client.failPutAttachmentNames["mermaid-1.svg"] = true

	var stdout bytes.Buffer
	app := New(client, &stdout)

	firstExit := app.Run(context.Background(), []string{"page", "sync", "guide.md"}, dir)
	if firstExit != 1 {
		t.Fatalf("first Run() exit = %d, want 1", firstExit)
	}

	delete(client.failPutAttachmentNames, "mermaid-1.svg")
	secondExit := app.Run(context.Background(), []string{"page", "sync", "guide.md"}, dir)
	if secondExit != 0 {
		t.Fatalf("second Run() exit = %d, want 0 (output: %q)", secondExit, stdout.String())
	}

	if len(client.putAttachmentCalls) != 1 {
		t.Fatalf("putAttachmentCalls = %d, want 1", len(client.putAttachmentCalls))
	}

	p := client.pageByID("400")
	if p == nil {
		t.Fatal("expected page 400 to exist")
	}
	if !strings.Contains(p.Body, `ri:filename="mermaid-1.svg"`) {
		t.Fatalf("page body = %q, want mermaid attachment macro", p.Body)
	}
}

func TestRunPageSyncWatch(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "guide.md")
	write := func(content string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}
	}

	write("---\ntitle: guide\nspace-key: TEST\npage-id: 400\nparent-id: 200\n---\n# Initial\n")

	client := newFakeClient()
	client.pages["400"] = &page.Page{ID: "400", URL: "https://example.test/pages/400"}

	var stdout bytes.Buffer
	app := New(client, &stdout)
	app.watchPollInterval = 10 * time.Millisecond
	app.watchDebounce = 25 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan int, 1)
	go func() {
		done <- app.Run(ctx, []string{"page", "sync", "guide.md", "--watch"}, dir)
	}()

	waitFor(t, time.Second, func() bool {
		return strings.Contains(client.pageByID("400").Body, "<h1>Initial</h1>")
	})

	write("---\ntitle: guide\nspace-key: TEST\npage-id: 400\nparent-id: 200\n---\n# Burst 1\n")
	time.Sleep(10 * time.Millisecond)
	write("---\ntitle: guide\nspace-key: TEST\npage-id: 400\nparent-id: 200\n---\n# Burst 2\n")

	waitFor(t, time.Second, func() bool {
		page := client.pageByID("400")
		return page != nil && strings.Contains(page.Body, "<h1>Burst 2</h1>")
	})

	waitForStableUpdateCount(t, client, "400", 120*time.Millisecond)
	syncedAfterBurst := client.updateCount("400")

	otherPath := filepath.Join(dir, "other.md")
	if err := os.WriteFile(otherPath, []byte("ignored"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	time.Sleep(80 * time.Millisecond)
	if got := client.updateCount("400"); got != syncedAfterBurst {
		t.Fatalf("updateCount() after unrelated file change = %d, want %d", got, syncedAfterBurst)
	}

	write("invalid")
	waitFor(t, time.Second, func() bool {
		return strings.Contains(stripANSI(stdout.String()), "!")
	})

	write("---\ntitle: guide\nspace-key: TEST\npage-id: 400\nparent-id: 200\n---\n# Recovered\n")
	waitFor(t, time.Second, func() bool {
		page := client.pageByID("400")
		return page != nil && strings.Contains(page.Body, "<h1>Recovered</h1>")
	})

	cancel()

	select {
	case exit := <-done:
		if exit != 0 {
			t.Fatalf("Run() exit = %d, want 0", exit)
		}
	case <-time.After(time.Second):
		t.Fatal("watch command did not stop after cancel")
	}

	output := stripANSI(stdout.String())
	if !strings.Contains(output, "https://example.test/pages/400") {
		t.Fatalf("watch output = %q, want initial URL", output)
	}
	if strings.Count(output, ".") == 0 {
		t.Fatalf("watch output = %q, want success dot", output)
	}
	if !strings.Contains(output, "!") {
		t.Fatalf("watch output = %q, want failure marker", output)
	}
	if strings.Contains(output, "Rendering Mermaid...") ||
		strings.Contains(output, "Uploading attachments...") ||
		strings.Contains(output, "Updating page...") {
		t.Fatalf("watch output = %q, must not include progress lines on non-tty", output)
	}
}

func TestRunPageSyncWatch_MissingFileReturnsError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	client := newFakeClient()

	var stdout bytes.Buffer
	app := New(client, &stdout)
	exit := app.Run(context.Background(), []string{"page", "sync", "missing.md", "--watch"}, dir)
	if exit != 1 {
		t.Fatalf("Run() exit = %d, want 1", exit)
	}

	out := stripANSI(stdout.String())
	if strings.Contains(out, "Watching: ") {
		t.Fatalf("output = %q, must not start watch for missing file", out)
	}
}

func TestRunPageNewWatch(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "guide.md")

	client := newFakeClient()
	var stdout bytes.Buffer
	app := New(client, &stdout)
	app.watchPollInterval = 10 * time.Millisecond
	app.watchDebounce = 25 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan int, 1)
	go func() {
		done <- app.Run(ctx, []string{"page", "new", "--title", "Guide", "--path", "guide.md", "--space-key", "TEST", "--parent-id", "200", "--watch"}, dir)
	}()

	waitFor(t, time.Second, func() bool {
		_, err := os.Stat(path)
		return err == nil
	})
	if got := client.updateCount("401"); got != 0 {
		t.Fatalf("updateCount() before edits = %d, want 0", got)
	}

	if err := os.WriteFile(path, []byte("---\ntitle: guide\nspace-key: TEST\npage-id: 401\nparent-id: 200\n---\n# Updated\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	waitFor(t, time.Second, func() bool {
		page := client.pageByID("401")
		return page != nil && strings.Contains(page.Body, "<h1>Updated</h1>")
	})

	cancel()
	select {
	case exit := <-done:
		if exit != 0 {
			t.Fatalf("Run() exit = %d, want 0", exit)
		}
	case <-time.After(time.Second):
		t.Fatal("new --watch command did not stop after cancel")
	}

	output := stripANSI(stdout.String())
	if !strings.Contains(output, "https://example.test/pages/401") {
		t.Fatalf("output = %q, want page URL", output)
	}
	if !strings.Contains(output, "Watching: guide.md (press q to quit)") {
		t.Fatalf("output = %q, want watch start message", output)
	}
	if !strings.Contains(output, "Stopped watching: guide.md") {
		t.Fatalf("output = %q, want watch stop message", output)
	}
	if strings.Count(output, ".") == 0 {
		t.Fatalf("output = %q, want success dot from watch", output)
	}
	if strings.Contains(output, "Rendering Mermaid...") ||
		strings.Contains(output, "Uploading attachments...") ||
		strings.Contains(output, "Updating page...") {
		t.Fatalf("output = %q, must not include progress lines on non-tty", output)
	}
}

func waitFor(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatal("condition not met before timeout")
}

func waitForStableUpdateCount(t *testing.T, client *fakeClient, id string, stableFor time.Duration) {
	t.Helper()

	last := client.updateCount(id)
	stableSince := time.Now()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
		current := client.updateCount(id)
		if current != last {
			last = current
			stableSince = time.Now()
			continue
		}
		if time.Since(stableSince) >= stableFor {
			return
		}
	}

	t.Fatal("update count did not stabilize before timeout")
}

func stripANSI(s string) string {
	replacer := strings.NewReplacer("\x1b[31m", "", "\x1b[32m", "", "\x1b[0m", "")
	return replacer.Replace(s)
}

type fakeClient struct {
	mu                     sync.Mutex
	nextID                 int
	spaceRoots             map[string]string
	spaceKeyToID           map[string]string
	pages                  map[string]*page.Page
	createDraftCalls       map[string]bool
	updateCalls            map[string]int
	updateDraftCalls       map[string]bool
	putAttachmentCalls     []string
	deleteAttachmentCalls  []string
	failPutAttachmentNames map[string]bool
}

func newFakeClient() *fakeClient {
	return &fakeClient{
		nextID:                 401,
		spaceRoots:             make(map[string]string),
		spaceKeyToID:           map[string]string{"TEST": "100"},
		pages:                  make(map[string]*page.Page),
		createDraftCalls:       make(map[string]bool),
		updateCalls:            make(map[string]int),
		updateDraftCalls:       make(map[string]bool),
		failPutAttachmentNames: make(map[string]bool),
	}
}

func (f *fakeClient) SiteBaseURL() string {
	return "https://example.test"
}

func (f *fakeClient) ResolveSpaceIDByKey(ctx context.Context, spaceKey string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	id, ok := f.spaceKeyToID[spaceKey]
	if !ok {
		return "", page.ErrNotFound
	}
	return id, nil
}

func (f *fakeClient) ResolveSpaceRootPage(ctx context.Context, spaceID string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	root, ok := f.spaceRoots[spaceID]
	if !ok {
		return "", page.ErrNotFound
	}
	return root, nil
}

func (f *fakeClient) CreatePage(ctx context.Context, spaceID string, parentID string, title string, body string, draft bool) (page.Page, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	id := f.nextID
	f.nextID++
	p := &page.Page{
		ID:       strconv.Itoa(id),
		SpaceID:  spaceID,
		ParentID: parentID,
		Title:    title,
		Body:     body,
		URL:      "https://example.test/pages/" + strconv.Itoa(id),
	}
	f.pages[p.ID] = p
	f.createDraftCalls[p.ID] = draft
	return *p, nil
}

func (f *fakeClient) UpdatePage(ctx context.Context, pageID string, title string, body string, draft bool) (page.Page, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	p, ok := f.pages[pageID]
	if !ok {
		return page.Page{}, page.ErrNotFound
	}
	p.Title = title
	p.Body = body
	if p.URL == "" {
		p.URL = "https://example.test/pages/" + pageID
	}
	f.updateCalls[pageID]++
	f.updateDraftCalls[pageID] = draft
	return *p, nil
}

func (f *fakeClient) PutAttachment(ctx context.Context, pageID string, filePath string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	name := filepath.Base(filePath)
	if f.failPutAttachmentNames[name] {
		return errors.New("put attachment failed")
	}
	f.putAttachmentCalls = append(f.putAttachmentCalls, pageID+":"+name)
	return nil
}

func (f *fakeClient) DeleteAttachment(ctx context.Context, pageID string, filename string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.deleteAttachmentCalls = append(f.deleteAttachmentCalls, pageID+":"+filename)
	return nil
}

func (f *fakeClient) pageByID(id string) *page.Page {
	f.mu.Lock()
	defer f.mu.Unlock()

	p, ok := f.pages[id]
	if !ok {
		return nil
	}
	cp := *p
	return &cp
}

func (f *fakeClient) updateCount(id string) int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.updateCalls[id]
}

func (f *fakeClient) createDraftByID(id string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.createDraftCalls[id]
}

func (f *fakeClient) updateDraftByID(id string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.updateDraftCalls[id]
}
