package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/takymt/cflcli/internal/config"
)

func TestListPagesBySpace_QueryAndAuth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wiki/api/v2/spaces/SPACE-1/pages" {
			http.NotFound(w, r)
			return
		}
		if got := r.URL.Query().Get("depth"); got != "all" {
			t.Fatalf("depth=%q", got)
		}
		if got := r.URL.Query().Get("status"); got != "current" {
			t.Fatalf("status=%q", got)
		}
		if got := r.URL.Query().Get("limit"); got != "25" {
			t.Fatalf("limit=%q", got)
		}
		if got := r.URL.Query().Get("cursor"); got != "CURSOR-1" {
			t.Fatalf("cursor=%q", got)
		}
		assertBasicAuth(t, r, "u@example.com", "token")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[{"id":"1","title":"A","status":"current","spaceId":"S1"}],"_links":{"next":"NEXT-1"}}`))
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

	result, err := cli.ListPagesBySpace("SPACE-1", 25, "CURSOR-1", "all", []string{"current"}, "")
	if err != nil {
		t.Fatalf("ListPagesBySpace: %v", err)
	}
	if len(result.Results) != 1 || result.Results[0].ID != "1" || result.Links.Next != "NEXT-1" {
		t.Fatalf("unexpected result: %+v", result)
	}
}
