package test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/takymt/cflcli/cmd"
	"github.com/takymt/cflcli/internal/client"
	"github.com/takymt/cflcli/internal/config"
)

func TestPageList_CLIOutputJSON(t *testing.T) {
	var gotQuery string
	var gotAuth string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		gotAuth = r.Header.Get("Authorization")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"results": []map[string]any{{
				"id":     "1",
				"title":  "Hello",
				"status": "current",
				"space":  map[string]any{"id": "S1"},
			}},
			"_links": map[string]any{"next": "cursor"},
		})
	}))
	defer server.Close()

	prevOutput := cmd.OutputFlag()
	prevProfile := cmd.ProfileFlag()
	prevToken := os.Getenv("CONFLUENCE_API_TOKEN")
	prevHTTPClient := client.DefaultHTTPClient
	defer func() {
		cmd.SetOutputFlag(prevOutput)
		cmd.SetProfileFlag(prevProfile)
		client.DefaultHTTPClient = prevHTTPClient
		if prevToken == "" {
			_ = os.Unsetenv("CONFLUENCE_API_TOKEN")
			return
		}
		_ = os.Setenv("CONFLUENCE_API_TOKEN", prevToken)
	}()

	_ = os.Setenv("CONFLUENCE_API_TOKEN", "token")
	cmd.SetOutputFlag("json")
	client.DefaultHTTPClient = server.Client()

	cfg := &config.Config{
		Current: "work",
		Profiles: []config.Profile{{
			Name:   "work",
			Domain: server.URL,
			User:   "user@example.com",
		}},
	}

	buf := &bytes.Buffer{}
	if err := cmd.RunPageListWithConfig(buf, &cmd.PageListOptions{SpaceID: "SPACE", Limit: 10}, cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotQuery != "limit=10&space-id=SPACE" {
		t.Fatalf("unexpected query: %s", gotQuery)
	}
	expectedAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("user@example.com:token"))
	if gotAuth != expectedAuth {
		t.Fatalf("unexpected auth: %s", gotAuth)
	}
	if buf.Len() == 0 {
		t.Fatalf("expected output")
	}
}
