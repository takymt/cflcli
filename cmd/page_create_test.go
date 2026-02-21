package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
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

func TestRunPageCreateWithConfig_UsesRepoConfigDefaults(t *testing.T) {
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
			_, _ = w.Write([]byte(`{"results":[{"id":"SPACE-R","key":"REPO"}]}`))
		case "/wiki/api/v2/pages":
			if r.Method != http.MethodPost {
				t.Fatalf("method=%q", r.Method)
			}
			if err := json.NewDecoder(r.Body).Decode(&gotCreatePayload); err != nil {
				t.Fatalf("decode create payload: %v", err)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"10","title":"Repo Doc","status":"current","spaceId":"SPACE-R"}`))
		case attachmentPath:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"results":[{"id":"att-1","title":"logo.png"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	setOutputMode(t, "table")
	t.Setenv("CFL_API_TOKEN", "token")

	repoDir := t.TempDir()
	writeRepoConfig(t, repoDir, fmt.Sprintf(
		"domain = %q\nspace_key = \"REPO\"\ncontent_root = \"assets\"\n",
		srv.URL,
	))
	bodyFile := filepath.Join(repoDir, "docs", "page.md")
	imageFile := filepath.Join(repoDir, "assets", "images", "logo.png")
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

	cfg := newPageListConfig("profile.example.atlassian.net", "PROFILE")
	opts := &pageCreateOptions{
		Title:    "Repo Doc",
		BodyFile: bodyFile,
	}
	var out bytes.Buffer
	if err := RunPageCreateWithConfig(&out, opts, cfg); err != nil {
		t.Fatalf("RunPageCreateWithConfig: %v", err)
	}

	if !strings.Contains(gotSpacesQuery, "keys=REPO") {
		t.Fatalf("unexpected spaces query: %q", gotSpacesQuery)
	}
	if gotCreatePayload.SpaceID != "SPACE-R" {
		t.Fatalf("spaceId=%q want %q", gotCreatePayload.SpaceID, "SPACE-R")
	}
	if !strings.Contains(gotCreatePayload.Body.Storage.Value, `<ri:attachment ri:filename="logo.png" />`) {
		t.Fatalf("repo content_root did not resolve root image path: %q", gotCreatePayload.Body.Storage.Value)
	}
}
