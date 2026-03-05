package cli

import (
	"bytes"
	"context"
	"os"
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
		wantBody      string
		wantParentID  string
		wantTitle     string
		wantOutput    string
		wantPageCount int
	}{
		{
			name:          "explicit parent",
			args:          []string{"page", "new", "guide.md", "--space-id", "100", "--parent-id", "200"},
			wantExit:      0,
			wantBody:      "---\nspace-id: 100\npage-id: 401\nparent-id: 200\n---\n",
			wantParentID:  "200",
			wantTitle:     "guide",
			wantOutput:    "https://example.test/pages/401",
			wantPageCount: 1,
		},
		{
			name: "resolved root parent",
			args: []string{"page", "new", "guide.md", "--space-id", "100"},
			setup: func(t *testing.T, _ string, client *fakeClient) {
				t.Helper()
				client.spaceRoots["100"] = "300"
			},
			wantExit:      0,
			wantBody:      "---\nspace-id: 100\npage-id: 401\nparent-id: 300\n---\n",
			wantParentID:  "300",
			wantTitle:     "guide",
			wantOutput:    "https://example.test/pages/401",
			wantPageCount: 1,
		},
		{
			name: "existing local file",
			args: []string{"page", "new", "guide.md", "--space-id", "100", "--parent-id", "200"},
			setup: func(t *testing.T, dir string, _ *fakeClient) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(dir, "guide.md"), []byte("existing"), 0o644); err != nil {
					t.Fatalf("WriteFile() error = %v", err)
				}
			},
			wantExit:      1,
			wantOutput:    "already exists",
			wantPageCount: 0,
		},
		{
			name: "duplicate sibling title",
			args: []string{"page", "new", "guide.md", "--space-id", "100", "--parent-id", "200"},
			setup: func(t *testing.T, _ string, client *fakeClient) {
				t.Helper()
				client.children["200"] = map[string]string{"guide": "999"}
			},
			wantExit:      1,
			wantOutput:    "already exists under parent",
			wantPageCount: 0,
		},
		{
			name:          "basename derived title",
			args:          []string{"page", "new", "architecture-overview.md", "--space-id", "100", "--parent-id", "200"},
			wantExit:      0,
			wantBody:      "---\nspace-id: 100\npage-id: 401\nparent-id: 200\n---\n",
			wantParentID:  "200",
			wantTitle:     "architecture-overview",
			wantOutput:    "https://example.test/pages/401",
			wantPageCount: 1,
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

			filename := filepath.Join(dir, page.TitleFromPath(tt.args[2])+".md")
			got, err := os.ReadFile(filename)
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
			fileBody: "---\nspace-id: 100\npage-id: 400\nparent-id: 200\n---\n# Title\n\nParagraph.\n",
			setup: func(client *fakeClient) {
				client.pages["400"] = &page.Page{ID: "400", URL: "https://example.test/pages/400"}
			},
			wantExit:      0,
			wantOutput:    "https://example.test/pages/400",
			wantPageTitle: "guide",
			wantPageBody:  []string{"<h1>Title</h1>", "<p>Paragraph.</p>"},
		},
		{
			name:     "empty body",
			filename: "guide.md",
			fileBody: "---\nspace-id: 100\npage-id: 400\nparent-id: 200\n---\n",
			setup: func(client *fakeClient) {
				client.pages["400"] = &page.Page{ID: "400", URL: "https://example.test/pages/400"}
			},
			wantExit:      0,
			wantOutput:    "https://example.test/pages/400",
			wantPageTitle: "guide",
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
			fileBody:   "---\nspace-id: 100\npage-id: 400\n---\n",
			wantExit:   1,
			wantOutput: "required",
		},
		{
			name:       "malformed frontmatter",
			filename:   "guide.md",
			fileBody:   "---\nspace-id: [100\npage-id: 400\nparent-id: 200\n---\n",
			wantExit:   1,
			wantOutput: "malformed",
		},
		{
			name:     "title follows basename",
			filename: "renamed-guide.md",
			fileBody: "---\nspace-id: 100\npage-id: 400\nparent-id: 200\n---\nBody\n",
			setup: func(client *fakeClient) {
				client.pages["400"] = &page.Page{ID: "400", Title: "old-title", URL: "https://example.test/pages/400"}
			},
			wantExit:      0,
			wantOutput:    "https://example.test/pages/400",
			wantPageTitle: "renamed-guide",
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

	write("---\nspace-id: 100\npage-id: 400\nparent-id: 200\n---\n# Initial\n")

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

	write("---\nspace-id: 100\npage-id: 400\nparent-id: 200\n---\n# Burst 1\n")
	time.Sleep(10 * time.Millisecond)
	write("---\nspace-id: 100\npage-id: 400\nparent-id: 200\n---\n# Burst 2\n")

	waitFor(t, time.Second, func() bool {
		page := client.pageByID("400")
		return page != nil && strings.Contains(page.Body, "<h1>Burst 2</h1>")
	})

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

	write("---\nspace-id: 100\npage-id: 400\nparent-id: 200\n---\n# Recovered\n")
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

func stripANSI(s string) string {
	replacer := strings.NewReplacer("\x1b[31m", "", "\x1b[32m", "", "\x1b[0m", "")
	return replacer.Replace(s)
}

type fakeClient struct {
	mu          sync.Mutex
	nextID      int
	spaceRoots  map[string]string
	children    map[string]map[string]string
	pages       map[string]*page.Page
	updateCalls map[string]int
}

func newFakeClient() *fakeClient {
	return &fakeClient{
		nextID:      401,
		spaceRoots:  map[string]string{"100": "300"},
		children:    make(map[string]map[string]string),
		pages:       make(map[string]*page.Page),
		updateCalls: make(map[string]int),
	}
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

func (f *fakeClient) PageExists(ctx context.Context, spaceID string, parentID string, title string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	children := f.children[parentID]
	_, ok := children[title]
	return ok, nil
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
	if _, ok := f.children[parentID]; !ok {
		f.children[parentID] = make(map[string]string)
	}
	f.children[parentID][title] = p.ID
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
