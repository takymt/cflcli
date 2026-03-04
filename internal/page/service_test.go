package page

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestServiceNewPageCreatesLocalAndRemoteArtifacts(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "guide.md")
	remote := NewFakeRemote()
	service := Service{Remote: remote}

	result, err := service.NewPage(context.Background(), path, "100", "200")
	if err != nil {
		t.Fatalf("NewPage returned error: %v", err)
	}

	if result.Action != "created" {
		t.Fatalf("unexpected action: %s", result.Action)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}

	content := string(raw)
	for _, want := range []string{
		"space-id: 100",
		"page-id: " + result.Page.ID,
		"parent-id: 200",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("generated file did not contain %q:\n%s", want, content)
		}
	}

	exists, err := remote.PageExists(context.Background(), "100", "200", "guide")
	if err != nil {
		t.Fatalf("PageExists returned error: %v", err)
	}
	if !exists {
		t.Fatal("expected created remote page to exist")
	}
}

func TestServiceNewPageResolvesRootParentWhenMissing(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "guide.md")
	remote := NewFakeRemote()
	remote.SetRootPage("100", "999")
	service := Service{Remote: remote}

	result, err := service.NewPage(context.Background(), path, "100", "")
	if err != nil {
		t.Fatalf("NewPage returned error: %v", err)
	}

	if result.Page.ParentID != "999" {
		t.Fatalf("unexpected parent id: %s", result.Page.ParentID)
	}
}

func TestServiceNewPageFailsWhenFileExists(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "guide.md")
	if err := os.WriteFile(path, []byte("existing"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	service := Service{Remote: NewFakeRemote()}
	_, err := service.NewPage(context.Background(), path, "100", "200")
	if err == nil {
		t.Fatal("expected error")
	}
	if err != ErrFileAlreadyExists {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServiceNewPageFailsOnDuplicateTitle(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "guide.md")
	remote := NewFakeRemote()
	remote.SeedPage(RemotePage{
		ID:       "123",
		SpaceID:  "100",
		ParentID: "200",
		Title:    "guide",
		URL:      "https://example.atlassian.net/wiki/pages/viewpage.action?pageId=123",
		Version:  1,
	})

	service := Service{Remote: remote}
	_, err := service.NewPage(context.Background(), path, "100", "200")
	if err == nil {
		t.Fatal("expected error")
	}
	if err != ErrDuplicatePage {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServiceSyncPageUpdatesTitleAndBody(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "renamed-guide.md")
	content := strings.Join([]string{
		"---",
		"space-id: 100",
		"page-id: 123",
		"parent-id: 200",
		"---",
		"# Heading",
		"",
		"- item",
	}, "\n")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	remote := NewFakeRemote()
	remote.SeedPage(RemotePage{
		ID:       "123",
		SpaceID:  "100",
		ParentID: "200",
		Title:    "old-title",
		URL:      "https://example.atlassian.net/wiki/pages/viewpage.action?pageId=123",
		Version:  1,
	})

	service := Service{Remote: remote}
	result, err := service.SyncPage(context.Background(), path)
	if err != nil {
		t.Fatalf("SyncPage returned error: %v", err)
	}

	if result.Page.Title != "renamed-guide" {
		t.Fatalf("unexpected title: %s", result.Page.Title)
	}
	if !strings.Contains(result.Page.Body, "<h1>Heading</h1>") {
		t.Fatalf("converted body did not contain heading: %s", result.Page.Body)
	}
	if !strings.Contains(result.Page.Body, "<ul><li>item</li></ul>") {
		t.Fatalf("converted body did not contain list: %s", result.Page.Body)
	}
}

func TestServiceSyncPageRejectsMissingFrontmatter(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "guide.md")
	if err := os.WriteFile(path, []byte("# no frontmatter"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	service := Service{Remote: NewFakeRemote()}
	_, err := service.SyncPage(context.Background(), path)
	if err == nil {
		t.Fatal("expected error")
	}
	if err != ErrMissingFrontmatter {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServiceSyncPageRejectsInvalidIdentifiers(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "guide.md")
	content := strings.Join([]string{
		"---",
		"space-id: abc",
		"page-id: 123",
		"parent-id: 200",
		"---",
	}, "\n")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	service := Service{Remote: NewFakeRemote()}
	_, err := service.SyncPage(context.Background(), path)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "invalid space-id") {
		t.Fatalf("unexpected error: %v", err)
	}
}
