package client

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/takymt/cflcli/internal/config"
)

func TestUpsertPageAttachment_RequestAndAuth(t *testing.T) {
	var gotFilename string
	var gotFileContent string
	var gotMinorEdit string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("method=%q", r.Method)
		}
		if r.URL.Path != "/wiki/rest/api/content/123/child/attachment" {
			http.NotFound(w, r)
			return
		}
		if got := r.Header.Get("X-Atlassian-Token"); got != "no-check" {
			t.Fatalf("X-Atlassian-Token=%q", got)
		}
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data; boundary=") {
			t.Fatalf("unexpected content-type=%q", r.Header.Get("Content-Type"))
		}
		assertBasicAuth(t, r, "u@example.com", "token")

		mr, err := r.MultipartReader()
		if err != nil {
			t.Fatalf("MultipartReader: %v", err)
		}
		for {
			part, err := mr.NextPart()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				t.Fatalf("NextPart: %v", err)
			}
			body, err := io.ReadAll(part)
			if err != nil {
				t.Fatalf("ReadAll(part): %v", err)
			}

			switch part.FormName() {
			case "file":
				gotFilename = part.FileName()
				gotFileContent = string(body)
			case "minorEdit":
				gotMinorEdit = string(body)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[{"id":"att-1","title":"logo.png"}]}`))
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

	sourcePath := filepath.Join(t.TempDir(), "logo.png")
	if err := os.WriteFile(sourcePath, []byte("PNGDATA"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if err := cli.UpsertPageAttachment("123", "logo.png", sourcePath); err != nil {
		t.Fatalf("UpsertPageAttachment: %v", err)
	}
	if gotFilename != "logo.png" {
		t.Fatalf("filename=%q want %q", gotFilename, "logo.png")
	}
	if gotFileContent != "PNGDATA" {
		t.Fatalf("file content=%q want %q", gotFileContent, "PNGDATA")
	}
	if gotMinorEdit != "true" {
		t.Fatalf("minorEdit=%q want %q", gotMinorEdit, "true")
	}
}
