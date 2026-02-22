package migrate

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/takymt/cflcli/internal/body"
)

func TestMarkdownStorageMarkdown_DirectiveBlocksFixedPoint(t *testing.T) {
	t.Parallel()

	attachmentPath := func(filename string) string {
		return filepath.ToSlash(filepath.Join("attachments", "_migrate", "123", filename))
	}

	input := []byte(strings.Join([]string{
		":::details 折りたたみタイトル",
		"折りたたみ本文1",
		"折りたたみ本文2",
		":::",
		"",
		":::info",
		"情報",
		":::",
		"",
		":::success",
		"成功",
		":::",
		"",
		":::memo",
		"メモ",
		":::",
		"",
		":::warn",
		"警告",
		":::",
		"",
		":::error",
		"エラー",
		":::",
	}, "\n"))

	markdown1, err := roundtripMarkdownFixture(input, attachmentPath)
	if err != nil {
		t.Fatalf("roundtrip #1: %v", err)
	}
	markdown2, err := roundtripMarkdownFixture([]byte(markdown1), attachmentPath)
	if err != nil {
		t.Fatalf("roundtrip #2: %v", err)
	}

	if markdown1 != markdown2 {
		t.Fatalf("directive fixture must become a fixed point after first roundtrip\nmarkdown1=%q\nmarkdown2=%q", markdown1, markdown2)
	}

	if strings.Contains(markdown1, "cfl:migrate-unsupported-macro") {
		t.Fatalf("markdown1 should not contain unsupported macro traces for supported directives: %q", markdown1)
	}
	if strings.Contains(markdown1, "<ac:adf-extension>") {
		t.Fatalf("markdown1 should not contain raw ADF extension for :::memo: %q", markdown1)
	}

	for _, want := range []string{
		":::details 折りたたみタイトル",
		":::info",
		":::success",
		":::memo",
	} {
		if !strings.Contains(markdown1, want) {
			t.Fatalf("markdown1 missing %q: %q", want, markdown1)
		}
	}
	if strings.Contains(markdown1, ":::error") {
		t.Fatalf("warning-family directives must canonicalize to :::warn: %q", markdown1)
	}
	if gotWarnCount := strings.Count(markdown1, ":::warn"); gotWarnCount != 2 {
		t.Fatalf(":::warn count=%d want 2 (warn + error should canonicalize to warn)\nmarkdown1=%q", gotWarnCount, markdown1)
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
