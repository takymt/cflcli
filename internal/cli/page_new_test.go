package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultPageNewFilename(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		title     string
		want      string
		wantError string
	}{
		{
			name:  "keeps spaces",
			title: "Architecture Overview",
			want:  "Architecture Overview.md",
		},
		{
			name:  "sanitizes path separators and invalid punctuation",
			title: "API: Auth/Login?",
			want:  "API Auth Login.md",
		},
		{
			name:      "reserved windows name",
			title:     "CON",
			wantError: "pass --path",
		},
		{
			name:      "only invalid characters",
			title:     "???///",
			wantError: "pass --path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := defaultPageNewFilename(tt.title)
			if tt.wantError != "" {
				if err == nil {
					t.Fatal("defaultPageNewFilename() error = nil, want non-nil")
				}
				if !strings.Contains(err.Error(), tt.wantError) {
					t.Fatalf("error = %q, want substring %q", err.Error(), tt.wantError)
				}
				return
			}
			if err != nil {
				t.Fatalf("defaultPageNewFilename() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("defaultPageNewFilename() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestValidatePageNewPath(t *testing.T) {
	t.Parallel()

	t.Run("fails when parent directory is missing", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		path, display, err := validatePageNewPath(dir, filepath.Join("docs", "guide.md"))
		if err == nil {
			t.Fatal("validatePageNewPath() error = nil, want non-nil")
		}
		if !strings.Contains(err.Error(), "parent directory") {
			t.Fatalf("error = %q, want parent directory message", err.Error())
		}
		if path != "" || display != "" {
			t.Fatalf("validatePageNewPath() = (%q, %q), want empty values on error", path, display)
		}
	})

	t.Run("accepts markdown path when parent exists", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		docsDir := filepath.Join(dir, "docs")
		if err := os.MkdirAll(docsDir, 0o755); err != nil {
			t.Fatalf("MkdirAll() error = %v", err)
		}
		path, display, err := validatePageNewPath(dir, filepath.Join("docs", "guide.md"))
		if err != nil {
			t.Fatalf("validatePageNewPath() error = %v", err)
		}
		if path != filepath.Join(dir, "docs", "guide.md") {
			t.Fatalf("path = %q, want %q", path, filepath.Join(dir, "docs", "guide.md"))
		}
		if display != filepath.Join("docs", "guide.md") {
			t.Fatalf("display = %q, want %q", display, filepath.Join("docs", "guide.md"))
		}
	})
}
