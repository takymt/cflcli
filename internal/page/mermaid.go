package page

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

const maxMermaidBlockChars = 2000

// ConvertMarkdownToStorageWithMermaid converts markdown to storage format and
// turns mermaid fenced blocks into attachment image macros.
func ConvertMarkdownToStorageWithMermaid(ctx context.Context, markdownPath string, markdown string) (string, error) {
	lines := strings.Split(markdown, "\n")
	var (
		parts        []string
		pending      []string
		mermaidIndex int
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
		if strings.HasPrefix(trimmed, "```") && strings.TrimSpace(strings.TrimPrefix(trimmed, "```")) == "mermaid" {
			if err := flushPending(); err != nil {
				return "", err
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
				return "", fmt.Errorf("mermaid block %d exceeds %d chars", mermaidIndex, maxMermaidBlockChars)
			}

			filename, err := renderMermaidSVG(ctx, markdownPath, source, mermaidIndex)
			if err != nil {
				return "", err
			}
			parts = append(parts, `<ac:image ac:align="center" ac:layout="center"><ri:attachment ri:filename="`+filename+`" /></ac:image>`)
			continue
		}
		pending = append(pending, lines[i])
	}

	if err := flushPending(); err != nil {
		return "", err
	}
	return strings.Join(parts, "\n"), nil
}

func renderMermaidSVG(ctx context.Context, markdownPath string, source string, index int) (string, error) {
	filename := "mermaid-" + strconv.Itoa(index) + ".svg"
	svgPath := filepath.Join(filepath.Dir(markdownPath), filename)

	tmpFile, err := os.CreateTemp("", "cfl-mermaid-*.mmd")
	if err != nil {
		return "", err
	}
	tmpPath := tmpFile.Name()
	if _, err := tmpFile.WriteString(source); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
		return "", err
	}
	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return "", err
	}
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	output, err := exec.CommandContext(ctx, "mmdc", "-i", tmpPath, "-o", svgPath).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("mmdc failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return filename, nil
}
