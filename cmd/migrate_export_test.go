package cmd

import (
	"bytes"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunMigrateExportWithConfig_WritesMarkdownAndAttachments(t *testing.T) {
	var gotPageCursors []string
	var gotAttachmentFilenameQuery string
	attachmentPath := "/wiki/" + "rest/api/content/1/child/attachment"

	srv := setupPageListServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/wiki/api/v2/spaces":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"results":[{"id":"SPACE-1","key":"WORK"}]}`))
		case "/wiki/api/v2/pages":
			cursor := r.URL.Query().Get("cursor")
			gotPageCursors = append(gotPageCursors, cursor)
			w.Header().Set("Content-Type", "application/json")
			if cursor == "" {
				_, _ = w.Write([]byte(`{"results":[{"id":"1","title":"Root Page","status":"current","spaceId":"SPACE-1"}],"_links":{"next":"/wiki/api/v2/pages?limit=250&cursor=cursor-2"}}`))
				return
			}
			if cursor == "cursor-2" {
				_, _ = w.Write([]byte(`{"results":[{"id":"2","title":"Child Page","status":"current","spaceId":"SPACE-1","parentId":"1"}],"_links":{}}`))
				return
			}
			t.Fatalf("unexpected cursor value: %q", cursor)
		case "/wiki/api/v2/pages/1":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"1","title":"Root Page","status":"current","spaceId":"SPACE-1","parentId":"","body":{"storage":{"representation":"storage","value":"<p>Intro</p><ac:image><ri:attachment ri:filename=\"logo.png\" /></ac:image>"}}}`))
		case "/wiki/api/v2/pages/2":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"2","title":"Child Page","status":"current","spaceId":"SPACE-1","parentId":"1","body":{"storage":{"representation":"storage","value":"<p>Child body</p>"}}}`))
		case attachmentPath:
			gotAttachmentFilenameQuery = r.URL.Query().Get("filename")
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"results":[{"id":"att-1","title":"logo.png","_links":{"download":"/download/attachments/1/logo.png"}}]}`))
		case "/download/attachments/1/logo.png":
			_, _ = w.Write([]byte("PNGDATA"))
		default:
			http.NotFound(w, r)
		}
	}))

	t.Setenv("CFL_API_TOKEN", "token")
	setOutputMode(t, "table")

	outDir := t.TempDir()
	opts := &migrateExportOptions{
		SpaceKey: "WORK",
		Out:      outDir,
	}

	var out bytes.Buffer
	if err := RunMigrateExportWithConfig(&out, opts, newPageListConfig(srv.URL, "WORK")); err != nil {
		t.Fatalf("RunMigrateExportWithConfig: %v", err)
	}

	if len(gotPageCursors) != 2 || gotPageCursors[0] != "" || gotPageCursors[1] != "cursor-2" {
		t.Fatalf("page list cursor sequence=%v want [\"\", \"cursor-2\"]", gotPageCursors)
	}
	if gotAttachmentFilenameQuery != "logo.png" {
		t.Fatalf("attachment filename query=%q want %q", gotAttachmentFilenameQuery, "logo.png")
	}

	rootMarkdownPath := filepath.Join(outDir, "root-page-1", "index.md")
	rootMarkdown, err := os.ReadFile(rootMarkdownPath)
	if err != nil {
		t.Fatalf("ReadFile(root markdown): %v", err)
	}
	rootContent := string(rootMarkdown)
	if !strings.Contains(rootContent, `page-id: "1"`) ||
		!strings.Contains(rootContent, `title: "Root Page"`) ||
		!strings.Contains(rootContent, `parent-id: ""`) ||
		!strings.Contains(rootContent, `space-key: "WORK"`) {
		t.Fatalf("frontmatter mismatch: %q", rootContent)
	}
	if !strings.Contains(rootContent, `attachments/_migrate/1/logo.png`) {
		t.Fatalf("attachment markdown missing: %q", rootContent)
	}

	childMarkdownPath := filepath.Join(outDir, "root-page-1", "child-page-2", "index.md")
	childMarkdown, err := os.ReadFile(childMarkdownPath)
	if err != nil {
		t.Fatalf("ReadFile(child markdown): %v", err)
	}
	childContent := string(childMarkdown)
	if !strings.Contains(childContent, `parent-id: "1"`) {
		t.Fatalf("child frontmatter parent-id mismatch: %q", childContent)
	}

	attachmentFilePath := filepath.Join(outDir, "attachments", "_migrate", "1", "logo.png")
	attachmentContent, err := os.ReadFile(attachmentFilePath)
	if err != nil {
		t.Fatalf("ReadFile(attachment): %v", err)
	}
	if string(attachmentContent) != "PNGDATA" {
		t.Fatalf("attachment content=%q want %q", string(attachmentContent), "PNGDATA")
	}

	if !strings.Contains(out.String(), `Exported 2 pages`) {
		t.Fatalf("unexpected output: %q", out.String())
	}
}

func TestRunMigrateExportWithConfig_RootPageAndCustomAttachmentsDir(t *testing.T) {
	attachmentPath := "/wiki/" + "rest/api/content/1/child/attachment"
	srv := setupPageListServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/wiki/api/v2/spaces":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"results":[{"id":"SPACE-1","key":"WORK"}]}`))
		case "/wiki/api/v2/pages":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"results":[{"id":"1","title":"Root","status":"current","spaceId":"SPACE-1"},{"id":"2","title":"Child","status":"current","spaceId":"SPACE-1","parentId":"1"},{"id":"3","title":"Outside","status":"current","spaceId":"SPACE-1"}],"_links":{}}`))
		case "/wiki/api/v2/pages/1":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"1","title":"Root","status":"current","spaceId":"SPACE-1","body":{"storage":{"representation":"storage","value":"<ac:image><ri:attachment ri:filename=\"logo.png\" /></ac:image>"}}}`))
		case "/wiki/api/v2/pages/2":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"2","title":"Child","status":"current","spaceId":"SPACE-1","parentId":"1","body":{"storage":{"representation":"storage","value":"<p>child</p>"}}}`))
		case "/wiki/api/v2/pages/3":
			t.Fatalf("page 3 must not be exported when root-page-id=1")
		case attachmentPath:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"results":[{"id":"att-1","title":"logo.png","_links":{"download":"/download/attachments/1/logo.png"}}]}`))
		case "/download/attachments/1/logo.png":
			_, _ = w.Write([]byte("PNGDATA"))
		default:
			http.NotFound(w, r)
		}
	}))

	t.Setenv("CFL_API_TOKEN", "token")
	setOutputMode(t, "table")

	outDir := t.TempDir()
	opts := &migrateExportOptions{
		SpaceKey:       "WORK",
		RootPageID:     "1",
		Out:            outDir,
		AttachmentsDir: "attachments/custom",
	}

	if err := RunMigrateExportWithConfig(&bytes.Buffer{}, opts, newPageListConfig(srv.URL, "WORK")); err != nil {
		t.Fatalf("RunMigrateExportWithConfig: %v", err)
	}

	if _, err := os.Stat(filepath.Join(outDir, "outside-3", "index.md")); !os.IsNotExist(err) {
		t.Fatalf("outside page must not be exported: err=%v", err)
	}

	rootMarkdown, err := os.ReadFile(filepath.Join(outDir, "root-1", "index.md"))
	if err != nil {
		t.Fatalf("ReadFile(root markdown): %v", err)
	}
	if !strings.Contains(string(rootMarkdown), `attachments/custom/1/logo.png`) {
		t.Fatalf("custom attachments dir was not reflected in markdown: %q", string(rootMarkdown))
	}
}

func TestRunMigrateExportWithConfig_RequiresOut(t *testing.T) {
	err := RunMigrateExportWithConfig(&bytes.Buffer{}, &migrateExportOptions{
		SpaceKey: "WORK",
	}, newPageListConfig("example.atlassian.net", "WORK"))
	if err == nil || !strings.Contains(err.Error(), "--out is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}
