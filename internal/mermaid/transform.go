package mermaid

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type rendererFactory func() (SVGRenderer, error)

// RenderMarkdownFences renders Mermaid fenced code blocks and replaces them with
// markdown image references that point to generated local SVG files.
func RenderMarkdownFences(
	ctx context.Context,
	markdown []byte,
	outputDir string,
	factory rendererFactory,
) ([]byte, func() error, error) {
	resolvedOutputDir := strings.TrimSpace(outputDir)
	if resolvedOutputDir == "" {
		resolvedOutputDir = "."
	}

	lines := strings.Split(strings.ReplaceAll(string(markdown), "\r\n", "\n"), "\n")
	output := make([]string, 0, len(lines))

	var (
		renderer       SVGRenderer
		createdTempDir string
		renderedCount  int
	)

	cleanup := func() error {
		if strings.TrimSpace(createdTempDir) == "" {
			return nil
		}
		return os.RemoveAll(createdTempDir)
	}

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		fence, ok := parseFenceStart(line)
		if !ok || !isMermaidInfoString(fence.infoString) {
			output = append(output, line)
			continue
		}

		end := -1
		for j := i + 1; j < len(lines); j++ {
			if isFenceEnd(lines[j], fence.fenceChar, fence.fenceLength) {
				end = j
				break
			}
		}
		if end == -1 {
			output = append(output, line)
			continue
		}

		if factory == nil {
			_ = cleanup()
			return nil, nil, fmt.Errorf("render mermaid: renderer factory is nil")
		}
		if renderer == nil {
			var err error
			renderer, err = factory()
			if err != nil {
				_ = cleanup()
				return nil, nil, fmt.Errorf("render mermaid: %w", err)
			}
		}

		source := strings.TrimSpace(strings.Join(lines[i+1:end], "\n"))
		svg, err := renderer.Render(ctx, source)
		if err != nil {
			_ = renderer.Close()
			_ = cleanup()
			return nil, nil, fmt.Errorf("render mermaid block %d: %w", renderedCount+1, err)
		}

		if strings.TrimSpace(createdTempDir) == "" {
			createdTempDir, err = os.MkdirTemp(resolvedOutputDir, ".cfl-mermaid-*")
			if err != nil {
				_ = renderer.Close()
				_ = cleanup()
				return nil, nil, fmt.Errorf("create temp dir for mermaid render: %w", err)
			}
		}

		renderedCount++
		filename := fmt.Sprintf("cfl-mermaid-%03d.svg", renderedCount)
		path := filepath.Join(createdTempDir, filename)
		if err := os.WriteFile(path, svg, 0o600); err != nil {
			_ = renderer.Close()
			_ = cleanup()
			return nil, nil, fmt.Errorf("write mermaid image %q: %w", filename, err)
		}
		relativePath, err := filepath.Rel(resolvedOutputDir, path)
		if err != nil {
			_ = renderer.Close()
			_ = cleanup()
			return nil, nil, fmt.Errorf("resolve relative mermaid image path %q: %w", path, err)
		}
		output = append(output, fence.indent+fmt.Sprintf("![mermaid-%d](<%s>)", renderedCount, filepath.ToSlash(relativePath)))
		i = end
	}

	if renderer != nil {
		if err := renderer.Close(); err != nil {
			_ = cleanup()
			return nil, nil, fmt.Errorf("close mermaid renderer: %w", err)
		}
	}

	result := []byte(strings.Join(output, "\n"))
	if renderedCount == 0 {
		return result, nil, nil
	}
	return result, cleanup, nil
}

type fenceMarker struct {
	indent      string
	fenceChar   byte
	fenceLength int
	infoString  string
}

func parseFenceStart(line string) (fenceMarker, bool) {
	var marker fenceMarker

	indentSize := 0
	for indentSize < len(line) && line[indentSize] == ' ' && indentSize < 4 {
		indentSize++
	}
	if indentSize > 3 || indentSize >= len(line) {
		return marker, false
	}

	ch := line[indentSize]
	if ch != '`' && ch != '~' {
		return marker, false
	}

	end := indentSize
	for end < len(line) && line[end] == ch {
		end++
	}
	if end-indentSize < 3 {
		return marker, false
	}

	info := strings.TrimSpace(line[end:])
	marker.indent = line[:indentSize]
	marker.fenceChar = ch
	marker.fenceLength = end - indentSize
	marker.infoString = info
	return marker, true
}

func isMermaidInfoString(value string) bool {
	fields := strings.Fields(strings.TrimSpace(value))
	if len(fields) == 0 {
		return false
	}
	return strings.EqualFold(fields[0], "mermaid")
}

func isFenceEnd(line string, fenceChar byte, minLength int) bool {
	trimmed := strings.TrimSpace(line)
	if len(trimmed) < minLength {
		return false
	}
	for i := 0; i < len(trimmed); i++ {
		if trimmed[i] != fenceChar {
			return false
		}
	}
	return true
}
