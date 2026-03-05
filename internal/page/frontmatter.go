package page

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

var numericIDPattern = regexp.MustCompile(`^[0-9]+$`)

// Frontmatter is the required metadata stored in local markdown files.
type Frontmatter struct {
	SpaceID  string
	PageID   string
	ParentID string
}

// ParseMarkdownFile parses and validates the required YAML frontmatter.
func ParseMarkdownFile(data []byte) (Frontmatter, string, error) {
	text := string(data)
	if !strings.HasPrefix(text, "---\n") {
		return Frontmatter{}, "", errors.New("frontmatter is required")
	}

	rest := strings.TrimPrefix(text, "---\n")
	idx := strings.Index(rest, "\n---\n")
	if idx < 0 {
		return Frontmatter{}, "", errors.New("frontmatter is malformed: missing closing delimiter")
	}

	rawFrontmatter := rest[:idx]
	body := rest[idx+len("\n---\n"):]

	var fields map[string]any
	if err := yaml.Unmarshal([]byte(rawFrontmatter), &fields); err != nil {
		return Frontmatter{}, "", fmt.Errorf("frontmatter is malformed: %w", err)
	}

	allowed := map[string]bool{
		"space-id":  true,
		"page-id":   true,
		"parent-id": true,
	}
	for key := range fields {
		if !allowed[key] {
			return Frontmatter{}, "", fmt.Errorf("frontmatter contains unsupported key %q", key)
		}
	}

	fm := Frontmatter{
		SpaceID:  valueAsString(fields["space-id"]),
		PageID:   valueAsString(fields["page-id"]),
		ParentID: valueAsString(fields["parent-id"]),
	}
	if fm.SpaceID == "" || fm.PageID == "" || fm.ParentID == "" {
		return Frontmatter{}, "", errors.New("frontmatter is missing required keys: space-id, page-id, parent-id")
	}
	for _, entry := range []struct{ label, value string }{
		{"space-id", fm.SpaceID},
		{"page-id", fm.PageID},
		{"parent-id", fm.ParentID},
	} {
		if !numericIDPattern.MatchString(entry.value) {
			return Frontmatter{}, "", fmt.Errorf("frontmatter %s must be a numeric Confluence id", entry.label)
		}
	}

	return fm, body, nil
}

// FormatMarkdownFile renders a markdown file with the required frontmatter.
func FormatMarkdownFile(frontmatter Frontmatter, body string) []byte {
	return []byte(fmt.Sprintf("---\nspace-id: %s\npage-id: %s\nparent-id: %s\n---\n%s",
		frontmatter.SpaceID, frontmatter.PageID, frontmatter.ParentID, body))
}

// TitleFromPath derives the Confluence page title from the file basename.
func TitleFromPath(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

func valueAsString(v any) string {
	switch value := v.(type) {
	case string:
		return value
	case int, int64, uint64:
		return fmt.Sprintf("%d", value)
	case float64:
		if value == float64(int64(value)) {
			return fmt.Sprintf("%.0f", value)
		}
		return ""
	default:
		return ""
	}
}
