package cmd

import (
	"bytes"
	"net/http"
	"strings"
	"testing"
)

func TestRunPageGetWithConfig_WritesStorageBody(t *testing.T) {
	srv := setupPageListServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wiki/api/v2/pages/123" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"123","title":"Doc","status":"current","spaceId":"SPACE-1","body":{"storage":{"representation":"storage","value":"<p>Hello</p>"}}}`))
	}))

	t.Setenv("CFL_API_TOKEN", "token")

	cfg := newPageListConfig(srv.URL, "WORK")

	var out bytes.Buffer
	err := RunPageGetWithConfig(&out, "123", cfg)
	if err != nil {
		t.Fatalf("RunPageGetWithConfig: %v", err)
	}
	if out.String() != "<p>Hello</p>" {
		t.Fatalf("unexpected output: %q", out.String())
	}
}

func TestRunPageGetWithConfig_NotFound(t *testing.T) {
	srv := setupPageListServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wiki/api/v2/pages/999" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"not found"}`))
	}))

	t.Setenv("CFL_API_TOKEN", "token")

	cfg := newPageListConfig(srv.URL, "WORK")

	err := RunPageGetWithConfig(&bytes.Buffer{}, "999", cfg)
	if err == nil || !strings.Contains(err.Error(), `page "999" not found`) {
		t.Fatalf("unexpected error: %v", err)
	}
}
