package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/takymt/cflcli/internal/config"
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

func TestRunPageCreateWithConfig_RejectsReservedTitleIndex(t *testing.T) {
	srv := setupPageListServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/wiki/api/v2/spaces":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"results":[{"id":"SPACE-1","key":"WORK"}]}`))
		case "/wiki/api/v2/pages":
			t.Fatalf("create API must not be called when title is reserved")
		default:
			http.NotFound(w, r)
		}
	}))
	setOutputMode(t, "table")
	t.Setenv("CFL_API_TOKEN", "token")

	cfg := newPageListConfig(srv.URL, "WORK")
	opts := &pageCreateOptions{
		Title:    "_index",
		BodyFile: writeTempBodyFile(t, "body"),
	}

	err := RunPageCreateWithConfig(&bytes.Buffer{}, opts, cfg)
	if err == nil || !strings.Contains(err.Error(), `title "_index" is reserved`) {
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
	var gotUpdatePayload struct {
		ID      string `json:"id"`
		Title   string `json:"title"`
		SpaceID string `json:"spaceId"`
		Version struct {
			Number int `json:"number"`
		} `json:"version"`
		Body struct {
			Storage struct {
				Value string `json:"value"`
			} `json:"storage"`
		} `json:"body"`
	}
	var requestOrder []string
	attachmentPath := "/wiki/" + "rest/api/content/10/child/attachment"

	srv := setupPageListServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/wiki/api/v2/spaces":
			requestOrder = append(requestOrder, "spaces.get")
			gotSpacesQuery = r.URL.RawQuery
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"results":[{"id":"SPACE-P","key":"PROFILE"}]}`))
		case "/wiki/api/v2/pages":
			if r.Method == http.MethodPost {
				requestOrder = append(requestOrder, "pages.post")
				if err := json.NewDecoder(r.Body).Decode(&gotCreatePayload); err != nil {
					t.Fatalf("decode create payload: %v", err)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"id":"10","title":"Profile Doc","status":"current","spaceId":"SPACE-P"}`))
				return
			}
			t.Fatalf("method=%q", r.Method)
		case "/wiki/api/v2/pages/10":
			if r.Method == http.MethodGet {
				requestOrder = append(requestOrder, "pages.get")
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"id":"10","title":"Profile Doc","status":"current","spaceId":"SPACE-P","version":{"number":1},"body":{"storage":{"representation":"storage","value":"<p>initial</p>"}}}`))
				return
			}
			if r.Method == http.MethodPut {
				requestOrder = append(requestOrder, "pages.put")
				if err := json.NewDecoder(r.Body).Decode(&gotUpdatePayload); err != nil {
					t.Fatalf("decode update payload: %v", err)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"id":"10","title":"Profile Doc","status":"current","spaceId":"SPACE-P"}`))
				return
			}
			t.Fatalf("method=%q", r.Method)
		case attachmentPath:
			requestOrder = append(requestOrder, "attachment.put")
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
	if gotCreatePayload.Body.Storage.Value != initialCreateBody {
		t.Fatalf("create body must use placeholder when local assets exist: %q", gotCreatePayload.Body.Storage.Value)
	}
	if gotUpdatePayload.ID != "10" {
		t.Fatalf("update id=%q want %q", gotUpdatePayload.ID, "10")
	}
	if gotUpdatePayload.Version.Number != 2 {
		t.Fatalf("update version=%d want 2", gotUpdatePayload.Version.Number)
	}
	if !strings.Contains(gotUpdatePayload.Body.Storage.Value, `<ri:attachment ri:filename="logo.png" />`) {
		t.Fatalf("update body must include attachment reference: %q", gotUpdatePayload.Body.Storage.Value)
	}

	gotSequence := strings.Join(requestOrder, " -> ")
	wantSequence := "spaces.get -> pages.post -> attachment.put -> pages.get -> pages.put"
	if gotSequence != wantSequence {
		t.Fatalf("request order=%q want %q", gotSequence, wantSequence)
	}
}
