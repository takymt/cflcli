package page

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

const maxMermaidBlockChars = 2000

var mermaidWarmupOnce sync.Once

// MermaidResult contains mermaid rendering output and generated attachment files.
type MermaidResult struct {
	Storage   string
	Generated map[string]string
	SaveCache func() error
}

// ConvertMarkdownToStorageWithMermaid converts markdown to storage format and
// turns mermaid fenced blocks into attachment image macros.
func ConvertMarkdownToStorageWithMermaid(ctx context.Context, markdownPath string, markdown string) (MermaidResult, error) {
	result := MermaidResult{
		Generated: make(map[string]string),
		SaveCache: func() error { return nil },
	}

	cachePath, err := mermaidCachePath(markdownPath)
	if err != nil {
		return MermaidResult{}, err
	}
	cache, err := loadMermaidCache(cachePath)
	if err != nil {
		return MermaidResult{}, err
	}

	lines := strings.Split(markdown, "\n")
	var (
		parts        []string
		pending      []string
		mermaidIndex int
		seen         = make(map[string]struct{})
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
		fence, hasFence := parseFenceStart(trimmed)
		opts, isMermaidFence := parseMermaidFence(trimmed)
		if isMermaidFence {
			if err := flushPending(); err != nil {
				return MermaidResult{}, err
			}
			start := i
			i++

			var mermaidLines []string
			closed := false
			for i < len(lines) {
				if isFenceClose(lines[i], fence.marker) {
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
				return MermaidResult{}, fmt.Errorf("mermaid block %d exceeds %d chars", mermaidIndex, maxMermaidBlockChars)
			}

			filename := "mermaid-" + strconv.Itoa(mermaidIndex) + ".svg"
			seen[filename] = struct{}{}
			sourceHash := textSHA256(source + "\n---\n" + mermaidOptionsKey(opts))
			cached, hasCache := cache.Entries[filename]
			needRender := !hasCache || cached.Source != sourceHash
			if needRender {
				renderedPath, renderErr := renderMermaidSVG(ctx, source, filename)
				if renderErr != nil {
					return MermaidResult{}, renderErr
				}
				fileHash, hashErr := fileSHA256(renderedPath)
				if hashErr != nil {
					_ = os.Remove(renderedPath)
					_ = os.Remove(filepath.Dir(renderedPath))
					return MermaidResult{}, hashErr
				}
				result.Generated[filename] = renderedPath
				cache.Entries[filename] = mermaidCacheEntry{
					Source: sourceHash,
					File:   fileHash,
				}
			}
			parts = append(parts, buildMermaidImageStorage(filename, opts))
			continue
		}
		if hasFence {
			start := i
			i++
			closed := false
			for i < len(lines) {
				if isFenceClose(lines[i], fence.marker) {
					closed = true
					break
				}
				i++
			}
			if closed {
				pending = append(pending, lines[start:i+1]...)
				continue
			}
			pending = append(pending, lines[start:]...)
			break
		}
		pending = append(pending, lines[i])
	}

	if err := flushPending(); err != nil {
		return MermaidResult{}, err
	}
	for filename := range cache.Entries {
		if _, ok := seen[filename]; !ok {
			delete(cache.Entries, filename)
		}
	}
	result.Storage = strings.Join(parts, "\n")
	result.SaveCache = func() error {
		return saveMermaidCache(cachePath, cache)
	}
	return result, nil
}

func mermaidOptionsKey(opts mermaidOptions) string {
	return "align=" + opts.align + ";width=" + strconv.Itoa(opts.width)
}

// WarmUpMermaidRenderer primes mmdc once to reduce first render latency in watch mode.
func WarmUpMermaidRenderer(ctx context.Context) {
	mermaidWarmupOnce.Do(func() {
		tmpIn, err := os.CreateTemp("", "cfl-mermaid-warmup-*.mmd")
		if err != nil {
			return
		}
		inPath := tmpIn.Name()
		if _, err := tmpIn.WriteString("graph TD\nA-->B\n"); err != nil {
			_ = tmpIn.Close()
			_ = os.Remove(inPath)
			return
		}
		if err := tmpIn.Close(); err != nil {
			_ = os.Remove(inPath)
			return
		}
		defer func() {
			_ = os.Remove(inPath)
		}()

		tmpOut, err := os.CreateTemp("", "cfl-mermaid-warmup-*.svg")
		if err != nil {
			return
		}
		outPath := tmpOut.Name()
		_ = tmpOut.Close()
		defer func() {
			_ = os.Remove(outPath)
		}()

		_ = exec.CommandContext(ctx, "mmdc", "-i", inPath, "-o", outPath).Run()
	})
}

func renderMermaidSVG(ctx context.Context, source string, filename string) (string, error) {
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

	tmpDir, err := os.MkdirTemp("", "cfl-mermaid-svg-*")
	if err != nil {
		return "", err
	}
	svgPath := filepath.Join(tmpDir, filename)

	output, err := exec.CommandContext(ctx, "mmdc", "-i", tmpPath, "-o", svgPath).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("mmdc failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return svgPath, nil
}

type mermaidOptions struct {
	align string
	width int
}

func parseMermaidFence(line string) (mermaidOptions, bool) {
	fence, ok := parseFenceStart(line)
	if !ok {
		return mermaidOptions{}, false
	}
	info := fence.info
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
