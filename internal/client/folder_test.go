package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/takymt/cflcli/internal/config"
)

func TestGetFolder(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wiki/api/v2/folders/f1" {
			http.NotFound(w, r)
			return
		}
		assertBasicAuth(t, r, "u@example.com", "token")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"f1","title":"Folder 2-2","spaceId":"S1","parentId":"1","parentType":"page"}`))
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

	folder, err := cli.GetFolder("f1")
	if err != nil {
		t.Fatalf("GetFolder: %v", err)
	}
	if folder.ID != "f1" || folder.Title != "Folder 2-2" || folder.ParentID != "1" || folder.ParentType != "page" {
		t.Fatalf("unexpected folder: %+v", folder)
	}
}
