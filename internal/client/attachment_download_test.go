package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/takymt/cflcli/internal/config"
)

func TestDownloadPageAttachmentByFilename(t *testing.T) {
	var gotFilenameQuery string
	var gotStatusQuery string
	attachmentPath := "/wiki/api/v2/pages/123/attachments"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case attachmentPath:
			gotFilenameQuery = r.URL.Query().Get("filename")
			gotStatusQuery = r.URL.Query().Get("status")
			assertBasicAuth(t, r, "u@example.com", "token")
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"results":[{"id":"att-1","title":"logo.png","downloadLink":"/download/attachments/123/logo.png"}]}`))
		case "/download/attachments/123/logo.png", "/wiki/download/attachments/123/logo.png":
			assertBasicAuth(t, r, "u@example.com", "token")
			_, _ = w.Write([]byte("PNGDATA"))
		default:
			http.NotFound(w, r)
		}
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

	content, err := cli.DownloadPageAttachmentByFilename("123", "logo.png")
	if err != nil {
		t.Fatalf("DownloadPageAttachmentByFilename: %v", err)
	}
	if gotFilenameQuery != "logo.png" {
		t.Fatalf("filename query=%q want %q", gotFilenameQuery, "logo.png")
	}
	if gotStatusQuery != "current" {
		t.Fatalf("status query=%q want %q", gotStatusQuery, "current")
	}
	if string(content) != "PNGDATA" {
		t.Fatalf("content=%q want %q", string(content), "PNGDATA")
	}
}
