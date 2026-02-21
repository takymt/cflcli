package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/takymt/cflcli/internal/client"
	"github.com/takymt/cflcli/internal/config"
)

func setupPageListServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	originalHTTPClient := client.DefaultHTTPClient
	client.DefaultHTTPClient = srv.Client()
	t.Cleanup(func() { client.DefaultHTTPClient = originalHTTPClient })

	return srv
}

func setOutputMode(t *testing.T, mode string) {
	t.Helper()

	originalOutput := OutputFlag()
	SetOutputFlag(mode)
	t.Cleanup(func() { SetOutputFlag(originalOutput) })
}

func newPageListConfig(domain, spaceKey string) *config.Config {
	return &config.Config{
		Current: "work",
		Profiles: []config.Profile{
			{Name: "work", Domain: domain, User: "user@example.com", SpaceKey: spaceKey},
		},
	}
}

func writeTempBodyFile(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "body.xhtml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write body file: %v", err)
	}
	return path
}

func TestResolvePageLocalImageAssets(t *testing.T) {
	workspace := t.TempDir()

	bodyDir := filepath.Join(workspace, "docs")
	if err := os.MkdirAll(bodyDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(bodyDir): %v", err)
	}
	bodyFile := filepath.Join(bodyDir, "page.md")

	assetsRoot := filepath.Join(workspace, "assets")
	if err := os.MkdirAll(filepath.Join(assetsRoot, "images"), 0o755); err != nil {
		t.Fatalf("MkdirAll(assetsRoot): %v", err)
	}

	localImagePath := filepath.Join(bodyDir, "local.png")
	if err := os.WriteFile(localImagePath, []byte("LOCAL"), 0o600); err != nil {
		t.Fatalf("WriteFile(localImage): %v", err)
	}
	rootImagePath := filepath.Join(assetsRoot, "images", "root.png")
	if err := os.WriteFile(rootImagePath, []byte("ROOT"), 0o600); err != nil {
		t.Fatalf("WriteFile(rootImage): %v", err)
	}

	storage := strings.Join([]string{
		`<ac:image><ri:url ri:value="local.png" /></ac:image>`,
		`<ac:image ac:alt="root"><ri:url ri:value="/images/root.png" /></ac:image>`,
		`<ac:image><ri:url ri:value="https://example.com/logo.png" /></ac:image>`,
	}, "")

	gotStorage, gotAssets, err := resolvePageLocalImageAssets(storage, bodyFile, assetsRoot)
	if err != nil {
		t.Fatalf("resolvePageLocalImageAssets: %v", err)
	}

	if !strings.Contains(gotStorage, `<ri:attachment ri:filename="local.png" />`) {
		t.Fatalf("local image was not converted to attachment: %q", gotStorage)
	}
	if !strings.Contains(gotStorage, `<ri:attachment ri:filename="root.png" />`) {
		t.Fatalf("root-based image was not converted to attachment: %q", gotStorage)
	}
	if !strings.Contains(gotStorage, `<ri:url ri:value="https://example.com/logo.png" />`) {
		t.Fatalf("remote image should remain url reference: %q", gotStorage)
	}

	if len(gotAssets) != 2 {
		t.Fatalf("asset count=%d want 2 (%+v)", len(gotAssets), gotAssets)
	}
	if gotAssets[0].Filename != "local.png" {
		t.Fatalf("asset[0].Filename=%q want %q", gotAssets[0].Filename, "local.png")
	}
	if filepath.Clean(gotAssets[0].SourcePath) != filepath.Clean(localImagePath) {
		t.Fatalf("asset[0].SourcePath=%q want %q", gotAssets[0].SourcePath, localImagePath)
	}
	if gotAssets[1].Filename != "root.png" {
		t.Fatalf("asset[1].Filename=%q want %q", gotAssets[1].Filename, "root.png")
	}
	if filepath.Clean(gotAssets[1].SourcePath) != filepath.Clean(rootImagePath) {
		t.Fatalf("asset[1].SourcePath=%q want %q", gotAssets[1].SourcePath, rootImagePath)
	}
}

func TestResolvePageLocalImageAssets_DuplicateFilename(t *testing.T) {
	workspace := t.TempDir()
	bodyDir := filepath.Join(workspace, "docs")
	assetsRoot := filepath.Join(workspace, "assets")
	if err := os.MkdirAll(filepath.Join(bodyDir, "sub"), 0o755); err != nil {
		t.Fatalf("MkdirAll(bodyDir): %v", err)
	}
	if err := os.MkdirAll(filepath.Join(assetsRoot, "sub"), 0o755); err != nil {
		t.Fatalf("MkdirAll(assetsRoot): %v", err)
	}

	if err := os.WriteFile(filepath.Join(bodyDir, "logo.png"), []byte("A"), 0o600); err != nil {
		t.Fatalf("WriteFile(bodyDir/logo.png): %v", err)
	}
	if err := os.WriteFile(filepath.Join(assetsRoot, "sub", "logo.png"), []byte("B"), 0o600); err != nil {
		t.Fatalf("WriteFile(assetsRoot/sub/logo.png): %v", err)
	}

	storage := strings.Join([]string{
		`<ac:image><ri:url ri:value="logo.png" /></ac:image>`,
		`<ac:image><ri:url ri:value="/sub/logo.png" /></ac:image>`,
	}, "")

	_, _, err := resolvePageLocalImageAssets(storage, filepath.Join(bodyDir, "page.md"), assetsRoot)
	if err == nil || !strings.Contains(err.Error(), "duplicate image filename") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunPageCreateWithConfig_UploadsLocalImageAssets(t *testing.T) {
	workspace := t.TempDir()
	bodyDir := filepath.Join(workspace, "docs")
	if err := os.MkdirAll(filepath.Join(bodyDir, "img"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	imagePath := filepath.Join(bodyDir, "img", "logo.png")
	if err := os.WriteFile(imagePath, []byte("PNGDATA"), 0o600); err != nil {
		t.Fatalf("WriteFile(image): %v", err)
	}
	bodyFile := filepath.Join(bodyDir, "page.md")
	if err := os.WriteFile(bodyFile, []byte("![logo](./img/logo.png)\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(body): %v", err)
	}

	var attachmentCalls int
	var gotStorageValue string
	var gotUploadFilename string
	var gotUploadBody string

	srv := setupPageListServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/wiki/api/v2/spaces":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"results":[{"id":"SPACE-1","key":"WORK"}]}`))
		case "/wiki/api/v2/pages":
			var payload struct {
				Body struct {
					Storage struct {
						Value string `json:"value"`
					} `json:"storage"`
				} `json:"body"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode create payload: %v", err)
			}
			gotStorageValue = payload.Body.Storage.Value
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"10","title":"Local Image","status":"current","spaceId":"SPACE-1"}`))
		case "/wiki/rest/api/content/10/child/attachment":
			attachmentCalls++
			if r.Method != http.MethodPut {
				t.Fatalf("attachment method=%q", r.Method)
			}
			if got := r.Header.Get("X-Atlassian-Token"); got != "no-check" {
				t.Fatalf("X-Atlassian-Token=%q", got)
			}

			mr, err := r.MultipartReader()
			if err != nil {
				t.Fatalf("MultipartReader: %v", err)
			}
			for {
				part, err := mr.NextPart()
				if err != nil {
					break
				}
				body, err := io.ReadAll(part)
				if err != nil {
					t.Fatalf("ReadAll(part): %v", err)
				}
				if part.FormName() == "file" {
					gotUploadFilename = part.FileName()
					gotUploadBody = string(body)
				}
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"results":[{"id":"att-1","title":"logo.png"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	setOutputMode(t, "table")
	t.Setenv("CONFLUENCE_API_TOKEN", "token")

	opts := &pageCreateOptions{
		Title:    "Local Image",
		BodyFile: bodyFile,
	}
	if err := RunPageCreateWithConfig(&bytes.Buffer{}, opts, newPageListConfig(srv.URL, "WORK")); err != nil {
		t.Fatalf("RunPageCreateWithConfig: %v", err)
	}

	if attachmentCalls != 1 {
		t.Fatalf("attachmentCalls=%d want 1", attachmentCalls)
	}
	if !strings.Contains(gotStorageValue, `<ri:attachment ri:filename="logo.png" />`) {
		t.Fatalf("storage did not reference attachment: %q", gotStorageValue)
	}
	if gotUploadFilename != "logo.png" || gotUploadBody != "PNGDATA" {
		t.Fatalf("unexpected uploaded file: name=%q body=%q", gotUploadFilename, gotUploadBody)
	}
}

func TestRunPageUpdateWithConfig_UploadFailureStopsUpdate(t *testing.T) {
	workspace := t.TempDir()
	bodyDir := filepath.Join(workspace, "docs")
	if err := os.MkdirAll(filepath.Join(bodyDir, "img"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	imagePath := filepath.Join(bodyDir, "img", "logo.png")
	if err := os.WriteFile(imagePath, []byte("PNGDATA"), 0o600); err != nil {
		t.Fatalf("WriteFile(image): %v", err)
	}
	bodyFile := filepath.Join(bodyDir, "page.md")
	if err := os.WriteFile(bodyFile, []byte("![logo](./img/logo.png)\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(body): %v", err)
	}

	updateCalled := false
	srv := setupPageListServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/wiki/rest/api/content/123/child/attachment":
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"message":"upload failed"}`))
		case "/wiki/api/v2/pages/123":
			if r.Method == http.MethodPut {
				updateCalled = true
			}
			http.NotFound(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Setenv("CONFLUENCE_API_TOKEN", "token")

	opts := &pageUpdateOptions{
		PageID:   "123",
		Title:    "Updated",
		BodyFile: bodyFile,
	}
	err := RunPageUpdateWithConfig(&bytes.Buffer{}, opts, newPageListConfig(srv.URL, "WORK"))
	if err == nil || !strings.Contains(err.Error(), "upload local image assets") {
		t.Fatalf("unexpected error: %v", err)
	}
	if updateCalled {
		t.Fatalf("update API should not be called when attachment upload fails")
	}
}
