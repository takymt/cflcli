package migrate

import (
	"encoding/base64"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	structuredMacroPattern   = regexp.MustCompile(`(?s)<ac:structured-macro\b[^>]*ac:name="([^"]+)"[^>]*>.*?</ac:structured-macro>`)
	macroNamePattern         = regexp.MustCompile(`ac:name="([^"]+)"`)
	mermaidLanguagePattern   = regexp.MustCompile(`(?s)<ac:parameter\b[^>]*ac:name="language"[^>]*>\s*mermaid\s*</ac:parameter>`)
	plainTextBodyPattern     = regexp.MustCompile(`(?s)<ac:plain-text-body\b[^>]*>(.*?)</ac:plain-text-body>`)
	cdataPattern             = regexp.MustCompile(`(?s)<!\[CDATA\[(.*?)\]\]>`)
	imageAttachmentPattern   = regexp.MustCompile(`(?s)<ac:image\b([^>]*)>\s*<ri:attachment\b([^>]*)/>\s*</ac:image>`)
	imageURLPattern          = regexp.MustCompile(`(?s)<ac:image\b([^>]*)>\s*<ri:url\b[^>]*ri:value="([^"]+)"[^>]*/>\s*</ac:image>`)
	attachmentTagPattern     = regexp.MustCompile(`(?s)<ri:attachment\b([^>]*)/?>`)
	filenameAttributePattern = regexp.MustCompile(`ri:filename="([^"]+)"`)
	altAttributePattern      = regexp.MustCompile(`ac:alt="([^"]*)"`)
	xmlHeaderPattern         = regexp.MustCompile(`(?s)<\?xml[^>]*\?>`)
	pOpenPattern             = regexp.MustCompile(`(?i)<p\b[^>]*>`)
	pClosePattern            = regexp.MustCompile(`(?i)</p>`)
	brPattern                = regexp.MustCompile(`(?i)<br\s*/?>`)
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
	normalized = pOpenPattern.ReplaceAllString(normalized, "")
	normalized = pClosePattern.ReplaceAllString(normalized, "\n\n")
	normalized = brPattern.ReplaceAllString(normalized, "\n")
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
		if macroName == "code" && mermaidLanguagePattern.MatchString(match) {
			source := strings.Trim(extractPlainTextBody(match), "\n")
			return "\n```mermaid\n" + source + "\n```\n"
		}

		if macroName == "" {
			macroName = "unknown"
		}
		encoded := base64.StdEncoding.EncodeToString([]byte(match))
		return fmt.Sprintf(`<!-- cfl:migrate-unsupported-macro name=%q storage-base64=%q -->`, sanitizeMacroName(macroName), encoded)
	})
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
	return "![" + alt + "](<" + targetPath + ">)"
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
