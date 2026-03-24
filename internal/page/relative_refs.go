package page

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type relativeReferenceResult struct {
	Markdown          string
	AttachmentSources map[string]string
	Warnings          []string
}

type resolvedReference struct {
	attachmentPath string
	pageURL        string
	pageTitle      string
	warning        string
}

func resolveRelativeReferences(markdownPath string, markdown string, siteBaseURL string) (relativeReferenceResult, error) {
	result := relativeReferenceResult{
		Markdown:          markdown,
		AttachmentSources: make(map[string]string),
	}

	lines := strings.Split(markdown, "\n")
	var currentFence string
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if currentFence != "" {
			if isFenceClose(trimmed, currentFence) {
				currentFence = ""
			}
			continue
		}
		fence, ok := parseFenceStart(trimmed)
		if ok {
			currentFence = fence.marker
			continue
		}

		var err error
		line, result.Warnings, err = rewriteRelativeImagesInLine(markdownPath, line, siteBaseURL, result.AttachmentSources, result.Warnings)
		if err != nil {
			return relativeReferenceResult{}, err
		}
		line, result.Warnings, err = rewriteRelativeLinksInLine(markdownPath, line, siteBaseURL, result.AttachmentSources, result.Warnings)
		if err != nil {
			return relativeReferenceResult{}, err
		}
		lines[i] = line
	}

	result.Markdown = strings.Join(lines, "\n")
	return result, nil
}

func rewriteRelativeImagesInLine(markdownPath string, line string, siteBaseURL string, attachments map[string]string, warnings []string) (string, []string, error) {
	matches := imageRE.FindAllStringSubmatchIndex(line, -1)
	if len(matches) == 0 {
		return line, warnings, nil
	}

	var b strings.Builder
	last := 0
	for _, idx := range matches {
		if len(idx) < 6 {
			continue
		}
		start, end := idx[0], idx[1]
		alt := strings.TrimSpace(line[idx[2]:idx[3]])
		target := strings.TrimSpace(line[idx[4]:idx[5]])

		resolved := resolveRelativeReference(markdownPath, target, siteBaseURL)
		b.WriteString(line[last:start])
		switch {
		case resolved.warning != "":
			warnings = append(warnings, resolved.warning)
			b.WriteString(line[start:end])
		case resolved.pageURL != "":
			label := alt
			if label == "" {
				label = resolved.pageTitle
			}
			if label == "" {
				label = attachmentFilenameFromTarget(target)
			}
			b.WriteString("[" + label + "](" + resolved.pageURL + ")")
		case resolved.attachmentPath != "":
			filename := attachmentFilenameFromTarget(target)
			if err := recordAttachmentSource(attachments, filename, resolved.attachmentPath); err != nil {
				return "", warnings, err
			}
			b.WriteString(line[start:end])
		default:
			b.WriteString(line[start:end])
		}
		last = end
	}
	b.WriteString(line[last:])
	return b.String(), warnings, nil
}

func rewriteRelativeLinksInLine(markdownPath string, line string, siteBaseURL string, attachments map[string]string, warnings []string) (string, []string, error) {
	matches := linkRE.FindAllStringSubmatchIndex(line, -1)
	if len(matches) == 0 {
		return line, warnings, nil
	}

	var b strings.Builder
	last := 0
	for _, idx := range matches {
		if len(idx) < 6 {
			continue
		}
		start, end := idx[0], idx[1]
		if start > 0 && line[start-1] == '!' {
			continue
		}

		label := strings.TrimSpace(line[idx[2]:idx[3]])
		target := strings.TrimSpace(line[idx[4]:idx[5]])

		resolved := resolveRelativeReference(markdownPath, target, siteBaseURL)
		b.WriteString(line[last:start])
		switch {
		case resolved.warning != "":
			warnings = append(warnings, resolved.warning)
			b.WriteString(line[start:end])
		case resolved.pageURL != "":
			b.WriteString("[" + label + "](" + resolved.pageURL + ")")
		case resolved.attachmentPath != "":
			filename := attachmentFilenameFromTarget(target)
			if err := recordAttachmentSource(attachments, filename, resolved.attachmentPath); err != nil {
				return "", warnings, err
			}
			b.WriteString(line[start:end])
		default:
			b.WriteString(line[start:end])
		}
		last = end
	}
	b.WriteString(line[last:])
	return b.String(), warnings, nil
}

func resolveRelativeReference(markdownPath string, target string, siteBaseURL string) resolvedReference {
	localPath, absolute, ok := localPathFromTarget(target)
	if !ok {
		return resolvedReference{}
	}
	if absolute {
		return resolvedReference{
			warning: fmt.Sprintf("skipped absolute local path %q", target),
		}
	}

	absPath := filepath.Clean(filepath.Join(filepath.Dir(markdownPath), localPath))
	if !strings.EqualFold(filepath.Ext(localPath), ".md") {
		return resolvedReference{attachmentPath: absPath}
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return resolvedReference{attachmentPath: absPath}
	}

	fm, _, err := ParseMarkdownFile(data)
	if err != nil || fm.SpaceKey == "" || fm.PageID == "" {
		return resolvedReference{attachmentPath: absPath}
	}

	pageURL := buildPageURL(siteBaseURL, fm.SpaceKey, fm.PageID)
	u, err := url.Parse(target)
	if err == nil {
		if u.RawQuery != "" {
			pageURL += "?" + u.RawQuery
		}
		if u.Fragment != "" {
			pageURL += "#" + u.Fragment
		}
	}

	return resolvedReference{
		pageURL:   pageURL,
		pageTitle: fm.Title,
	}
}

func buildPageURL(siteBaseURL string, spaceKey string, pageID string) string {
	return strings.TrimSuffix(siteBaseURL, "/") + "/wiki/spaces/" + spaceKey + "/pages/" + pageID
}

func recordAttachmentSource(attachments map[string]string, filename string, sourcePath string) error {
	if attachments == nil || filename == "" || sourcePath == "" {
		return nil
	}

	sourcePath = filepath.Clean(sourcePath)
	if existing, ok := attachments[filename]; ok {
		existing = filepath.Clean(existing)
		if existing != sourcePath {
			return fmt.Errorf("attachment filename collision: %q resolves to multiple files:\n- %s\n- %s\nrename one of the files to keep attachment filenames unique", filename, existing, sourcePath)
		}
		return nil
	}

	attachments[filename] = sourcePath
	return nil
}
