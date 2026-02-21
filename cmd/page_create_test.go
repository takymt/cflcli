package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/takymt/cflcli/internal/config"
	"github.com/takymt/cflcli/internal/mermaid"
)

func TestRunPageCreateWithConfig_Table_UsesSpaceKey(t *testing.T) {
	var gotSpacesQuery string
	var gotPayload struct {
		SpaceID string `json:"spaceId"`
		Body    struct {
			Storage struct {
				Value string `json:"value"`
			} `json:"storage"`
		} `json:"body"`
	}

	srv := setupPageListServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/wiki/api/v2/spaces":
			gotSpacesQuery = r.URL.RawQuery
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"results":[{"id":"SPACE-1","key":"WORK"}]}`))
		case "/wiki/api/v2/pages":
			if r.Method != http.MethodPost {
				t.Fatalf("method=%q", r.Method)
			}
			if err := json.NewDecoder(r.Body).Decode(&gotPayload); err != nil {
				t.Fatalf("decode create payload: %v", err)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"10","title":"New Doc","status":"current","spaceId":"SPACE-1"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	setOutputMode(t, "table")
	t.Setenv("CFL_API_TOKEN", "token")

	cfg := newPageListConfig(srv.URL, "WORK")
	opts := &pageCreateOptions{
		Title:    "New Doc",
		BodyFile: writeTempBodyFile(t, "Hello **world**"),
	}

	var out bytes.Buffer
	if err := RunPageCreateWithConfig(&out, opts, cfg); err != nil {
		t.Fatalf("RunPageCreateWithConfig: %v", err)
	}

	if !strings.Contains(gotSpacesQuery, "keys=WORK") {
		t.Fatalf("unexpected spaces query: %q", gotSpacesQuery)
	}
	if gotPayload.SpaceID != "SPACE-1" {
		t.Fatalf("unexpected payload: %+v", gotPayload)
	}
	if !strings.Contains(gotPayload.Body.Storage.Value, "<strong>world</strong>") {
		t.Fatalf("unexpected markdown conversion: %q", gotPayload.Body.Storage.Value)
	}

	raw := out.String()
	if !strings.Contains(raw, `Created page "New Doc" (id: "10").`) {
		t.Fatalf("unexpected output: %q", raw)
	}
}

func TestRunPageCreateWithConfig_UsesFrontMatterTitleWhenFlagEmpty(t *testing.T) {
	var gotPayload struct {
		Title    string `json:"title"`
		ParentID string `json:"parentId"`
		Body     struct {
			Storage struct {
				Value string `json:"value"`
			} `json:"storage"`
		} `json:"body"`
	}

	srv := setupPageListServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/wiki/api/v2/spaces":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"results":[{"id":"SPACE-1","key":"WORK"}]}`))
		case "/wiki/api/v2/pages":
			if r.Method != http.MethodPost {
				t.Fatalf("method=%q", r.Method)
			}
			if err := json.NewDecoder(r.Body).Decode(&gotPayload); err != nil {
				t.Fatalf("decode create payload: %v", err)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"10","title":"Frontmatter Title","status":"current","spaceId":"SPACE-1"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	setOutputMode(t, "table")
	t.Setenv("CFL_API_TOKEN", "token")

	cfg := newPageListConfig(srv.URL, "WORK")
	opts := &pageCreateOptions{
		BodyFile: writeTempBodyFile(t, strings.Join([]string{
			"---",
			"title: Frontmatter Title",
			"parent-id: 777",
			"---",
			"",
			"Hello **world**",
		}, "\n")),
	}

	var out bytes.Buffer
	if err := RunPageCreateWithConfig(&out, opts, cfg); err != nil {
		t.Fatalf("RunPageCreateWithConfig: %v", err)
	}

	if gotPayload.Title != "Frontmatter Title" {
		t.Fatalf("title=%q want %q", gotPayload.Title, "Frontmatter Title")
	}
	if gotPayload.ParentID != "777" {
		t.Fatalf("parentId=%q want %q", gotPayload.ParentID, "777")
	}
	if strings.Contains(gotPayload.Body.Storage.Value, "Frontmatter Title") {
		t.Fatalf("frontmatter title leaked to body: %q", gotPayload.Body.Storage.Value)
	}
	if !strings.Contains(gotPayload.Body.Storage.Value, "<strong>world</strong>") {
		t.Fatalf("unexpected markdown conversion: %q", gotPayload.Body.Storage.Value)
	}
	if !strings.Contains(out.String(), `Created page "Frontmatter Title" (id: "10").`) {
		t.Fatalf("unexpected output: %q", out.String())
	}
}

func TestRunPageCreateWithConfig_TitleSourcesAreMutuallyExclusive(t *testing.T) {
	srv := setupPageListServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/wiki/api/v2/spaces":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"results":[{"id":"SPACE-1","key":"WORK"}]}`))
		case "/wiki/api/v2/pages":
			t.Fatalf("create API must not be called when title sources conflict")
		default:
			http.NotFound(w, r)
		}
	}))
	setOutputMode(t, "table")
	t.Setenv("CFL_API_TOKEN", "token")

	cfg := newPageListConfig(srv.URL, "WORK")
	opts := &pageCreateOptions{
		Title: "CLI Title",
		BodyFile: writeTempBodyFile(t, strings.Join([]string{
			"---",
			"title: Frontmatter Title",
			"---",
			"",
			"body",
		}, "\n")),
	}

	err := RunPageCreateWithConfig(&bytes.Buffer{}, opts, cfg)
	if err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunPageCreateWithConfig_UsesProfileContentRoot(t *testing.T) {
	var gotSpacesQuery string
	var gotCreatePayload struct {
		SpaceID string `json:"spaceId"`
		Body    struct {
			Storage struct {
				Value string `json:"value"`
			} `json:"storage"`
		} `json:"body"`
	}
	attachmentPath := "/wiki/" + "rest/api/content/10/child/attachment"

	srv := setupPageListServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/wiki/api/v2/spaces":
			gotSpacesQuery = r.URL.RawQuery
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"results":[{"id":"SPACE-P","key":"PROFILE"}]}`))
		case "/wiki/api/v2/pages":
			if r.Method != http.MethodPost {
				t.Fatalf("method=%q", r.Method)
			}
			if err := json.NewDecoder(r.Body).Decode(&gotCreatePayload); err != nil {
				t.Fatalf("decode create payload: %v", err)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"10","title":"Profile Doc","status":"current","spaceId":"SPACE-P"}`))
		case attachmentPath:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"results":[{"id":"att-1","title":"logo.png"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	setOutputMode(t, "table")
	t.Setenv("CFL_API_TOKEN", "token")

	root := t.TempDir()
	bodyFile := filepath.Join(root, "docs", "page.md")
	contentRoot := filepath.Join(root, "assets")
	imageFile := filepath.Join(contentRoot, "images", "logo.png")
	if err := os.MkdirAll(filepath.Dir(bodyFile), 0o755); err != nil {
		t.Fatalf("MkdirAll(body): %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(imageFile), 0o755); err != nil {
		t.Fatalf("MkdirAll(image): %v", err)
	}
	if err := os.WriteFile(bodyFile, []byte("![logo](/images/logo.png)"), 0o600); err != nil {
		t.Fatalf("WriteFile(body): %v", err)
	}
	if err := os.WriteFile(imageFile, []byte("PNGDATA"), 0o600); err != nil {
		t.Fatalf("WriteFile(image): %v", err)
	}

	cfg := &config.Config{
		Current: "work",
		Profiles: []config.Profile{
			{
				Name:        "work",
				Domain:      srv.URL,
				User:        "user@example.com",
				SpaceKey:    "PROFILE",
				ContentRoot: contentRoot,
			},
		},
	}
	opts := &pageCreateOptions{
		Title:    "Profile Doc",
		BodyFile: bodyFile,
	}
	var out bytes.Buffer
	if err := RunPageCreateWithConfig(&out, opts, cfg); err != nil {
		t.Fatalf("RunPageCreateWithConfig: %v", err)
	}

	if !strings.Contains(gotSpacesQuery, "keys=PROFILE") {
		t.Fatalf("unexpected spaces query: %q", gotSpacesQuery)
	}
	if gotCreatePayload.SpaceID != "SPACE-P" {
		t.Fatalf("spaceId=%q want %q", gotCreatePayload.SpaceID, "SPACE-P")
	}
	if !strings.Contains(gotCreatePayload.Body.Storage.Value, `<ri:attachment ri:filename="logo.png" />`) {
		t.Fatalf("profile content_root did not resolve root image path: %q", gotCreatePayload.Body.Storage.Value)
	}
}

func TestRunPageCreateWithConfig_MermaidFenceUploadsRenderedSVG(t *testing.T) {
	var (
		gotCreateStorageBody string
		gotUploadFilename    string
		gotUploadContent     string
	)

	attachmentPath := "/wiki/" + "rest/api/content/10/child/attachment"
	srv := setupPageListServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/wiki/api/v2/spaces":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"results":[{"id":"SPACE-1","key":"WORK"}]}`))
		case "/wiki/api/v2/pages":
			if r.Method != http.MethodPost {
				t.Fatalf("method=%q", r.Method)
			}
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
			gotCreateStorageBody = payload.Body.Storage.Value
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"10","title":"Mermaid Doc","status":"current","spaceId":"SPACE-1"}`))
		case attachmentPath:
			if r.Method != http.MethodPut {
				t.Fatalf("attachment upload method=%q", r.Method)
			}
			reader, err := r.MultipartReader()
			if err != nil {
				t.Fatalf("MultipartReader: %v", err)
			}
			for {
				part, nextErr := reader.NextPart()
				if nextErr == io.EOF {
					break
				}
				if nextErr != nil {
					t.Fatalf("NextPart: %v", nextErr)
				}
				body, readErr := io.ReadAll(part)
				if readErr != nil {
					t.Fatalf("ReadAll(part): %v", readErr)
				}
				if part.FormName() == "file" {
					gotUploadFilename = part.FileName()
					gotUploadContent = string(body)
				}
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"results":[{"id":"att-1","title":"cfl-mermaid-001.svg"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	setOutputMode(t, "table")
	t.Setenv("CFL_API_TOKEN", "token")

	fakeRenderer := &fakeMermaidRenderer{
		renderFn: func(_ context.Context, source string) ([]byte, error) {
			if !strings.Contains(source, "flowchart TD") {
				t.Fatalf("unexpected mermaid source: %q", source)
			}
			return []byte(`<svg xmlns="http://www.w3.org/2000/svg"><text>diagram</text></svg>`), nil
		},
	}
	setMermaidRendererFactory(t, func() (mermaid.SVGRenderer, error) {
		return fakeRenderer, nil
	})

	cfg := newPageListConfig(srv.URL, "WORK")
	opts := &pageCreateOptions{
		Title: "Mermaid Doc",
		BodyFile: writeTempBodyFile(t, strings.Join([]string{
			"# Diagram",
			"",
			"```mermaid",
			"flowchart TD",
			"  A --> B",
			"```",
		}, "\n")),
	}

	var out bytes.Buffer
	if err := RunPageCreateWithConfig(&out, opts, cfg); err != nil {
		t.Fatalf("RunPageCreateWithConfig: %v", err)
	}

	if !fakeRenderer.closeCalled {
		t.Fatalf("mermaid renderer must be closed")
	}
	if !strings.Contains(gotCreateStorageBody, `<ri:attachment ri:filename="cfl-mermaid-001.svg" />`) {
		t.Fatalf("create body must reference rendered mermaid attachment: %q", gotCreateStorageBody)
	}
	if gotUploadFilename != "cfl-mermaid-001.svg" {
		t.Fatalf("uploaded filename=%q want %q", gotUploadFilename, "cfl-mermaid-001.svg")
	}
	if !strings.Contains(gotUploadContent, "<svg") {
		t.Fatalf("uploaded mermaid content must be svg: %q", gotUploadContent)
	}
}
