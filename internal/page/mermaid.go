package page

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

const maxMermaidBlockChars = 2000

// ConvertMarkdownToStorageWithMermaid converts markdown to storage format and
// turns mermaid fenced blocks into attachment image macros.
func ConvertMarkdownToStorageWithMermaid(ctx context.Context, markdownPath string, markdown string, siteBaseURL string) (string, []string, error) {
	markdown = resolveRelativeMarkdownLinks(markdownPath, markdown, strings.TrimSuffix(siteBaseURL, "/"))

	lines := strings.Split(markdown, "\n")
	var (
		parts        []string
		pending      []string
		mermaidIndex int
		generated    []string
	)

	flushPending := func() error {
		if len(pending) == 0 {
			return nil
		}
		converted, err := ConvertMarkdownToStorage(strings.Join(pending, "\n"))
		if err != nil {
			return err
		}
		if converted != "" {
			parts = append(parts, converted)
		}
		pending = pending[:0]
		return nil
	}

	for i := 0; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		opts, isMermaidFence := parseMermaidFence(trimmed)
		if isMermaidFence {
			if err := flushPending(); err != nil {
				return "", nil, err
			}
			start := i
			i++

			var mermaidLines []string
			closed := false
			for i < len(lines) {
				if strings.HasPrefix(strings.TrimSpace(lines[i]), "```") {
					closed = true
					break
				}
				mermaidLines = append(mermaidLines, lines[i])
				i++
			}
			if !closed {
				pending = append(pending, lines[start:]...)
				break
			}

			mermaidIndex++
			source := strings.Join(mermaidLines, "\n")
			if len(source) > maxMermaidBlockChars {
				return "", nil, fmt.Errorf("mermaid block %d exceeds %d chars", mermaidIndex, maxMermaidBlockChars)
			}

			filename, generatedPath, err := renderMermaidSVG(ctx, markdownPath, source, mermaidIndex)
			if err != nil {
				return "", nil, err
			}
			generated = append(generated, generatedPath)
			parts = append(parts, buildMermaidImageStorage(filename, opts))
			continue
		}
		pending = append(pending, lines[i])
	}

	if err := flushPending(); err != nil {
		return "", nil, err
	}
	return strings.Join(parts, "\n"), generated, nil
}

func resolveRelativeMarkdownLinks(markdownPath string, markdown string, siteBaseURL string) string {
	lines := strings.Split(markdown, "\n")
	inFence := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		lines[i] = rewriteRelativeMarkdownLinksInLine(markdownPath, line, siteBaseURL)
	}
	return strings.Join(lines, "\n")
}

func rewriteRelativeMarkdownLinksInLine(markdownPath string, line string, siteBaseURL string) string {
	matches := linkRE.FindAllStringSubmatchIndex(line, -1)
	if len(matches) == 0 {
		return line
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

		resolved := resolveRelativeMarkdownTarget(markdownPath, target, siteBaseURL)
		b.WriteString(line[last:start])
		if resolved != "" {
			b.WriteString("[" + label + "](" + resolved + ")")
		} else {
			b.WriteString(line[start:end])
		}
		last = end
	}
	b.WriteString(line[last:])
	return b.String()
}

func resolveRelativeMarkdownTarget(markdownPath string, target string, siteBaseURL string) string {
	if target == "" || strings.HasPrefix(target, "/") || strings.Contains(target, "://") {
		return ""
	}
	u, err := url.Parse(target)
	if err != nil {
		return ""
	}
	if u.Path == "" || !strings.HasSuffix(strings.ToLower(u.Path), ".md") {
		return ""
	}
	absPath := filepath.Clean(filepath.Join(filepath.Dir(markdownPath), u.Path))
	data, err := os.ReadFile(absPath)
	if err != nil {
		return ""
	}
	fm, _, err := ParseMarkdownFile(data)
	if err != nil || fm.SpaceKey == "" || fm.PageID == "" {
		return ""
	}
	return siteBaseURL + "/wiki/spaces/" + fm.SpaceKey + "/pages/" + fm.PageID
}

type mermaidOptions struct {
	align string
	width int
}

func parseMermaidFence(line string) (mermaidOptions, bool) {
	if !strings.HasPrefix(line, "```") {
		return mermaidOptions{}, false
	}
	info := strings.TrimSpace(strings.TrimPrefix(line, "```"))
	if info == "" {
		return mermaidOptions{}, false
	}
	parts := strings.Fields(info)
	if len(parts) == 0 || parts[0] != "mermaid" {
		return mermaidOptions{}, false
	}

	opts := mermaidOptions{align: "center"}
	for _, token := range parts[1:] {
		kv := strings.SplitN(token, "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(kv[0]))
		value := strings.ToLower(strings.TrimSpace(kv[1]))
		switch key {
		case "align":
			if value == "left" || value == "center" || value == "right" {
				opts.align = value
			}
		case "width":
			w, err := strconv.Atoi(value)
			if err == nil && w > 0 {
				opts.width = w
			}
		}
	}
	return opts, true
}

func buildMermaidImageStorage(filename string, opts mermaidOptions) string {
	align := opts.align
	if align == "" {
		align = "center"
	}
	layout := "center"
	switch align {
	case "left":
		layout = "align-start"
	case "right":
		layout = "align-end"
	}

	var attrs []string
	attrs = append(attrs, `ac:align="`+align+`"`, `ac:layout="`+layout+`"`)
	if opts.width > 0 {
		attrs = append(attrs, `ac:width="`+strconv.Itoa(opts.width)+`"`)
	}
	return `<ac:image ` + strings.Join(attrs, " ") + `><ri:attachment ri:filename="` + filename + `" /></ac:image>`
}

func renderMermaidSVG(ctx context.Context, markdownPath string, source string, index int) (string, string, error) {
	filename := "mermaid-" + strconv.Itoa(index) + ".svg"
	svgPath := filepath.Join(filepath.Dir(markdownPath), filename)

	tmpFile, err := os.CreateTemp("", "cfl-mermaid-*.mmd")
	if err != nil {
		return "", "", err
	}
	tmpPath := tmpFile.Name()
	if _, err := tmpFile.WriteString(source); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
		return "", "", err
	}
	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return "", "", err
	}
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	output, err := exec.CommandContext(ctx, "mmdc", "-i", tmpPath, "-o", svgPath).CombinedOutput()
	if err != nil {
		return "", "", fmt.Errorf("mmdc failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return filename, svgPath, nil
}
