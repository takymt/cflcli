package migrate

import (
	"os"
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

func TestMarkdownStorageMarkdown_DirectiveBlocksFixtureFixedPoint(t *testing.T) {
	t.Parallel()

	attachmentPath := func(filename string) string {
		return filepath.ToSlash(filepath.Join("attachments", "_migrate", "123", filename))
	}

	fixture, err := os.ReadFile("testdata/directive_roundtrip_fixture.md")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	markdown1, err := roundtripMarkdownFixture(fixture, attachmentPath)
	if err != nil {
		t.Fatalf("roundtrip #1: %v", err)
	}
	markdown2, err := roundtripMarkdownFixture([]byte(markdown1), attachmentPath)
	if err != nil {
		t.Fatalf("roundtrip #2: %v", err)
	}

	if markdown1 == markdown2 {
		t.Fatalf("directive fixture unexpectedly became a fixed point; this test currently documents non-idempotent behavior")
	}

	if !strings.Contains(markdown1, "cfl:migrate-unsupported-macro") {
		t.Fatalf("markdown1 should contain unsupported macro traces: %q", markdown1)
	}
	if !strings.Contains(markdown1, "<ac:adf-extension>") {
		t.Fatalf("markdown1 should contain raw ADF extension for :::memo: %q", markdown1)
	}
}

func roundtripMarkdownFixture(input []byte, attachmentPath func(filename string) string) (string, error) {
	storage, err := body.ToStorage(input, body.FormatMarkdown)
	if err != nil {
		return "", err
	}
	markdown, _, err := StorageToMarkdown(storage, attachmentPath)
	if err != nil {
		return "", err
	}
	return markdown, nil
}
