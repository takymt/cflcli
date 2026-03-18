package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizePageTitle(t *testing.T) {
	t.Parallel()

	t.Run("trims surrounding whitespace", func(t *testing.T) {
		t.Parallel()

		got, err := normalizePageTitle("  Guide Title  ")
		if err != nil {
			t.Fatalf("normalizePageTitle() error = %v", err)
		}
		if got != "Guide Title" {
			t.Fatalf("normalizePageTitle() = %q, want %q", got, "Guide Title")
		}
	})

	t.Run("rejects empty title", func(t *testing.T) {
		t.Parallel()

		_, err := normalizePageTitle("")
		if err == nil {
			t.Fatal("normalizePageTitle() error = nil, want non-nil")
		}
		if !strings.Contains(err.Error(), "must not be empty") {
			t.Fatalf("error = %q, want empty-title message", err.Error())
		}
	})

	t.Run("rejects whitespace only title", func(t *testing.T) {
		t.Parallel()

		_, err := normalizePageTitle("   \n\t  ")
		if err == nil {
			t.Fatal("normalizePageTitle() error = nil, want non-nil")
		}
		if !strings.Contains(err.Error(), "must not be empty") {
			t.Fatalf("error = %q, want empty-title message", err.Error())
		}
	})
}

func TestSanitizePageNewFilename(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		title string
		want  string
	}{
		{
			name:  "preserves plain title",
			title: "Architecture Overview",
			want:  "Architecture Overview",
		},
		{
			name:  "replaces invalid punctuation with spaces",
			title: `API: Auth/Login?`,
			want:  "API Auth Login",
		},
		{
			name:  "collapses whitespace and invalid separators",
			title: `Team   Plan / Draft`,
			want:  "Team Plan Draft",
		},
		{
			name:  "removes trailing dots and spaces",
			title: `Release Notes.  `,
			want:  "Release Notes",
		},
		{
			name:  "replaces control characters",
			title: "Guide\tTitle\nv2",
			want:  "Guide Title v2",
		},
		{
			name:  "becomes empty when only invalid characters remain",
			title: `???///...   `,
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := sanitizePageNewFilename(tt.title); got != tt.want {
				t.Fatalf("sanitizePageNewFilename() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsWindowsReservedPageName(t *testing.T) {
	t.Parallel()

	if !isWindowsReservedPageName("CON") {
		t.Fatal("isWindowsReservedPageName(CON) = false, want true")
	}
	if !isWindowsReservedPageName("aux") {
		t.Fatal("isWindowsReservedPageName(aux) = false, want true")
	}
	if isWindowsReservedPageName("guide") {
		t.Fatal("isWindowsReservedPageName(guide) = true, want false")
	}
}

func TestDefaultPageNewFilename(t *testing.T) {
	t.Parallel()

	t.Run("adds markdown extension to sanitized title", func(t *testing.T) {
		t.Parallel()

		got, err := defaultPageNewFilename("API: Auth/Login?")
		if err != nil {
			t.Fatalf("defaultPageNewFilename() error = %v", err)
		}
		if got != "API Auth Login.md" {
			t.Fatalf("defaultPageNewFilename() = %q, want %q", got, "API Auth Login.md")
		}
	})

	t.Run("rejects empty sanitized filename", func(t *testing.T) {
		t.Parallel()

		_, err := defaultPageNewFilename("???///")
		if err == nil {
			t.Fatal("defaultPageNewFilename() error = nil, want non-nil")
		}
		if !strings.Contains(err.Error(), "pass --path") {
			t.Fatalf("error = %q, want pass --path message", err.Error())
		}
	})

	t.Run("rejects windows reserved names case insensitively", func(t *testing.T) {
		t.Parallel()

		_, err := defaultPageNewFilename("Aux")
		if err == nil {
			t.Fatal("defaultPageNewFilename() error = nil, want non-nil")
		}
		if !strings.Contains(err.Error(), "Windows") {
			t.Fatalf("error = %q, want Windows-reserved-name message", err.Error())
		}
	})
}

func TestValidatePageNewPath(t *testing.T) {
	t.Parallel()

	t.Run("rejects empty path", func(t *testing.T) {
		t.Parallel()

		assertValidatePageNewPathError(t, t.TempDir(), "", "must not be empty")
	})

	t.Run("rejects whitespace only path", func(t *testing.T) {
		t.Parallel()

		assertValidatePageNewPathError(t, t.TempDir(), "   ", "must not be empty")
	})

	t.Run("rejects non markdown extension", func(t *testing.T) {
		t.Parallel()

		assertValidatePageNewPathError(t, t.TempDir(), "guide.txt", "ending in .md")
	})

	t.Run("rejects missing filename before extension", func(t *testing.T) {
		t.Parallel()

		assertValidatePageNewPathError(t, t.TempDir(), ".md", "include a filename")
	})

	t.Run("rejects existing target file", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		target := filepath.Join(dir, "guide.md")
		if err := os.WriteFile(target, []byte("existing"), 0o644); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		assertValidatePageNewPathError(t, dir, "guide.md", "already exists")
	})

	t.Run("rejects existing target directory", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		if err := os.Mkdir(filepath.Join(dir, "guide.md"), 0o755); err != nil {
			t.Fatalf("Mkdir() error = %v", err)
		}

		assertValidatePageNewPathError(t, dir, "guide.md", "is a directory")
	})

	t.Run("rejects missing parent directory", func(t *testing.T) {
		t.Parallel()

		assertValidatePageNewPathError(t, t.TempDir(), filepath.Join("docs", "guide.md"), "parent directory")
	})

	t.Run("rejects parent path that is not a directory", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "docs"), []byte("not a directory"), 0o644); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		assertValidatePageNewPathError(t, dir, filepath.Join("docs", "guide.md"), "not a directory")
	})

	t.Run("accepts relative markdown path when parent exists", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dir, "docs"), 0o755); err != nil {
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

	t.Run("accepts absolute markdown path when parent exists", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		target := filepath.Join(dir, "guide.md")

		path, display, err := validatePageNewPath(dir, target)
		if err != nil {
			t.Fatalf("validatePageNewPath() error = %v", err)
		}
		if path != target {
			t.Fatalf("path = %q, want %q", path, target)
		}
		if display != target {
			t.Fatalf("display = %q, want %q", display, target)
		}
	})
}

func TestResolvePageNewTarget(t *testing.T) {
	t.Parallel()

	t.Run("uses explicit path when provided", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dir, "docs"), 0o755); err != nil {
			t.Fatalf("MkdirAll() error = %v", err)
		}

		path, display, err := resolvePageNewTarget(dir, "Guide", filepath.Join("docs", "guide.md"))
		if err != nil {
			t.Fatalf("resolvePageNewTarget() error = %v", err)
		}
		if path != filepath.Join(dir, "docs", "guide.md") {
			t.Fatalf("path = %q, want %q", path, filepath.Join(dir, "docs", "guide.md"))
		}
		if display != filepath.Join("docs", "guide.md") {
			t.Fatalf("display = %q, want %q", display, filepath.Join("docs", "guide.md"))
		}
	})

	t.Run("treats whitespace path as omitted and derives default filename", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		path, display, err := resolvePageNewTarget(dir, "Guide", "   ")
		if err != nil {
			t.Fatalf("resolvePageNewTarget() error = %v", err)
		}
		if path != filepath.Join(dir, "Guide.md") {
			t.Fatalf("path = %q, want %q", path, filepath.Join(dir, "Guide.md"))
		}
		if display != "Guide.md" {
			t.Fatalf("display = %q, want %q", display, "Guide.md")
		}
	})

	t.Run("returns default filename derivation error", func(t *testing.T) {
		t.Parallel()

		_, _, err := resolvePageNewTarget(t.TempDir(), "CON", "")
		if err == nil {
			t.Fatal("resolvePageNewTarget() error = nil, want non-nil")
		}
		if !strings.Contains(err.Error(), "pass --path") {
			t.Fatalf("error = %q, want pass --path message", err.Error())
		}
	})
}

func assertValidatePageNewPathError(t *testing.T, workdir string, pathArg string, wantSubstring string) {
	t.Helper()

	path, display, err := validatePageNewPath(workdir, pathArg)
	if err == nil {
		t.Fatal("validatePageNewPath() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), wantSubstring) {
		t.Fatalf("error = %q, want substring %q", err.Error(), wantSubstring)
	}
	if path != "" || display != "" {
		t.Fatalf("validatePageNewPath() = (%q, %q), want empty values on error", path, display)
	}
}
