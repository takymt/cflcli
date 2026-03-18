package page

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

var numericIDPattern = regexp.MustCompile(`^[0-9]+$`)

// Frontmatter is the required metadata stored in local markdown files.
type Frontmatter struct {
	Title    string
	SpaceKey string
	PageID   string
	ParentID string
	Private  bool
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
		"title":     true,
		"space-key": true,
		"page-id":   true,
		"parent-id": true,
		"private":   true,
	}
	for key := range fields {
		if !allowed[key] {
			return Frontmatter{}, "", fmt.Errorf("frontmatter contains unsupported key %q", key)
		}
	}

	fm := Frontmatter{
		Title:    valueAsString(fields["title"]),
		SpaceKey: valueAsString(fields["space-key"]),
		PageID:   valueAsString(fields["page-id"]),
		ParentID: valueAsString(fields["parent-id"]),
	}
	private, err := valueAsBool(fields["private"])
	if err != nil {
		return Frontmatter{}, "", fmt.Errorf("frontmatter private must be a boolean")
	}
	fm.Private = private
	if fm.Title == "" || fm.SpaceKey == "" || fm.PageID == "" || fm.ParentID == "" {
		return Frontmatter{}, "", errors.New("frontmatter is missing required keys: title, space-key, page-id, parent-id")
	}
	if strings.ContainsAny(fm.SpaceKey, " \t\r\n") {
		return Frontmatter{}, "", errors.New("frontmatter space-key must not contain whitespace")
	}
	for _, entry := range []struct{ label, value string }{
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
	return []byte(fmt.Sprintf("---\ntitle: %s\nspace-key: %s\npage-id: %s\nparent-id: %s\nprivate: %t\n---\n%s",
		frontmatter.Title, frontmatter.SpaceKey, frontmatter.PageID, frontmatter.ParentID, frontmatter.Private, body))
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

func valueAsBool(v any) (bool, error) {
	if v == nil {
		return false, nil
	}

	value, ok := v.(bool)
	if !ok {
		return false, errors.New("not a boolean")
	}
	return value, nil
}
