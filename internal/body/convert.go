package body

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/yuin/goldmark"
)

const (
	// FormatMarkdown represents markdown input.
	FormatMarkdown = "markdown"
	// FormatStorage represents Confluence storage format input.
	FormatStorage = "storage"
)

// NormalizeFormat validates and normalizes body format.
func NormalizeFormat(value string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case FormatMarkdown, FormatStorage:
		return normalized, nil
	default:
		return "", fmt.Errorf("body format must be one of: markdown, storage")
	}
}

// ToStorage converts body content to Confluence storage format.
func ToStorage(content []byte, format string) (string, error) {
	normalized, err := NormalizeFormat(format)
	if err != nil {
		return "", err
	}

	switch normalized {
	case FormatStorage:
		return string(content), nil
	case FormatMarkdown:
		var out bytes.Buffer
		if err := goldmark.Convert(content, &out); err != nil {
			return "", fmt.Errorf("convert markdown to storage: %w", err)
		}
		return out.String(), nil
	default:
		return "", fmt.Errorf("unsupported body format: %s", normalized)
	}
}
