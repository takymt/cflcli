package cmd

import (
	"bytes"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunMigrateExportWithConfig(t *testing.T) {
	t.Run("writes markdown and attachments", func(t *testing.T) {
		attachmentPath := "/wiki/api/v2/pages/1/attachments"
		srv := setupPageListServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/wiki/api/v2/spaces":
				writeJSON(t, w, `{"results":[{"id":"SPACE-1","key":"WORK"}]}`)
			case "/wiki/api/v2/spaces/SPACE-1/pages":
				switch r.URL.Query().Get("cursor") {
				case "":
					writeJSON(t, w, `{"results":[{"id":"1","title":"Root Page","status":"current","spaceId":"SPACE-1"}],"_links":{"next":"/wiki/api/v2/pages?limit=250&cursor=cursor-2"}}`)
				case "cursor-2":
					writeJSON(t, w, `{"results":[{"id":"2","title":"Child Page","status":"current","spaceId":"SPACE-1","parentId":"1"}],"_links":{}}`)
				default:
					t.Fatalf("unexpected cursor value: %q", r.URL.Query().Get("cursor"))
				}
			case "/wiki/api/v2/pages/1":
				writeJSON(t, w, `{"id":"1","title":"Root Page","status":"current","spaceId":"SPACE-1","parentId":"","body":{"storage":{"representation":"storage","value":"<p>Intro</p><ac:image><ri:attachment ri:filename=\"logo.png\" /></ac:image>"}}}`)
			case "/wiki/api/v2/pages/2":
				writeJSON(t, w, `{"id":"2","title":"Child Page","status":"current","spaceId":"SPACE-1","parentId":"1","body":{"storage":{"representation":"storage","value":"<p>Child body</p>"}}}`)
			case attachmentPath:
				writeJSON(t, w, `{"results":[{"id":"att-1","title":"logo.png","downloadLink":"/download/attachments/1/logo.png"}]}`)
			case "/download/attachments/1/logo.png", "/wiki/download/attachments/1/logo.png":
				_, _ = w.Write([]byte("PNGDATA"))
			default:
				http.NotFound(w, r)
			}
		}))

		outDir := t.TempDir()
		var out bytes.Buffer
		err := runMigrateExportForTest(t, &out, &migrateExportOptions{
			SpaceKey: "WORK",
			Out:      outDir,
		}, srv.URL)
		if err != nil {
			t.Fatalf("RunMigrateExportWithConfig: %v", err)
		}

		rootMarkdown := mustReadFileString(t, filepath.Join(outDir, "root-page", "_index.md"))
		assertContainsAll(t, rootMarkdown,
			`page-id: "1"`,
			`title: "Root Page"`,
			`parent-id: ""`,
			`space-key: "WORK"`,
			`attachments/_migrate/1/logo.png`,
		)

		childMarkdown := mustReadFileString(t, filepath.Join(outDir, "root-page", "child-page.md"))
		assertContainsAll(t, childMarkdown, `parent-id: "1"`)

		if got := mustReadFileString(t, filepath.Join(outDir, "attachments", "_migrate", "1", "logo.png")); got != "PNGDATA" {
			t.Fatalf("attachment content=%q want %q", got, "PNGDATA")
		}

		assertContainsAll(t, out.String(), `Exported 2 pages`)
	})

	t.Run("root page and custom attachments dir", func(t *testing.T) {
		attachmentPath := "/wiki/api/v2/pages/1/attachments"
		srv := setupPageListServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/wiki/api/v2/spaces":
				writeJSON(t, w, `{"results":[{"id":"SPACE-1","key":"WORK"}]}`)
			case "/wiki/api/v2/spaces/SPACE-1/pages":
				writeJSON(t, w, `{"results":[{"id":"1","title":"Root","status":"current","spaceId":"SPACE-1"},{"id":"2","title":"Child","status":"current","spaceId":"SPACE-1","parentId":"f1","parentType":"folder"},{"id":"3","title":"Outside","status":"current","spaceId":"SPACE-1"}],"_links":{}}`)
			case "/wiki/api/v2/pages/1":
				writeJSON(t, w, `{"id":"1","title":"Root","status":"current","spaceId":"SPACE-1","body":{"storage":{"representation":"storage","value":"<ac:image><ri:attachment ri:filename=\"logo.png\" /></ac:image>"}}}`)
			case "/wiki/api/v2/pages/2":
				writeJSON(t, w, `{"id":"2","title":"Child","status":"current","spaceId":"SPACE-1","parentId":"f1","parentType":"folder","body":{"storage":{"representation":"storage","value":"<p>child</p>"}}}`)
			case "/wiki/api/v2/pages/3":
				t.Fatalf("page 3 must not be exported when root-page-id=1")
			case "/wiki/api/v2/folders/f1":
				writeJSON(t, w, `{"id":"f1","title":"Folder 2-2","spaceId":"SPACE-1","parentId":"1","parentType":"page"}`)
			case attachmentPath:
				writeJSON(t, w, `{"results":[{"id":"att-1","title":"logo.png","downloadLink":"/download/attachments/1/logo.png"}]}`)
			case "/download/attachments/1/logo.png", "/wiki/download/attachments/1/logo.png":
				_, _ = w.Write([]byte("PNGDATA"))
			default:
				http.NotFound(w, r)
			}
		}))

		outDir := t.TempDir()
		err := runMigrateExportForTest(t, &bytes.Buffer{}, &migrateExportOptions{
			SpaceKey:       "WORK",
			RootPageID:     "1",
			Out:            outDir,
			AttachmentsDir: "attachments/custom",
		}, srv.URL)
		if err != nil {
			t.Fatalf("RunMigrateExportWithConfig: %v", err)
		}

		assertPathNotExists(t, filepath.Join(outDir, "outside.md"))
		assertContainsAll(t, mustReadFileString(t, filepath.Join(outDir, "root", "_index.md")), `attachments/custom/1/logo.png`)
		assertPathExists(t, filepath.Join(outDir, "root", "folder-2-2", "child.md"))
	})

	t.Run("warns and continues on attachment 404", func(t *testing.T) {
		attachmentPath := "/wiki/api/v2/pages/1/attachments"
		srv := setupPageListServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/wiki/api/v2/spaces":
				writeJSON(t, w, `{"results":[{"id":"SPACE-1","key":"WORK"}]}`)
			case "/wiki/api/v2/spaces/SPACE-1/pages":
				writeJSON(t, w, `{"results":[{"id":"1","title":"Root Page","status":"current","spaceId":"SPACE-1"},{"id":"2","title":"Child Page","status":"current","spaceId":"SPACE-1","parentId":"1"}],"_links":{}}`)
			case "/wiki/api/v2/pages/1":
				writeJSON(t, w, `{"id":"1","title":"Root Page","status":"current","spaceId":"SPACE-1","body":{"storage":{"representation":"storage","value":"<p>Intro</p><ac:image><ri:attachment ri:filename=\"cfl-mermaid-001.svg\" /></ac:image>"}}}`)
			case "/wiki/api/v2/pages/2":
				writeJSON(t, w, `{"id":"2","title":"Child Page","status":"current","spaceId":"SPACE-1","parentId":"1","body":{"storage":{"representation":"storage","value":"<p>Child body</p>"}}}`)
			case attachmentPath:
				writeJSON(t, w, `{"results":[{"id":"att-1","title":"cfl-mermaid-001.svg","downloadLink":"/download/attachments/1/cfl-mermaid-001.svg"}]}`)
			case "/download/attachments/1/cfl-mermaid-001.svg", "/wiki/download/attachments/1/cfl-mermaid-001.svg":
				http.NotFound(w, r)
			default:
				http.NotFound(w, r)
			}
		}))

		outDir := t.TempDir()
		var out bytes.Buffer
		err := runMigrateExportForTest(t, &out, &migrateExportOptions{
			SpaceKey: "WORK",
			Out:      outDir,
		}, srv.URL)
		if err != nil {
			t.Fatalf("RunMigrateExportWithConfig: %v", err)
		}

		rootMarkdown := mustReadFileString(t, filepath.Join(outDir, "root-page", "_index.md"))
		assertContainsAll(t, rootMarkdown, `attachments/_migrate/1/cfl-mermaid-001.svg`)
		assertPathExists(t, filepath.Join(outDir, "root-page", "child-page.md"))
		assertPathNotExists(t, filepath.Join(outDir, "attachments", "_migrate", "1", "cfl-mermaid-001.svg"))
		assertContainsAll(t, out.String(),
			`Exported 2 pages`,
			`Warnings (1):`,
			`download attachment "cfl-mermaid-001.svg" for page "1" skipped: 404 Not Found`,
		)
	})

	t.Run("requires out", func(t *testing.T) {
		err := RunMigrateExportWithConfig(&bytes.Buffer{}, &migrateExportOptions{
			SpaceKey: "WORK",
		}, newPageListConfig("example.atlassian.net", "WORK"))
		if err == nil || !strings.Contains(err.Error(), "--out is required") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func runMigrateExportForTest(t *testing.T, out *bytes.Buffer, opts *migrateExportOptions, baseURL string) error {
	t.Helper()
	t.Setenv("CFL_API_TOKEN", "token")
	setOutputMode(t, "table")
	return RunMigrateExportWithConfig(out, opts, newPageListConfig(baseURL, "WORK"))
}

func writeJSON(t *testing.T, w http.ResponseWriter, body string) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(body))
}

func mustReadFileString(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", path, err)
	}
	return string(b)
}

func assertContainsAll(t *testing.T, got string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if strings.Contains(got, want) {
			continue
		}
		t.Fatalf("missing substring %q in %q", want, got)
	}
}

func assertPathExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected path %q to exist: %v", path, err)
	}
}

func assertPathNotExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected path %q to be absent: err=%v", path, err)
	}
}
