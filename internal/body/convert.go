package body

import (
	"bytes"
	"fmt"
	stdhtml "html"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	gmhtml "github.com/yuin/goldmark/renderer/html"
)

const (
	// FormatMarkdown represents markdown input.
	FormatMarkdown = "markdown"
	// FormatStorage represents Confluence storage format input.
	FormatStorage = "storage"
)

var markdownConverter = goldmark.New(
	goldmark.WithExtensions(extension.GFM),
	goldmark.WithRendererOptions(gmhtml.WithUnsafe()),
)

var (
	codeBlockPattern = regexp.MustCompile(`(?s)<pre><code(?: class="language-([^"]+)")?>(.*?)</code></pre>`)
	listItemPattern  = regexp.MustCompile(`^([ \t]*)(?:[-*+]|\d+\.)\s+.+$`)
	hrTagPattern     = regexp.MustCompile(`(?i)<hr\s*/?>`)
	anchorTagPattern = regexp.MustCompile(`(?s)<a\s+href="([^"]+)"(?:\s+[^>]*)?>(.*?)</a>`)
	imageTagPattern  = regexp.MustCompile(`(?s)<img\s+[^>]*>`)
	srcAttrPattern   = regexp.MustCompile(`\ssrc="([^"]*)"`)
	altAttrPattern   = regexp.MustCompile(`\salt="([^"]*)"`)
	htmlTagPattern   = regexp.MustCompile(`(?s)<[^>]+>`)
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
		markdown := preprocessMarkdown(string(content))
		html, err := markdownToHTML(markdown)
		if err != nil {
			return "", err
		}

		storage := htmlToConfluenceStorage(html)
		return applyEditModeCompatibility(storage), nil
	default:
		return "", fmt.Errorf("unsupported body format: %s", normalized)
	}
}

func preprocessMarkdown(markdown string) string {
	return normalizeListSpacing(markdown)
}

func markdownToHTML(markdown string) (string, error) {
	var out bytes.Buffer
	if err := markdownConverter.Convert([]byte(markdown), &out); err != nil {
		return "", fmt.Errorf("convert markdown to storage: %w", err)
	}
	return out.String(), nil
}

func htmlToConfluenceStorage(value string) string {
	storage := convertAnchorTags(value)
	storage = convertImageTags(storage)
	storage = codeBlockPattern.ReplaceAllStringFunc(storage, func(match string) string {
		submatches := codeBlockPattern.FindStringSubmatch(match)
		if len(submatches) != 3 {
			return match
		}

		language := strings.TrimSpace(submatches[1])
		if language == "" {
			language = "text"
		}

		code := stdhtml.UnescapeString(submatches[2])
		code = trimTrailingCodeFenceNewline(code)
		code = strings.ReplaceAll(code, "]]>", "]]]]><![CDATA[>")

		return fmt.Sprintf(
			`<ac:structured-macro ac:name="code"><ac:parameter ac:name="language">%s</ac:parameter><ac:plain-text-body><![CDATA[%s]]></ac:plain-text-body></ac:structured-macro>`,
			language,
			code,
		)
	})
	storage = hrTagPattern.ReplaceAllString(storage, "<hr />")
	return storage
}

func convertAnchorTags(value string) string {
	return anchorTagPattern.ReplaceAllStringFunc(value, func(match string) string {
		submatches := anchorTagPattern.FindStringSubmatch(match)
		if len(submatches) != 3 {
			return match
		}

		url := stdhtml.UnescapeString(submatches[1])
		text := strings.TrimSpace(htmlTagPattern.ReplaceAllString(submatches[2], ""))
		if text == "" {
			text = url
		}
		text = stdhtml.UnescapeString(text)
		text = strings.ReplaceAll(text, "]]>", "]]]]><![CDATA[>")

		return `<ac:link><ri:url ri:value="` +
			stdhtml.EscapeString(url) +
			`" /><ac:plain-text-link-body><![CDATA[` +
			text +
			`]]></ac:plain-text-link-body></ac:link>`
	})
}

func convertImageTags(value string) string {
	return imageTagPattern.ReplaceAllStringFunc(value, func(match string) string {
		srcMatches := srcAttrPattern.FindStringSubmatch(match)
		if len(srcMatches) != 2 {
			return match
		}

		src := stdhtml.UnescapeString(srcMatches[1])
		if strings.TrimSpace(src) == "" {
			return match
		}

		alt := ""
		if altMatches := altAttrPattern.FindStringSubmatch(match); len(altMatches) == 2 {
			alt = stdhtml.UnescapeString(altMatches[1])
		}

		if strings.TrimSpace(alt) == "" {
			return `<ac:image><ri:url ri:value="` +
				stdhtml.EscapeString(src) +
				`" /></ac:image>`
		}

		return `<ac:image ac:alt="` +
			stdhtml.EscapeString(alt) +
			`"><ri:url ri:value="` +
			stdhtml.EscapeString(src) +
			`" /></ac:image>`
	})
}

func applyEditModeCompatibility(storage string) string {
	// Confluence edit mode may treat line breaks between list item text and nested lists
	// as soft line breaks. Collapse only list-adjacent newlines.
	storage = strings.ReplaceAll(storage, "\n<ul>", "<ul>")
	storage = strings.ReplaceAll(storage, "\n<ol>", "<ol>")
	storage = strings.ReplaceAll(storage, "</ul>\n</li>", "</ul></li>")
	storage = strings.ReplaceAll(storage, "</ol>\n</li>", "</ol></li>")
	return storage
}

func trimTrailingCodeFenceNewline(code string) string {
	if strings.HasSuffix(code, "\r\n") {
		return strings.TrimSuffix(code, "\r\n")
	}
	if strings.HasSuffix(code, "\n") {
		return strings.TrimSuffix(code, "\n")
	}
	return code
}

func normalizeListSpacing(markdown string) string {
	if !strings.Contains(markdown, "\n\n") {
		return markdown
	}

	lines := strings.Split(markdown, "\n")
	normalized := make([]string, 0, len(lines))
	for i, line := range lines {
		if strings.TrimSpace(line) != "" || i == 0 || i+1 >= len(lines) {
			normalized = append(normalized, line)
			continue
		}

		_, prevIsList := listItemIndent(lines[i-1])
		_, nextIsList := listItemIndent(lines[i+1])
		if prevIsList && nextIsList {
			continue
		}

		normalized = append(normalized, line)
	}

	return strings.Join(normalized, "\n")
}

func listItemIndent(line string) (int, bool) {
	match := listItemPattern.FindStringSubmatch(line)
	if len(match) != 2 {
		return 0, false
	}

	indent := 0
	for _, r := range match[1] {
		if r == '\t' {
			indent += 4
			continue
		}
		indent++
	}
	return indent, true
}
