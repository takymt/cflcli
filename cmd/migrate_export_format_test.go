package cmd

import (
	"bytes"
	"strings"
	"testing"

	migratepkg "github.com/takymt/cflcli/internal/migrate"
)

func TestBuildMigrateExportResult(t *testing.T) {
	got := buildMigrateExportResult(&migratepkg.ExportResult{
		SpaceID:        "SPACE-1",
		SpaceKey:       "WORK",
		OutDir:         "/tmp/out",
		AttachmentsDir: "attachments/_migrate",
		Pages: []migratepkg.ExportedPage{
			{ID: "1", Title: "Root", File: "root/_index.md"},
			{ID: "2", Title: "Child", ParentID: "1", File: "root/child.md"},
		},
		Warnings: []migratepkg.ExportWarning{
			{Message: "first warning"},
			{Message: " "},
			{Message: "second warning"},
		},
	})

	if got.SpaceID != "SPACE-1" || got.SpaceKey != "WORK" || got.Out != "/tmp/out" || got.AttachmentsDir != "attachments/_migrate" {
		t.Fatalf("metadata mismatch: %+v", got)
	}
	if len(got.Pages) != 2 {
		t.Fatalf("pages len=%d want 2", len(got.Pages))
	}
	if got.Pages[1].ParentID != "1" || got.Pages[1].File != "root/child.md" {
		t.Fatalf("page mapping mismatch: %+v", got.Pages[1])
	}
	if want := []string{"first warning", "second warning"}; len(got.Warnings) != len(want) || got.Warnings[0] != want[0] || got.Warnings[1] != want[1] {
		t.Fatalf("warnings=%v want %v", got.Warnings, want)
	}
}

func TestWriteMigrateExportTable(t *testing.T) {
	tests := []struct {
		name   string
		result migrateExportResult
		wants  []string
	}{
		{
			name: "without warnings",
			result: migrateExportResult{
				Out:   "/tmp/out",
				Pages: []migrateExportedPage{{ID: "1"}},
			},
			wants: []string{
				`Exported 1 pages to "/tmp/out".`,
			},
		},
		{
			name: "with warnings",
			result: migrateExportResult{
				Out:      "/tmp/out",
				Pages:    []migrateExportedPage{{ID: "1"}, {ID: "2"}},
				Warnings: []string{"warn 1", "warn 2"},
			},
			wants: []string{
				`Exported 2 pages to "/tmp/out".`,
				`Warnings (2):`,
				`- warn 1`,
				`- warn 2`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			if err := writeMigrateExportTable(&out, &tt.result); err != nil {
				t.Fatalf("writeMigrateExportTable: %v", err)
			}

			got := out.String()
			for _, want := range tt.wants {
				if strings.Contains(got, want) {
					continue
				}
				t.Fatalf("missing %q in output %q", want, got)
			}
		})
	}
}
