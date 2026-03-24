package page

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestAttachmentFilenamesFromStorage(t *testing.T) {
	t.Parallel()

	storage := `<ac:image><ri:attachment ri:filename="mermaid-1.svg" /></ac:image>` +
		`<ac:image><ri:attachment ri:filename="diagram.svg" /></ac:image>` +
		`<ac:image><ri:attachment ri:filename="mermaid-1.svg" /></ac:image>`

	got := AttachmentFilenamesFromStorage(storage)
	if len(got) != 2 {
		t.Fatalf("AttachmentFilenamesFromStorage() len = %d, want 2", len(got))
	}
	if got[0] != "mermaid-1.svg" || got[1] != "diagram.svg" {
		t.Fatalf("AttachmentFilenamesFromStorage() = %#v, want [mermaid-1.svg diagram.svg]", got)
	}
}

func TestSyncAttachmentsFromStorage(t *testing.T) {
	t.Parallel()

	t.Run("uploads referenced files", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "mermaid-1.svg"), []byte("<svg></svg>"), 0o644); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		client := &fakeAttachmentClient{}
		storage := `<ac:image><ri:attachment ri:filename="mermaid-1.svg" /></ac:image>`
		if err := SyncAttachmentsFromStorage(context.Background(), client, "400", filepath.Join(dir, "guide.md"), storage, nil, nil); err != nil {
			t.Fatalf("SyncAttachmentsFromStorage() error = %v", err)
		}
		if len(client.putCalls) != 1 {
			t.Fatalf("putCalls = %d, want 1", len(client.putCalls))
		}
	})

	t.Run("skips unchanged files using cache", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "diagram.svg"), []byte("<svg></svg>"), 0o644); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		client := &fakeAttachmentClient{}
		storage := `<ac:image><ri:attachment ri:filename="diagram.svg" /></ac:image>`
		markdownPath := filepath.Join(dir, "guide.md")
		if err := SyncAttachmentsFromStorage(context.Background(), client, "400", markdownPath, storage, nil, nil); err != nil {
			t.Fatalf("first SyncAttachmentsFromStorage() error = %v", err)
		}
		if err := SyncAttachmentsFromStorage(context.Background(), client, "400", markdownPath, storage, nil, nil); err != nil {
			t.Fatalf("second SyncAttachmentsFromStorage() error = %v", err)
		}
		if len(client.putCalls) != 1 {
			t.Fatalf("putCalls = %d, want 1", len(client.putCalls))
		}
	})

	t.Run("different page id invalidates cache", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "diagram.svg"), []byte("<svg></svg>"), 0o644); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		client := &fakeAttachmentClient{}
		storage := `<ac:image><ri:attachment ri:filename="diagram.svg" /></ac:image>`
		markdownPath := filepath.Join(dir, "guide.md")
		if err := SyncAttachmentsFromStorage(context.Background(), client, "400", markdownPath, storage, nil, nil); err != nil {
			t.Fatalf("first SyncAttachmentsFromStorage() error = %v", err)
		}
		if err := SyncAttachmentsFromStorage(context.Background(), client, "401", markdownPath, storage, nil, nil); err != nil {
			t.Fatalf("second SyncAttachmentsFromStorage() error = %v", err)
		}
		if len(client.putCalls) != 2 {
			t.Fatalf("putCalls = %d, want 2", len(client.putCalls))
		}
	})

	t.Run("fails when file missing", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		client := &fakeAttachmentClient{}
		storage := `<ac:image><ri:attachment ri:filename="missing.svg" /></ac:image>`
		err := SyncAttachmentsFromStorage(context.Background(), client, "400", filepath.Join(dir, "guide.md"), storage, nil, nil)
		if err == nil {
			t.Fatal("SyncAttachmentsFromStorage() error = nil, want non-nil")
		}
	})

	t.Run("fails when upload fails", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "mermaid-1.svg"), []byte("<svg></svg>"), 0o644); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}
		client := &fakeAttachmentClient{failPut: true}
		storage := `<ac:image><ri:attachment ri:filename="mermaid-1.svg" /></ac:image>`
		err := SyncAttachmentsFromStorage(context.Background(), client, "400", filepath.Join(dir, "guide.md"), storage, nil, nil)
		if err == nil {
			t.Fatal("SyncAttachmentsFromStorage() error = nil, want non-nil")
		}
	})

	t.Run("uses cached mermaid asset path when local file is absent", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		markdownPath := filepath.Join(dir, "guide.md")
		generatedPath := filepath.Join(dir, "generated-mermaid.svg")
		if err := os.WriteFile(generatedPath, []byte("<svg></svg>"), 0o644); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		client := &fakeAttachmentClient{}
		storage := `<ac:image><ri:attachment ri:filename="mermaid-1.svg" /></ac:image>`
		generated := map[string]string{"mermaid-1.svg": generatedPath}
		if err := SyncAttachmentsFromStorage(context.Background(), client, "400", markdownPath, storage, nil, generated); err != nil {
			t.Fatalf("SyncAttachmentsFromStorage() error = %v", err)
		}
		if len(client.putCalls) != 1 {
			t.Fatalf("putCalls = %d, want 1", len(client.putCalls))
		}
	})
}

func TestSyncAttachments_MermaidReRenderAfterSourceChange(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	markdownPath := filepath.Join(dir, "guide.md")
	firstPath := filepath.Join(dir, "generated-a.svg")
	secondPath := filepath.Join(dir, "generated-b.svg")
	if err := os.WriteFile(firstPath, []byte("<svg>A</svg>"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.WriteFile(secondPath, []byte("<svg>B</svg>"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	client := &fakeAttachmentClient{}
	storage := `<ac:image><ri:attachment ri:filename="mermaid-1.svg" /></ac:image>`
	if err := SyncAttachmentsFromStorage(context.Background(), client, "400", markdownPath, storage, nil, map[string]string{"mermaid-1.svg": firstPath}); err != nil {
		t.Fatalf("first SyncAttachmentsFromStorage() error = %v", err)
	}
	if err := SyncAttachmentsFromStorage(context.Background(), client, "400", markdownPath, storage, nil, map[string]string{"mermaid-1.svg": secondPath}); err != nil {
		t.Fatalf("second SyncAttachmentsFromStorage() error = %v", err)
	}
	if len(client.putCalls) != 2 {
		t.Fatalf("putCalls = %d, want 2", len(client.putCalls))
	}
}

func TestSyncAttachments_MermaidUnchangedUsesFileCacheHash(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	markdownPath := filepath.Join(dir, "guide.md")
	generatedPath := filepath.Join(dir, "generated.svg")
	if err := os.WriteFile(generatedPath, []byte("<svg>stable</svg>"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	client := &fakeAttachmentClient{}
	storage := `<ac:image><ri:attachment ri:filename="mermaid-1.svg" /></ac:image>`
	if err := SyncAttachmentsFromStorage(context.Background(), client, "400", markdownPath, storage, nil, map[string]string{"mermaid-1.svg": generatedPath}); err != nil {
		t.Fatalf("first SyncAttachmentsFromStorage() error = %v", err)
	}

	fileHash, err := fileSHA256(generatedPath)
	if err != nil {
		t.Fatalf("fileSHA256() error = %v", err)
	}
	cachePath, err := mermaidCachePath(markdownPath)
	if err != nil {
		t.Fatalf("mermaidCachePath() error = %v", err)
	}
	cache := mermaidCache{
		Entries: map[string]mermaidCacheEntry{
			"mermaid-1.svg": {
				Source: "source-hash",
				File:   fileHash,
			},
		},
	}
	if err := saveMermaidCache(cachePath, cache); err != nil {
		t.Fatalf("saveMermaidCache() error = %v", err)
	}

	client.putCalls = nil
	if err := SyncAttachmentsFromStorage(context.Background(), client, "400", markdownPath, storage, nil, nil); err != nil {
		t.Fatalf("second SyncAttachmentsFromStorage() error = %v", err)
	}
	if len(client.putCalls) != 0 {
		t.Fatalf("putCalls = %d, want 0", len(client.putCalls))
	}
}

func TestSyncAttachmentsFromStorage_UsesAttachmentSourceMap(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	markdownPath := filepath.Join(dir, "docs", "guide.md")
	attachmentPath := filepath.Join(dir, "assets", "diagram.png")
	if err := os.MkdirAll(filepath.Dir(markdownPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(attachmentPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(attachmentPath, []byte("diagram"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	client := &fakeAttachmentClient{}
	storage := `<ac:image><ri:attachment ri:filename="diagram.png" /></ac:image>`
	attachmentSources := map[string]string{"diagram.png": attachmentPath}
	if err := SyncAttachmentsFromStorage(context.Background(), client, "400", markdownPath, storage, attachmentSources, nil); err != nil {
		t.Fatalf("SyncAttachmentsFromStorage() error = %v", err)
	}
	if len(client.putCalls) != 1 {
		t.Fatalf("putCalls = %d, want 1", len(client.putCalls))
	}
	if len(client.putPaths) != 1 {
		t.Fatalf("putPaths = %d, want 1", len(client.putPaths))
	}
	if client.putPaths[0] != attachmentPath {
		t.Fatalf("putPaths[0] = %q, want %q", client.putPaths[0], attachmentPath)
	}
}

type fakeAttachmentClient struct {
	putCalls []string
	putPaths []string
	failPut  bool
}

func (f *fakeAttachmentClient) SiteBaseURL() string {
	return "https://example.test"
}

func (f *fakeAttachmentClient) ResolveSpaceIDByKey(context.Context, string) (string, error) {
	return "", nil
}

func (f *fakeAttachmentClient) ResolveSpaceRootPage(context.Context, string) (string, error) {
	return "", nil
}

func (f *fakeAttachmentClient) CreatePage(context.Context, string, string, string, string, bool) (Page, error) {
	return Page{}, nil
}

func (f *fakeAttachmentClient) UpdatePage(context.Context, string, string, string, bool) (Page, error) {
	return Page{}, nil
}

func (f *fakeAttachmentClient) PutAttachment(_ context.Context, pageID string, filePath string) error {
	if f.failPut {
		return errors.New("put failed")
	}
	f.putCalls = append(f.putCalls, pageID+":"+filepath.Base(filePath))
	f.putPaths = append(f.putPaths, filePath)
	return nil
}

func (f *fakeAttachmentClient) DeleteAttachment(context.Context, string, string) error {
	return nil
}
