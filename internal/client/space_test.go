package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/takymt/cflcli/internal/config"
)

func TestResolveSpaceKeyByID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wiki/api/v2/spaces/SPACE-1" {
			http.NotFound(w, r)
			return
		}
		assertBasicAuth(t, r, "u@example.com", "token")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"SPACE-1","key":"WORK"}`))
	}))
	defer srv.Close()

	old := DefaultHTTPClient
	DefaultHTTPClient = srv.Client()
	t.Cleanup(func() { DefaultHTTPClient = old })

	cli, err := New(
		context.Background(),
		&config.Profile{Name: "work", Domain: srv.URL, User: "u@example.com"},
		"token",
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	spaceKey, err := cli.ResolveSpaceKeyByID("SPACE-1")
	if err != nil {
		t.Fatalf("ResolveSpaceKeyByID: %v", err)
	}
	if spaceKey != "WORK" {
		t.Fatalf("spaceKey=%q want %q", spaceKey, "WORK")
	}
}
