package page

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPRemoteResolveRootPageIDLiveSmoke(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wiki/api/v2/spaces/100" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if user, pass, ok := r.BasicAuth(); !ok || user != "user@example.com" || pass != "token" {
			t.Fatal("missing basic auth")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":         "100",
			"homepageId": "200",
		})
	}))
	defer server.Close()

	remote := NewHTTPRemote(server.URL, "user@example.com", "token", server.Client())
	rootID, err := remote.ResolveRootPageID(context.Background(), "100")
	if err != nil {
		t.Fatalf("ResolveRootPageID returned error: %v", err)
	}
	if rootID != "200" {
		t.Fatalf("unexpected root id: %s", rootID)
	}
}

func TestHTTPRemotePageExistsLiveSmoke(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wiki/api/v2/spaces/100/pages" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("title"); got != "guide" {
			t.Fatalf("unexpected title query: %s", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"results": []map[string]any{
				{"id": "1", "parentId": "999", "title": "guide"},
				{"id": "2", "parentId": "200", "title": "guide"},
			},
		})
	}))
	defer server.Close()

	remote := NewHTTPRemote(server.URL, "user@example.com", "token", server.Client())
	exists, err := remote.PageExists(context.Background(), "100", "200", "guide")
	if err != nil {
		t.Fatalf("PageExists returned error: %v", err)
	}
	if !exists {
		t.Fatal("expected page to exist")
	}
}

func TestHTTPRemoteCreateAndUpdatePageLiveSmoke(t *testing.T) {
	t.Parallel()

	var sawCreate bool
	var sawUpdate bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/wiki/api/v2/pages":
			sawCreate = true
			var req map[string]any
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("Decode returned error: %v", err)
			}
			if req["title"] != "guide" {
				t.Fatalf("unexpected title: %#v", req["title"])
			}
			body := req["body"].(map[string]any)
			if body["representation"] != "storage" {
				t.Fatalf("unexpected representation: %#v", body["representation"])
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":       "123",
				"spaceId":  "100",
				"parentId": "200",
				"title":    "guide",
				"version":  map[string]any{"number": 1},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/wiki/api/v2/pages/123":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":       "123",
				"spaceId":  "100",
				"parentId": "200",
				"title":    "guide",
				"version":  map[string]any{"number": 1},
			})
		case r.Method == http.MethodPut && r.URL.Path == "/wiki/api/v2/pages/123":
			sawUpdate = true
			var req map[string]any
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("Decode returned error: %v", err)
			}
			version := req["version"].(map[string]any)
			if version["number"].(float64) != 2 {
				t.Fatalf("unexpected version: %#v", version["number"])
			}
			body := req["body"].(map[string]any)
			if !strings.Contains(body["value"].(string), "<p>updated</p>") {
				t.Fatalf("unexpected body: %#v", body["value"])
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":       "123",
				"spaceId":  "100",
				"parentId": "200",
				"title":    "guide",
				"version":  map[string]any{"number": 2},
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	remote := NewHTTPRemote(server.URL, "user@example.com", "token", server.Client())
	created, err := remote.CreatePage(context.Background(), CreatePageInput{
		SpaceID:  "100",
		ParentID: "200",
		Title:    "guide",
		Body:     "<p>created</p>",
	})
	if err != nil {
		t.Fatalf("CreatePage returned error: %v", err)
	}
	if created.ID != "123" || !strings.HasSuffix(created.URL, "pageId=123") {
		t.Fatalf("unexpected created page: %#v", created)
	}

	updated, err := remote.UpdatePage(context.Background(), UpdatePageInput{
		PageID:   "123",
		SpaceID:  "100",
		ParentID: "200",
		Title:    "guide",
		Body:     "<p>updated</p>",
	})
	if err != nil {
		t.Fatalf("UpdatePage returned error: %v", err)
	}
	if updated.Version != 2 {
		t.Fatalf("unexpected updated version: %d", updated.Version)
	}
	if !sawCreate || !sawUpdate {
		t.Fatalf("expected create and update requests, got create=%v update=%v", sawCreate, sawUpdate)
	}
}
