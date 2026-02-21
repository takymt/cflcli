package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestRunPageUpdateWithConfig_JSON_AutoVersion(t *testing.T) {
	var gotVersion int
	var gotBodyValue string

	srv := setupPageListServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/wiki/api/v2/pages/123":
			if r.Method == http.MethodGet {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"id":"123","title":"Old","status":"current","spaceId":"SPACE-1","version":{"number":7},"body":{"storage":{"representation":"storage","value":"<p>old</p>"}}}`))
				return
			}
			if r.Method == http.MethodPut {
				var payload struct {
					Version struct {
						Number int `json:"number"`
					} `json:"version"`
					Body struct {
						Storage struct {
							Value string `json:"value"`
						} `json:"storage"`
					} `json:"body"`
				}
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					t.Fatalf("decode update payload: %v", err)
				}
				gotVersion = payload.Version.Number
				gotBodyValue = payload.Body.Storage.Value
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"id":"123","title":"Updated","status":"current","spaceId":"SPACE-1"}`))
				return
			}
			t.Fatalf("unexpected method: %s", r.Method)
		default:
			http.NotFound(w, r)
		}
	}))
	setOutputMode(t, "json")
	t.Setenv("CFL_API_TOKEN", "token")

	cfg := newPageListConfig(srv.URL, "WORK")
	opts := &pageUpdateOptions{
		PageID:   "123",
		Title:    "Updated",
		BodyFile: writeTempBodyFile(t, "updated **body**"),
	}

	var out bytes.Buffer
	if err := RunPageUpdateWithConfig(&out, opts, cfg); err != nil {
		t.Fatalf("RunPageUpdateWithConfig: %v", err)
	}
	if gotVersion != 8 {
		t.Fatalf("version=%d want 8", gotVersion)
	}
	if !strings.Contains(gotBodyValue, "<strong>body</strong>") {
		t.Fatalf("unexpected body conversion: %q", gotBodyValue)
	}

	var payload struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	}
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}
	if payload.ID != "123" || payload.Title != "Updated" {
		t.Fatalf("unexpected output: %+v", payload)
	}
}

func TestRunPageUpdateWithConfig_UsesFrontMatterTitleWhenFlagEmpty(t *testing.T) {
	var gotTitle string
	var gotVersion int

	srv := setupPageListServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/wiki/api/v2/pages/123":
			if r.Method == http.MethodGet {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"id":"123","title":"Old","status":"current","spaceId":"SPACE-1","version":{"number":7},"body":{"storage":{"representation":"storage","value":"<p>old</p>"}}}`))
				return
			}
			if r.Method == http.MethodPut {
				var payload struct {
					Title    string `json:"title"`
					ParentID string `json:"parentId"`
					Version  struct {
						Number int `json:"number"`
					} `json:"version"`
					Body struct {
						Storage struct {
							Value string `json:"value"`
						} `json:"storage"`
					} `json:"body"`
				}
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					t.Fatalf("decode update payload: %v", err)
				}
				gotTitle = payload.Title
				if payload.ParentID != "777" {
					t.Fatalf("parentId=%q want %q", payload.ParentID, "777")
				}
				gotVersion = payload.Version.Number
				if strings.Contains(payload.Body.Storage.Value, "Frontmatter Update") {
					t.Fatalf("frontmatter title leaked to body: %q", payload.Body.Storage.Value)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"id":"123","title":"Frontmatter Update","status":"current","spaceId":"SPACE-1"}`))
				return
			}
			t.Fatalf("unexpected method: %s", r.Method)
		default:
			http.NotFound(w, r)
		}
	}))
	setOutputMode(t, "table")
	t.Setenv("CFL_API_TOKEN", "token")

	cfg := newPageListConfig(srv.URL, "WORK")
	opts := &pageUpdateOptions{
		PageID: "123",
		BodyFile: writeTempBodyFile(t, strings.Join([]string{
			"---",
			"title: Frontmatter Update",
			"parent-id: 777",
			"---",
			"",
			"updated body",
		}, "\n")),
	}

	var out bytes.Buffer
	if err := RunPageUpdateWithConfig(&out, opts, cfg); err != nil {
		t.Fatalf("RunPageUpdateWithConfig: %v", err)
	}
	if gotTitle != "Frontmatter Update" {
		t.Fatalf("title=%q want %q", gotTitle, "Frontmatter Update")
	}
	if gotVersion != 8 {
		t.Fatalf("version=%d want 8", gotVersion)
	}
	if !strings.Contains(out.String(), `Updated page "Frontmatter Update" (id: "123").`) {
		t.Fatalf("unexpected output: %q", out.String())
	}
}

func TestRunPageUpdateWithConfig_Conflict(t *testing.T) {
	srv := setupPageListServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/wiki/api/v2/pages/123":
			if r.Method == http.MethodGet {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"id":"123","title":"Old","status":"current","spaceId":"SPACE-1","version":{"number":7},"body":{"storage":{"representation":"storage","value":"<p>old</p>"}}}`))
				return
			}
			if r.Method == http.MethodPut {
				w.WriteHeader(http.StatusConflict)
				_, _ = w.Write([]byte(`{"message":"version conflict"}`))
				return
			}
			t.Fatalf("unexpected method: %s", r.Method)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Setenv("CFL_API_TOKEN", "token")

	cfg := newPageListConfig(srv.URL, "WORK")
	opts := &pageUpdateOptions{
		PageID:   "123",
		Title:    "Updated",
		BodyFile: writeTempBodyFile(t, "updated"),
	}

	err := RunPageUpdateWithConfig(&bytes.Buffer{}, opts, cfg)
	if err == nil || !strings.Contains(err.Error(), "update conflict") {
		t.Fatalf("unexpected error: %v", err)
	}
}
