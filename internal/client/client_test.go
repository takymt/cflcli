package client

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/takymt/cflcli/internal/config"
)

func TestNew_Validation(t *testing.T) {
	profile := &config.Profile{Domain: "example.atlassian.net", User: "user@example.com"}
	if _, err := New(profile, ""); err == nil {
		t.Fatalf("expected error for missing token")
	}
}

func TestListPages(t *testing.T) {
	var gotAuth string
	var gotQuery string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotQuery = r.URL.RawQuery
		_ = json.NewEncoder(w).Encode(map[string]any{
			"results": []map[string]any{{
				"id":     "123",
				"title":  "Hello",
				"status": "current",
				"space":  map[string]any{"id": "S1"},
			}},
			"_links": map[string]any{"next": "next-cursor"},
		})
	}))
	defer server.Close()

	profile := &config.Profile{User: "user@example.com"}
	cli, err := NewWithBaseURL(profile, "token", server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, err := cli.ListPages("SPACE", 10, "CURSOR")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}
	if result.Results[0].ID != "123" {
		t.Errorf("expected id 123, got %s", result.Results[0].ID)
	}

	if gotQuery != "cursor=CURSOR&limit=10&space-id=SPACE" {
		t.Errorf("unexpected query: %s", gotQuery)
	}

	expectedAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("user@example.com:token"))
	if gotAuth != expectedAuth {
		t.Errorf("unexpected auth header: %s", gotAuth)
	}
}
