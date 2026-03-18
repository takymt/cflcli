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
		name          string
		args          []string
		setup         func(t *testing.T, dir string, client *fakeClient)
		wantExit      int
		wantPath      string
		wantBody      string
		wantParentID  string
		wantTitle     string
		wantOutput    string
		wantPageCount int
	}{
		{
			name:          "explicit parent",
			args:          []string{"page", "new", "--title", "Guide", "--space-key", "TEST", "--parent-id", "200"},
			wantExit:      0,
			wantPath:      "Guide.md",
			wantBody:      "---\ntitle: Guide\nspace-key: TEST\npage-id: 401\nparent-id: 200\n---\n",
			wantParentID:  "200",
			wantTitle:     "Guide",
			wantOutput:    "https://example.test/pages/401",
			wantPageCount: 1,
		},
		{
			name: "resolved root parent",
			args: []string{"page", "new", "--title", "Guide", "--space-key", "TEST"},
			setup: func(t *testing.T, _ string, client *fakeClient) {
				t.Helper()
				client.spaceRoots["100"] = "300"
				client.spaceKeyToID["TEST"] = "100"
			},
			wantExit:      0,
			wantPath:      "Guide.md",
			wantBody:      "---\ntitle: Guide\nspace-key: TEST\npage-id: 401\nparent-id: 300\n---\n",
			wantParentID:  "300",
			wantTitle:     "Guide",
			wantOutput:    "https://example.test/pages/401",
			wantPageCount: 1,
		},
		{
			name: "explicit path",
			args: []string{"page", "new", "--title", "Guide", "--path", "docs/guide.md", "--space-key", "TEST", "--parent-id", "200"},
			setup: func(t *testing.T, dir string, _ *fakeClient) {
				t.Helper()
				if err := os.MkdirAll(filepath.Join(dir, "docs"), 0o755); err != nil {
					t.Fatalf("MkdirAll() error = %v", err)
				}
			},
			wantExit:      0,
			wantPath:      "docs/guide.md",
			wantBody:      "---\ntitle: Guide\nspace-key: TEST\npage-id: 401\nparent-id: 200\n---\n",
			wantParentID:  "200",
			wantTitle:     "Guide",
			wantOutput:    "https://example.test/pages/401",
			wantPageCount: 1,
		},
		{
			name: "existing local file",
			args: []string{"page", "new", "--title", "Guide", "--space-key", "TEST", "--parent-id", "200"},
			setup: func(t *testing.T, dir string, _ *fakeClient) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(dir, "Guide.md"), []byte("existing"), 0o644); err != nil {
					t.Fatalf("WriteFile() error = %v", err)
				}
			},
			wantExit:      1,
			wantOutput:    "already exists",
			wantPageCount: 0,
		},
		{
			name:          "sanitizes title for default filename",
			args:          []string{"page", "new", "--title", "Architecture: Overview / Draft?", "--space-key", "TEST", "--parent-id", "200"},
			wantExit:      0,
			wantPath:      "Architecture Overview Draft.md",
			wantBody:      "---\ntitle: Architecture: Overview / Draft?\nspace-key: TEST\npage-id: 401\nparent-id: 200\n---\n",
			wantParentID:  "200",
			wantTitle:     "Architecture: Overview / Draft?",
			wantOutput:    "https://example.test/pages/401",
			wantPageCount: 1,
		},
		{
			name:          "windows reserved default filename",
			args:          []string{"page", "new", "--title", "CON", "--space-key", "TEST", "--parent-id", "200"},
			wantExit:      1,
			wantOutput:    "pass --path",
			wantPageCount: 0,
		},
		{
			name:          "path must be markdown",
			args:          []string{"page", "new", "--title", "Guide", "--path", "guide.txt", "--space-key", "TEST", "--parent-id", "200"},
			wantExit:      1,
			wantOutput:    "ending in .md",
			wantPageCount: 0,
		},
		{
			name:          "path parent must exist",
			args:          []string{"page", "new", "--title", "Guide", "--path", "missing/guide.md", "--space-key", "TEST", "--parent-id", "200"},
			wantExit:      1,
			wantOutput:    "parent directory",
			wantPageCount: 0,
		},
		{
			name: "path must not be directory",
			args: []string{"page", "new", "--title", "Guide", "--path", "guide.md", "--space-key", "TEST", "--parent-id", "200"},
			setup: func(t *testing.T, dir string, _ *fakeClient) {
				t.Helper()
				if err := os.Mkdir(filepath.Join(dir, "guide.md"), 0o755); err != nil {
					t.Fatalf("Mkdir() error = %v", err)
				}
			},
			wantExit:      1,
			wantOutput:    "is a directory",
			wantPageCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			client := newFakeClient()
			if tt.setup != nil {
				tt.setup(t, dir, client)
			}

			var stdout bytes.Buffer
			app := New(client, &stdout)

			exit := app.Run(context.Background(), tt.args, dir)
			if exit != tt.wantExit {
				t.Fatalf("Run() exit = %d, want %d", exit, tt.wantExit)
			}

			if !strings.Contains(stdout.String(), tt.wantOutput) {
				t.Fatalf("Run() output = %q, want substring %q", stdout.String(), tt.wantOutput)
			}

			if len(client.pages) != tt.wantPageCount {
				t.Fatalf("page count = %d, want %d", len(client.pages), tt.wantPageCount)
			}

			if tt.wantExit != 0 {
				return
			}

			got, err := os.ReadFile(filepath.Join(dir, tt.wantPath))
			if err != nil {
				t.Fatalf("ReadFile() error = %v", err)
			}

			if string(got) != tt.wantBody {
				t.Fatalf("created file = %q, want %q", string(got), tt.wantBody)
			}

			if gotPage := client.pageByID("401"); gotPage == nil {
				t.Fatal("expected created page to exist")
			} else {
				if gotPage.ParentID != tt.wantParentID {
					t.Fatalf("created page parent = %q, want %q", gotPage.ParentID, tt.wantParentID)
				}
				if gotPage.Title != tt.wantTitle {
					t.Fatalf("created page title = %q, want %q", gotPage.Title, tt.wantTitle)
				}
			}
		})
	}
}

func TestRunPageSync(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		filename      string
		fileBody      string
		setup         func(*fakeClient)
		wantExit      int
		wantOutput    string
		wantPageTitle string
		wantPageBody  []string
	}{
		{
			name:     "valid file",
			filename: "guide.md",
			fileBody: "---\ntitle: Guide Title\nspace-key: TEST\npage-id: 400\nparent-id: 200\n---\n# Title\n\nParagraph.\n",
			setup: func(client *fakeClient) {
				client.pages["400"] = &page.Page{ID: "400", URL: "https://example.test/pages/400"}
			},
			wantExit:      0,
			wantOutput:    "https://example.test/pages/400",
			wantPageTitle: "Guide Title",
			wantPageBody:  []string{"<h1>Title</h1>", "<p>Paragraph.</p>"},
		},
		{
			name:     "empty body",
			filename: "guide.md",
			fileBody: "---\ntitle: Empty Body\nspace-key: TEST\npage-id: 400\nparent-id: 200\n---\n",
			setup: func(client *fakeClient) {
				client.pages["400"] = &page.Page{ID: "400", URL: "https://example.test/pages/400"}
			},
			wantExit:      0,
			wantOutput:    "https://example.test/pages/400",
			wantPageTitle: "Empty Body",
			wantPageBody:  []string{""},
		},
		{
			name:       "missing frontmatter",
			filename:   "guide.md",
			fileBody:   "# no frontmatter\n",
			wantExit:   1,
			wantOutput: "frontmatter",
		},
		{
			name:       "missing required key",
			filename:   "guide.md",
			fileBody:   "---\ntitle: guide\nspace-key: TEST\npage-id: 400\n---\n",
			wantExit:   1,
			wantOutput: "required",
		},
		{
			name:       "malformed frontmatter",
			filename:   "guide.md",
			fileBody:   "---\ntitle: guide\nspace-key: [TEST\npage-id: 400\nparent-id: 200\n---\n",
			wantExit:   1,
			wantOutput: "malformed",
		},
		{
			name:     "title comes from frontmatter",
			filename: "renamed-guide.md",
			fileBody: "---\ntitle: Guide From Frontmatter\nspace-key: TEST\npage-id: 400\nparent-id: 200\n---\nBody\n",
			setup: func(client *fakeClient) {
				client.pages["400"] = &page.Page{ID: "400", Title: "old-title", URL: "https://example.test/pages/400"}
			},
			wantExit:      0,
			wantOutput:    "https://example.test/pages/400",
			wantPageTitle: "Guide From Frontmatter",
			wantPageBody:  []string{"<p>Body</p>"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			path := filepath.Join(dir, tt.filename)
			if err := os.WriteFile(path, []byte(tt.fileBody), 0o644); err != nil {
				t.Fatalf("WriteFile() error = %v", err)
			}

			client := newFakeClient()
			if tt.setup != nil {
				tt.setup(client)
			}

			var stdout bytes.Buffer
			app := New(client, &stdout)
			exit := app.Run(context.Background(), []string{"page", "sync", tt.filename}, dir)
			if exit != tt.wantExit {
				t.Fatalf("Run() exit = %d, want %d", exit, tt.wantExit)
			}

			if !strings.Contains(stdout.String(), tt.wantOutput) {
				t.Fatalf("Run() output = %q, want substring %q", stdout.String(), tt.wantOutput)
			}

			if tt.wantExit != 0 {
				return
			}

			if gotPage := client.pageByID("400"); gotPage == nil {
				t.Fatal("expected page 400 to exist")
			} else {
				if gotPage.Title != tt.wantPageTitle {
					t.Fatalf("updated page title = %q, want %q", gotPage.Title, tt.wantPageTitle)
				}
				for _, want := range tt.wantPageBody {
					if !strings.Contains(gotPage.Body, want) {
						t.Fatalf("updated page body = %q, want substring %q", gotPage.Body, want)
					}
				}
			}
		})
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
	updateCalls            map[string]int
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
		updateCalls:            make(map[string]int),
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

func (f *fakeClient) CreatePage(ctx context.Context, spaceID string, parentID string, title string, body string) (page.Page, error) {
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
	return *p, nil
}

func (f *fakeClient) UpdatePage(ctx context.Context, pageID string, title string, body string) (page.Page, error) {
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
