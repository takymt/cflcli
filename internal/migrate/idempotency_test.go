package migrate

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/takymt/cflcli/internal/body"
)

func TestStorageMarkdownStorage_IdempotencyAcceptanceCases(t *testing.T) {
	testCases := []struct {
		name        string
		storage     string
		wantMarkers []string
	}{
		{
			name: "mermaid macro survives roundtrip",
			storage: `<ac:structured-macro ac:name="code">` +
				`<ac:parameter ac:name="language">mermaid</ac:parameter>` +
				`<ac:plain-text-body><![CDATA[flowchart TD` + "\n" +
				`A --> B` + "\n" +
				`]]></ac:plain-text-body>` +
				`</ac:structured-macro>`,
			wantMarkers: []string{
				"```mermaid",
				"flowchart TD",
			},
		},
		{
			name: "unsupported macro trace remains",
			storage: `<ac:structured-macro ac:name="toc">` +
				`<ac:parameter ac:name="maxLevel">2</ac:parameter>` +
				`</ac:structured-macro>`,
			wantMarkers: []string{
				"cfl:migrate-unsupported-macro",
				`name="toc"`,
			},
		},
		{
			name:    "attachment link path remains under attachments",
			storage: `<ac:image><ri:attachment ri:filename="logo.png" /></ac:image>`,
			wantMarkers: []string{
				"attachments/_migrate/123/logo.png",
			},
		},
	}

	attachmentPath := func(filename string) string {
		return filepath.ToSlash(filepath.Join("attachments", "_migrate", "123", filename))
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			markdown1, _, err := StorageToMarkdown(tc.storage, attachmentPath)
			if err != nil {
				t.Fatalf("StorageToMarkdown first: %v", err)
			}

			storage1, err := body.ToStorage([]byte(markdown1), body.FormatMarkdown)
			if err != nil {
				t.Fatalf("body.ToStorage: %v", err)
			}

			markdown2, _, err := StorageToMarkdown(storage1, attachmentPath)
			if err != nil {
				t.Fatalf("StorageToMarkdown second: %v", err)
			}

			for _, marker := range tc.wantMarkers {
				if !strings.Contains(markdown2, marker) {
					t.Fatalf("markdown2 missing marker %q\nmarkdown1=%q\nstorage1=%q\nmarkdown2=%q", marker, markdown1, storage1, markdown2)
				}
			}
		})
	}
}
