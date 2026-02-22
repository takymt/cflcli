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

func TestMarkdownStorageMarkdown_DirectiveBlockRoundTripMatrix(t *testing.T) {
	t.Parallel()

	attachmentPath := func(filename string) string {
		return filepath.ToSlash(filepath.Join("attachments", "_migrate", "123", filename))
	}

	testCases := []struct {
		name                  string
		input                 string
		wantDirectiveSurvives bool
		wantStableAfterFirst  bool
		wantMarkers           []string
		reason                string
	}{
		{
			name: "details maps to expand macro and loses directive syntax",
			input: strings.Join([]string{
				":::details 折りたたみタイトル",
				"折りたたみ本文1",
				"折りたたみ本文2",
				":::",
			}, "\n"),
			wantDirectiveSurvives: false,
			wantStableAfterFirst:  true,
			wantMarkers: []string{
				"cfl:migrate-unsupported-macro",
				`name="expand"`,
			},
			reason: "migrate export does not implement expand -> :::details reverse conversion",
		},
		{
			name: "info maps to info macro and loses directive syntax",
			input: strings.Join([]string{
				":::info",
				"情報",
				":::",
			}, "\n"),
			wantDirectiveSurvives: false,
			wantStableAfterFirst:  true,
			wantMarkers: []string{
				"cfl:migrate-unsupported-macro",
				`name="info"`,
			},
			reason: "migrate export treats info macro as unsupported",
		},
		{
			name: "success aliases to tip macro and loses original directive name",
			input: strings.Join([]string{
				":::success",
				"成功",
				":::",
			}, "\n"),
			wantDirectiveSurvives: false,
			wantStableAfterFirst:  true,
			wantMarkers: []string{
				"cfl:migrate-unsupported-macro",
				`name="tip"`,
			},
			reason: "create side normalizes success -> tip and export has no tip -> :::success reverse mapping",
		},
		{
			name: "memo emits adf panel extension and fallback note marker",
			input: strings.Join([]string{
				":::memo",
				"メモ",
				":::",
			}, "\n"),
			wantDirectiveSurvives: false,
			wantStableAfterFirst:  false,
			wantMarkers: []string{
				"<ac:adf-extension>",
				`name="note"`,
			},
			reason: "create side emits ADF panel extension for memo; export preserves raw ADF tags and the next markdown parse mutates some ac:* tags further",
		},
		{
			name: "warn aliases to note macro and loses directive syntax",
			input: strings.Join([]string{
				":::warn",
				"警告",
				":::",
			}, "\n"),
			wantDirectiveSurvives: false,
			wantStableAfterFirst:  true,
			wantMarkers: []string{
				"cfl:migrate-unsupported-macro",
				`name="note"`,
			},
			reason: "create side normalizes warn -> note and export treats note macro as unsupported",
		},
		{
			name: "error aliases to warning macro and loses directive syntax",
			input: strings.Join([]string{
				":::error",
				"エラー",
				":::",
			}, "\n"),
			wantDirectiveSurvives: false,
			wantStableAfterFirst:  true,
			wantMarkers: []string{
				"cfl:migrate-unsupported-macro",
				`name="warning"`,
			},
			reason: "create side normalizes error -> warning and export treats warning macro as unsupported",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			storage1, err := body.ToStorage([]byte(tc.input), body.FormatMarkdown)
			if err != nil {
				t.Fatalf("body.ToStorage first: %v", err)
			}

			markdown1, _, err := StorageToMarkdown(storage1, attachmentPath)
			if err != nil {
				t.Fatalf("StorageToMarkdown first: %v", err)
			}

			storage2, err := body.ToStorage([]byte(markdown1), body.FormatMarkdown)
			if err != nil {
				t.Fatalf("body.ToStorage second: %v", err)
			}

			markdown2, _, err := StorageToMarkdown(storage2, attachmentPath)
			if err != nil {
				t.Fatalf("StorageToMarkdown second: %v", err)
			}

			gotDirectiveSurvives := strings.Contains(markdown1, ":::")
			if gotDirectiveSurvives != tc.wantDirectiveSurvives {
				t.Fatalf("directive survival=%v want %v\nreason=%s\ninput=%q\nmarkdown1=%q", gotDirectiveSurvives, tc.wantDirectiveSurvives, tc.reason, tc.input, markdown1)
			}

			gotStableAfterFirst := markdown1 == markdown2
			if gotStableAfterFirst != tc.wantStableAfterFirst {
				t.Fatalf("stable-after-first=%v want %v\nreason=%s\nmarkdown1=%q\nmarkdown2=%q", gotStableAfterFirst, tc.wantStableAfterFirst, tc.reason, markdown1, markdown2)
			}

			for _, marker := range tc.wantMarkers {
				if !strings.Contains(markdown1, marker) {
					t.Fatalf("markdown1 missing marker %q\nreason=%s\nmarkdown1=%q", marker, tc.reason, markdown1)
				}
			}
		})
	}
}
