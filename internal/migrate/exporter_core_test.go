package migrate

import (
	"errors"
	"net/http"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/takymt/cflcli/internal/client"
	"github.com/takymt/cflcli/internal/model"
)

func TestBuildPageFileMap(t *testing.T) {
	pages := map[string]client.Page{
		"1": {ID: "1", Title: "Root Page"},
		"2": {ID: "2", Title: "Child Page", ParentID: "1", ParentType: "page"},
		"3": {ID: "3", Title: "Folder Child", ParentID: "f1", ParentType: "folder"},
		"4": {ID: "4", Title: "Leaf Root"},
	}
	folders := &folderResolver{
		cache: map[string]*client.Folder{
			"f1": {
				ID:         "f1",
				Title:      "Folder 2-2",
				ParentID:   "1",
				ParentType: "page",
			},
		},
	}

	got, err := buildPageFileMap(pages, folders)
	if err != nil {
		t.Fatalf("buildPageFileMap: %v", err)
	}

	want := map[string]string{
		"1": "root-page/_index.md",
		"2": "root-page/child-page.md",
		"3": "root-page/folder-2-2/folder-child.md",
		"4": "leaf-root.md",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("buildPageFileMap mismatch\n got=%s\nwant=%s", formatStringMap(got), formatStringMap(want))
	}
}

func TestRenderExportPage(t *testing.T) {
	detail := &client.PageDetail{
		ID:       "1409064",
		Title:    "mermaid js",
		ParentID: "1",
		Body: model.PageBody{
			Storage: model.BodyType{
				Representation: "storage",
				Value:          `<h2>mermaid js</h2><p><ac:image ac:alt="mermaid-1"><ri:attachment ri:filename="cfl-mermaid-001.svg" /></ac:image></p>`,
			},
		},
	}

	got, err := renderExportPage(detail, "root-page/folder-2-2/mermaid-js.md", "attachments/custom", "WORK")
	if err != nil {
		t.Fatalf("renderExportPage: %v", err)
	}

	if got.page.File != "root-page/folder-2-2/mermaid-js.md" {
		t.Fatalf("page.File=%q", got.page.File)
	}
	if got.page.ID != "1409064" || got.page.ParentID != "1" {
		t.Fatalf("page metadata mismatch: %+v", got.page)
	}
	if !reflect.DeepEqual(got.attachments, []string{"cfl-mermaid-001.svg"}) {
		t.Fatalf("attachments=%v want [cfl-mermaid-001.svg]", got.attachments)
	}

	content := string(got.content)
	for _, want := range []string{
		`page-id: "1409064"`,
		`title: "mermaid js"`,
		`parent-id: "1"`,
		`space-key: "WORK"`,
		`../../attachments/custom/1409064/cfl-mermaid-001.svg`,
	} {
		if strings.Contains(content, want) {
			continue
		}
		t.Fatalf("missing %q in rendered content: %q", want, content)
	}
}

func TestRelativeAttachmentPathForMarkdown(t *testing.T) {
	tests := []struct {
		name           string
		pageDir        string
		attachmentsDir string
		pageID         string
		filename       string
		want           string
	}{
		{
			name:           "page under top-level page directory",
			pageDir:        "root-page",
			attachmentsDir: "attachments/_migrate",
			pageID:         "1",
			filename:       "logo.png",
			want:           "../attachments/_migrate/1/logo.png",
		},
		{
			name:           "page under nested folder",
			pageDir:        "root-page/folder-2-2",
			attachmentsDir: "attachments/custom",
			pageID:         "1409064",
			filename:       "cfl-mermaid-001.svg",
			want:           "../../attachments/custom/1409064/cfl-mermaid-001.svg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := relativeAttachmentPathForMarkdown(tt.pageDir, tt.attachmentsDir, tt.pageID, tt.filename); got != tt.want {
				t.Fatalf("relativeAttachmentPathForMarkdown()=%q want %q", got, tt.want)
			}
		})
	}
}

func TestClassifyAttachmentDownloadError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantWarn   string
		wantErrSub string
	}{
		{
			name:     "nil error",
			err:      nil,
			wantWarn: "",
		},
		{
			name: "404 becomes warning",
			err: &client.HTTPError{
				StatusCode: http.StatusNotFound,
				Status:     "404 Not Found",
				Body:       "not found",
			},
			wantWarn: `download attachment "cfl-mermaid-001.svg" for page "1409064" skipped: 404 Not Found`,
		},
		{
			name: "500 stays fatal",
			err: &client.HTTPError{
				StatusCode: http.StatusInternalServerError,
				Status:     "500 Internal Server Error",
				Body:       "boom",
			},
			wantErrSub: `download attachment "cfl-mermaid-001.svg" for page "1409064"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warning, err := classifyAttachmentDownloadError("1409064", "cfl-mermaid-001.svg", tt.err)

			if tt.wantWarn != "" {
				if err != nil {
					t.Fatalf("classifyAttachmentDownloadError err=%v want warning", err)
				}
				if warning == nil || warning.Message != tt.wantWarn {
					t.Fatalf("warning=%v want %q", warning, tt.wantWarn)
				}
				return
			}

			if tt.wantErrSub != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErrSub) {
					t.Fatalf("err=%v want contains %q", err, tt.wantErrSub)
				}
				var httpErr *client.HTTPError
				if !errors.As(err, &httpErr) || httpErr.StatusCode != http.StatusInternalServerError {
					t.Fatalf("wrapped error mismatch: %v", err)
				}
				if warning != nil {
					t.Fatalf("warning=%v want nil", warning)
				}
				return
			}

			if err != nil || warning != nil {
				t.Fatalf("warning=%v err=%v want nil,nil", warning, err)
			}
		})
	}
}

func formatStringMap(m map[string]string) string {
	if len(m) == 0 {
		return "{}"
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	b.WriteString("{")
	for i, k := range keys {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(k)
		b.WriteString(":")
		b.WriteString(m[k])
	}
	b.WriteString("}")
	return b.String()
}
