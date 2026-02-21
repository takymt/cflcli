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

func TestRunPageCreateWithConfig_JSON_UsesSpaceID(t *testing.T) {
	var gotPayload struct {
		SpaceID  string `json:"spaceId"`
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
			t.Fatalf("unexpected spaces lookup for explicit --space-id")
		case "/wiki/api/v2/pages":
			if r.Method != http.MethodPost {
				t.Fatalf("method=%q", r.Method)
			}
			if err := json.NewDecoder(r.Body).Decode(&gotPayload); err != nil {
				t.Fatalf("decode create payload: %v", err)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"11","title":"Child Doc","status":"current","spaceId":"SPACE-99"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	setOutputMode(t, "json")
	t.Setenv("CFL_API_TOKEN", "token")

	cfg := newPageListConfig(srv.URL, "WORK")
	opts := &pageCreateOptions{
		Title:      "Child Doc",
		BodyFile:   writeTempBodyFile(t, "<p>child</p>"),
		BodyFormat: "storage",
		ParentID:   "55",
		SpaceID:    "SPACE-99",
	}

	var out bytes.Buffer
	if err := RunPageCreateWithConfig(&out, opts, cfg); err != nil {
		t.Fatalf("RunPageCreateWithConfig: %v", err)
	}

	if gotPayload.SpaceID != "SPACE-99" || gotPayload.ParentID != "55" {
		t.Fatalf("unexpected payload: %+v", gotPayload)
	}
	if gotPayload.Body.Storage.Value != "<p>child</p>" {
		t.Fatalf("unexpected storage passthrough: %q", gotPayload.Body.Storage.Value)
	}

	var created struct {
		ID      string `json:"id"`
		Title   string `json:"title"`
		Status  string `json:"status"`
		SpaceID string `json:"spaceId"`
	}
	if err := json.Unmarshal(out.Bytes(), &created); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}
	if created.ID != "11" || created.Title != "Child Doc" || created.SpaceID != "SPACE-99" {
		t.Fatalf("unexpected output: %+v", created)
	}
}

func TestRunPageCreateWithConfig_Validation(t *testing.T) {
	t.Setenv("CFL_API_TOKEN", "token")

	cfg := newPageListConfig("example.atlassian.net", "WORK")
	testCases := []struct {
		name       string
		opts       *pageCreateOptions
		wantErrSub string
	}{
		{
			name: "missing title",
			opts: &pageCreateOptions{
				BodyFile: writeTempBodyFile(t, "<p>Hello</p>"),
			},
			wantErrSub: "--title is required",
		},
		{
			name: "missing body file",
			opts: &pageCreateOptions{
				Title: "Doc",
			},
			wantErrSub: "--body-file is required",
		},
		{
			name: "body file read error",
			opts: &pageCreateOptions{
				Title:    "Doc",
				BodyFile: filepath.Join(t.TempDir(), "missing.xhtml"),
			},
			wantErrSub: "read body file",
		},
		{
			name: "space selectors are mutually exclusive",
			opts: &pageCreateOptions{
				Title:    "Doc",
				BodyFile: writeTempBodyFile(t, "<p>Hello</p>"),
				SpaceID:  "1",
				SpaceKey: "WORK",
			},
			wantErrSub: "mutually exclusive",
		},
		{
			name: "invalid body format",
			opts: &pageCreateOptions{
				Title:      "Doc",
				BodyFile:   writeTempBodyFile(t, "<p>Hello</p>"),
				BodyFormat: "wiki",
			},
			wantErrSub: "invalid --body-format",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := RunPageCreateWithConfig(&bytes.Buffer{}, tc.opts, cfg)
			if err == nil || !strings.Contains(err.Error(), tc.wantErrSub) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
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

func TestRunPageCreateWithConfig_ParentIDSourcesAreMutuallyExclusive(t *testing.T) {
	t.Setenv("CFL_API_TOKEN", "token")

	cfg := newPageListConfig("example.atlassian.net", "WORK")
	opts := &pageCreateOptions{
		Title:    "Doc",
		ParentID: "123",
		BodyFile: writeTempBodyFile(t, strings.Join([]string{
			"---",
			"parent-id: 777",
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

func TestRunPageCreateWithConfig_InvalidFrontMatter(t *testing.T) {
	t.Setenv("CFL_API_TOKEN", "token")

	cfg := newPageListConfig("example.atlassian.net", "WORK")
	opts := &pageCreateOptions{
		BodyFile: writeTempBodyFile(t, strings.Join([]string{
			"---",
			"title: Broken",
			"body without closing delimiter",
		}, "\n")),
	}

	err := RunPageCreateWithConfig(&bytes.Buffer{}, opts, cfg)
	if err == nil || !strings.Contains(err.Error(), "invalid frontmatter") {
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
		case "/wiki/rest/api/content/10/child/attachment":
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
