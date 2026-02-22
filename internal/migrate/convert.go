package migrate

import (
	"encoding/base64"
	"fmt"
	stdhtml "html"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	structuredMacroPattern   = regexp.MustCompile(`(?s)<ac:structured-macro\b[^>]*ac:name="([^"]+)"[^>]*>.*?</ac:structured-macro>`)
	adfPanelExtensionPattern = regexp.MustCompile(`(?s)<ac:adf-extension\b[^>]*>.*?<ac:adf-node\b[^>]*type="panel"[^>]*>.*?<ac:adf-attribute\b[^>]*key="panel-type"[^>]*>\s*([^<]+?)\s*</ac:adf-attribute>.*?<ac:adf-content\b[^>]*>(.*?)</ac:adf-content>.*?</ac:adf-node>.*?</ac:adf-extension>`)
	macroNamePattern         = regexp.MustCompile(`ac:name="([^"]+)"`)
	mermaidLanguagePattern   = regexp.MustCompile(`(?s)<ac:parameter\b[^>]*ac:name="language"[^>]*>\s*mermaid\s*</ac:parameter>`)
	codeLanguagePattern      = regexp.MustCompile(`(?s)<ac:parameter\b[^>]*ac:name="language"[^>]*>\s*([^<]+?)\s*</ac:parameter>`)
	plainTextBodyPattern     = regexp.MustCompile(`(?s)<ac:plain-text-body\b[^>]*>(.*?)</ac:plain-text-body>`)
	richTextBodyPattern      = regexp.MustCompile(`(?s)<ac:rich-text-body\b[^>]*>(.*?)</ac:rich-text-body>`)
	cdataPattern             = regexp.MustCompile(`(?s)<!\[CDATA\[(.*?)\]\]>`)
	imageAttachmentPattern   = regexp.MustCompile(`(?s)<ac:image\b([^>]*)>\s*<ri:attachment\b([^>]*)/>\s*</ac:image>`)
	imageURLPattern          = regexp.MustCompile(`(?s)<ac:image\b([^>]*)>\s*<ri:url\b[^>]*ri:value="([^"]+)"[^>]*/>\s*</ac:image>`)
	attachmentTagPattern     = regexp.MustCompile(`(?s)<ri:attachment\b([^>]*)/?>`)
	filenameAttributePattern = regexp.MustCompile(`ri:filename="([^"]+)"`)
	altAttributePattern      = regexp.MustCompile(`ac:alt="([^"]*)"`)
	xmlHeaderPattern         = regexp.MustCompile(`(?s)<\?xml[^>]*\?>`)
	excessBlankLinesPattern  = regexp.MustCompile(`\n{3,}`)
)

// StorageToMarkdown converts Confluence storage into migrate-friendly markdown.
//
// The conversion focuses on migration requirements:
//   - mermaid macro -> fenced mermaid block
//   - unsupported macros -> HTML comment with macro name + raw storage
//   - ri:attachment -> markdown image link under attachments
func StorageToMarkdown(storage string, attachmentPath func(filename string) string) (string, []string, error) {
	if attachmentPath == nil {
		attachmentPath = func(filename string) string {
			return filename
		}
	}

	normalized := strings.ReplaceAll(storage, "\r\n", "\n")
	normalized = convertADFPanels(normalized)
	normalized = convertStructuredMacros(normalized)

	attachments := newOrderedStringSet()
	normalized = imageAttachmentPattern.ReplaceAllStringFunc(normalized, func(match string) string {
		submatches := imageAttachmentPattern.FindStringSubmatch(match)
		if len(submatches) != 3 {
			return match
		}

		filename := extractAttribute(filenameAttributePattern, submatches[2])
		if filename == "" {
			return match
		}

		alt := extractAttribute(altAttributePattern, submatches[1])
		if alt == "" {
			alt = filename
		}

		attachments.Add(filename)
		targetPath := filepath.ToSlash(attachmentPath(filename))
		return markdownImage(alt, targetPath)
	})
	normalized = imageURLPattern.ReplaceAllStringFunc(normalized, func(match string) string {
		submatches := imageURLPattern.FindStringSubmatch(match)
		if len(submatches) != 3 {
			return match
		}

		targetPath := strings.TrimSpace(submatches[2])
		if targetPath == "" {
			return match
		}
		alt := extractAttribute(altAttributePattern, submatches[1])
		if alt == "" {
			alt = targetPath
		}

		return markdownImage(alt, filepath.ToSlash(targetPath))
	})

	normalized = attachmentTagPattern.ReplaceAllStringFunc(normalized, func(match string) string {
		filename := extractAttribute(filenameAttributePattern, match)
		if filename == "" {
			return match
		}

		attachments.Add(filename)
		targetPath := filepath.ToSlash(attachmentPath(filename))
		return markdownImage(filename, targetPath)
	})

	normalized = xmlHeaderPattern.ReplaceAllString(normalized, "")
	normalized = convertHTMLToMarkdownSyntax(normalized)
	normalized = excessBlankLinesPattern.ReplaceAllString(normalized, "\n\n")
	normalized = strings.TrimSpace(normalized)
	if normalized != "" {
		normalized += "\n"
	}

	return normalized, attachments.Items(), nil
}

func convertStructuredMacros(storage string) string {
	return structuredMacroPattern.ReplaceAllStringFunc(storage, func(match string) string {
		macroName := strings.ToLower(strings.TrimSpace(extractAttribute(macroNamePattern, match)))
		if directive, ok := convertDirectiveMacro(match, macroName); ok {
			return directive
		}
		if macroName == "code" && mermaidLanguagePattern.MatchString(match) {
			source := strings.Trim(extractPlainTextBody(match), "\n")
			return "\n```mermaid\n" + source + "\n```\n"
		}
		if macroName == "code" {
			source := strings.Trim(extractPlainTextBody(match), "\n")
			language := strings.TrimSpace(extractAttribute(codeLanguagePattern, match))
			if source == "" {
				return ""
			}
			if language == "" {
				return "\n```\n" + source + "\n```\n"
			}
			return "\n```" + language + "\n" + source + "\n```\n"
		}

		if macroName == "" {
			macroName = "unknown"
		}
		encoded := base64.StdEncoding.EncodeToString([]byte(match))
		return fmt.Sprintf(`<!-- cfl:migrate-unsupported-macro name=%q storage-base64=%q -->`, sanitizeMacroName(macroName), encoded)
	})
}

func convertADFPanels(storage string) string {
	return adfPanelExtensionPattern.ReplaceAllStringFunc(storage, func(match string) string {
		submatches := adfPanelExtensionPattern.FindStringSubmatch(match)
		if len(submatches) != 3 {
			return match
		}
		panelType := strings.ToLower(strings.TrimSpace(submatches[1]))
		body := strings.TrimSpace(submatches[2])
		switch panelType {
		case "note":
			return markdownDirective("memo", "", body)
		default:
			return match
		}
	})
}

func convertDirectiveMacro(match, macroName string) (string, bool) {
	body := strings.TrimSpace(extractRichTextBody(match))
	switch macroName {
	case "expand":
		title := stdhtml.UnescapeString(strings.TrimSpace(extractMacroParameter(match, "title")))
		// Deliberately ignore `expanded` and canonicalize to :::details.
		return markdownDirective("details", title, body), true
	case "info":
		return markdownDirective("info", "", body), true
	case "tip":
		return markdownDirective("success", "", body), true
	case "note":
		return markdownDirective("warn", "", body), true
	case "warning":
		return markdownDirective("warn", "", body), true
	default:
		return "", false
	}
}

func extractRichTextBody(storage string) string {
	submatches := richTextBodyPattern.FindStringSubmatch(storage)
	if len(submatches) != 2 {
		return ""
	}
	return submatches[1]
}

func extractMacroParameter(storage, name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	pattern := regexp.MustCompile(`(?s)<ac:parameter\b[^>]*ac:name="` + regexp.QuoteMeta(name) + `"[^>]*>(.*?)</ac:parameter>`)
	submatches := pattern.FindStringSubmatch(storage)
	if len(submatches) != 2 {
		return ""
	}
	return submatches[1]
}

func markdownDirective(kind, title, body string) string {
	kind = strings.TrimSpace(kind)
	if kind == "" {
		return body
	}
	body = strings.Trim(strings.ReplaceAll(body, "\r\n", "\n"), "\n")
	header := ":::" + kind
	title = strings.TrimSpace(title)
	if title != "" {
		header += " " + title
	}
	if body == "" {
		return "\n" + header + "\n:::\n"
	}
	return "\n" + header + "\n" + body + "\n:::\n"
}

func extractPlainTextBody(storage string) string {
	submatches := plainTextBodyPattern.FindStringSubmatch(storage)
	if len(submatches) != 2 {
		return ""
	}

	body := submatches[1]
	cdata := cdataPattern.FindStringSubmatch(body)
	if len(cdata) == 2 {
		return cdata[1]
	}
	return strings.TrimSpace(body)
}

func sanitizeMacroName(name string) string {
	name = strings.ReplaceAll(name, "\"", "'")
	return strings.TrimSpace(name)
}

func extractAttribute(pattern *regexp.Regexp, value string) string {
	submatches := pattern.FindStringSubmatch(value)
	if len(submatches) != 2 {
		return ""
	}
	return strings.TrimSpace(submatches[1])
}

func markdownImage(alt, targetPath string) string {
	alt = strings.ReplaceAll(alt, "]", `\]`)
	return "![" + alt + "](" + targetPath + ")"
}

type orderedStringSet struct {
	order []string
	seen  map[string]struct{}
}

func newOrderedStringSet() *orderedStringSet {
	return &orderedStringSet{
		order: []string{},
		seen:  map[string]struct{}{},
	}
}

func (s *orderedStringSet) Add(value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	if _, ok := s.seen[value]; ok {
		return
	}
	s.seen[value] = struct{}{}
	s.order = append(s.order, value)
}

func (s *orderedStringSet) Items() []string {
	items := make([]string, len(s.order))
	copy(items, s.order)
	return items
}
