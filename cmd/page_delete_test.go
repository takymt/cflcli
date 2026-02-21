package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestRunPageDeleteWithConfig_Table(t *testing.T) {
	srv := setupPageListServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("method=%q", r.Method)
		}
		if r.URL.Path != "/wiki/api/v2/pages/123" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	setOutputMode(t, "table")
	t.Setenv("CFL_API_TOKEN", "token")

	cfg := newPageListConfig(srv.URL, "WORK")

	var out bytes.Buffer
	if err := RunPageDeleteWithConfig(&out, "123", cfg); err != nil {
		t.Fatalf("RunPageDeleteWithConfig: %v", err)
	}

	if !strings.Contains(out.String(), `Deleted page "123".`) {
		t.Fatalf("unexpected output: %q", out.String())
	}
}

func TestRunPageDeleteWithConfig_JSON(t *testing.T) {
	srv := setupPageListServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("method=%q", r.Method)
		}
		if r.URL.Path != "/wiki/api/v2/pages/123" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	setOutputMode(t, "json")
	t.Setenv("CFL_API_TOKEN", "token")

	cfg := newPageListConfig(srv.URL, "WORK")

	var out bytes.Buffer
	if err := RunPageDeleteWithConfig(&out, "123", cfg); err != nil {
		t.Fatalf("RunPageDeleteWithConfig: %v", err)
	}

	var payload struct {
		ID      string `json:"id"`
		Deleted bool   `json:"deleted"`
	}
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}
	if payload.ID != "123" || !payload.Deleted {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}

func TestRunPageDeleteWithConfig_NotFound(t *testing.T) {
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

	err := RunPageDeleteWithConfig(&bytes.Buffer{}, "999", cfg)
	if err == nil || !strings.Contains(err.Error(), `page "999" not found`) {
		t.Fatalf("unexpected error: %v", err)
	}
}
