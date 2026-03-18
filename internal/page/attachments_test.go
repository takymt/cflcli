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
		if err := SyncAttachmentsFromStorage(context.Background(), client, "400", filepath.Join(dir, "guide.md"), storage, nil); err != nil {
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
		if err := SyncAttachmentsFromStorage(context.Background(), client, "400", markdownPath, storage, nil); err != nil {
			t.Fatalf("first SyncAttachmentsFromStorage() error = %v", err)
		}
		if err := SyncAttachmentsFromStorage(context.Background(), client, "400", markdownPath, storage, nil); err != nil {
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
		if err := SyncAttachmentsFromStorage(context.Background(), client, "400", markdownPath, storage, nil); err != nil {
			t.Fatalf("first SyncAttachmentsFromStorage() error = %v", err)
		}
		if err := SyncAttachmentsFromStorage(context.Background(), client, "401", markdownPath, storage, nil); err != nil {
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
		err := SyncAttachmentsFromStorage(context.Background(), client, "400", filepath.Join(dir, "guide.md"), storage, nil)
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
		err := SyncAttachmentsFromStorage(context.Background(), client, "400", filepath.Join(dir, "guide.md"), storage, nil)
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
		if err := SyncAttachmentsFromStorage(context.Background(), client, "400", markdownPath, storage, generated); err != nil {
			t.Fatalf("SyncAttachmentsFromStorage() error = %v", err)
		}
		if len(client.putCalls) != 1 {
			t.Fatalf("putCalls = %d, want 1", len(client.putCalls))
		}
	})
}

type fakeAttachmentClient struct {
	putCalls []string
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

func (f *fakeAttachmentClient) CreatePage(context.Context, string, string, string, string) (Page, error) {
	return Page{}, nil
}

func (f *fakeAttachmentClient) UpdatePage(context.Context, string, string, string) (Page, error) {
	return Page{}, nil
}

func (f *fakeAttachmentClient) PutAttachment(_ context.Context, pageID string, filePath string) error {
	if f.failPut {
		return errors.New("put failed")
	}
	f.putCalls = append(f.putCalls, pageID+":"+filepath.Base(filePath))
	return nil
}

func (f *fakeAttachmentClient) DeleteAttachment(context.Context, string, string) error {
	return nil
}
