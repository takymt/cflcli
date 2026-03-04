package page

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var numericIDPattern = regexp.MustCompile(`^[0-9]+$`)

type Document struct {
	Path     string
	Title    string
	SpaceID  string
	PageID   string
	ParentID string
	Body     string
}

func ParseDocument(path string) (Document, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Document{}, err
	}

	content := string(raw)
	if !strings.HasPrefix(content, "---\n") {
		return Document{}, ErrMissingFrontmatter
	}

	lines := strings.Split(content, "\n")
	if len(lines) < 2 || lines[0] != "---" {
		return Document{}, ErrMissingFrontmatter
	}

	end := -1
	for i := 1; i < len(lines); i++ {
		if lines[i] == "---" {
			end = i
			break
		}
	}
	if end < 0 {
		return Document{}, fmt.Errorf("%w: missing closing delimiter", ErrInvalidFrontmatter)
	}

	frontmatter := strings.Join(lines[1:end], "\n")
	body := strings.Join(lines[end+1:], "\n")

	fields, err := parseFrontmatter(frontmatter)
	if err != nil {
		return Document{}, err
	}

	doc := Document{
		Path:     path,
		Title:    TitleFromPath(path),
		SpaceID:  fields["space-id"],
		PageID:   fields["page-id"],
		ParentID: fields["parent-id"],
		Body:     body,
	}

	if err := validateRequiredIDs(doc.SpaceID, doc.PageID, doc.ParentID); err != nil {
		return Document{}, err
	}

	return doc, nil
}

func WriteNewDocument(path, spaceID, pageID, parentID string) error {
	if err := validateRequiredIDs(spaceID, pageID, parentID); err != nil {
		return err
	}

	content := fmt.Sprintf(
		"---\nspace-id: %s\npage-id: %s\nparent-id: %s\n---\n",
		spaceID,
		pageID,
		parentID,
	)

	return os.WriteFile(path, []byte(content), 0o644)
}

func TitleFromPath(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	return strings.TrimSuffix(base, ext)
}

func parseFrontmatter(frontmatter string) (map[string]string, error) {
	fields := map[string]string{}
	lines := strings.Split(frontmatter, "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("%w: malformed line %q", ErrInvalidFrontmatter, line)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if key == "" || value == "" {
			return nil, fmt.Errorf("%w: malformed line %q", ErrInvalidFrontmatter, line)
		}

		fields[key] = value
	}

	return fields, nil
}

func validateRequiredIDs(spaceID, pageID, parentID string) error {
	if !numericIDPattern.MatchString(spaceID) {
		return fmt.Errorf("%w: invalid space-id %q", ErrInvalidFrontmatter, spaceID)
	}
	if !numericIDPattern.MatchString(pageID) {
		return fmt.Errorf("%w: invalid page-id %q", ErrInvalidFrontmatter, pageID)
	}
	if !numericIDPattern.MatchString(parentID) {
		return fmt.Errorf("%w: invalid parent-id %q", ErrInvalidFrontmatter, parentID)
	}
	return nil
}
