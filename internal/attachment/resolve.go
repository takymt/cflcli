package attachment

import (
	"fmt"
	stdhtml "html"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var storageImageURLPattern = regexp.MustCompile(`(?s)<ac:image([^>]*)><ri:url ri:value="([^"]*)" /></ac:image>`)

// Asset describes a local image file to be uploaded as a Confluence attachment.
type Asset struct {
	Filename   string
	SourcePath string
}

// ResolveMarkdownImageAssets rewrites local image URL references in storage to attachment references.
func ResolveMarkdownImageAssets(storage, bodyFilePath, assetsRoot string) (string, []Asset, error) {
	baseDir := filepath.Dir(bodyFilePath)
	if strings.TrimSpace(baseDir) == "" {
		baseDir = "."
	}
	baseDirAbs, err := filepath.Abs(baseDir)
	if err != nil {
		return "", nil, fmt.Errorf("resolve body file directory: %w", err)
	}

	assetsRootAbs, err := resolveAssetsRootPath(assetsRoot, baseDirAbs)
	if err != nil {
		return "", nil, err
	}

	localImageAssets := make([]Asset, 0)
	seenByFilename := make(map[string]string)
	var resolveErr error

	resolvedStorage := storageImageURLPattern.ReplaceAllStringFunc(storage, func(match string) string {
		if resolveErr != nil {
			return match
		}

		submatches := storageImageURLPattern.FindStringSubmatch(match)
		if len(submatches) != 3 {
			return match
		}

		attrs := submatches[1]
		rawSource := stdhtml.UnescapeString(submatches[2])
		source := strings.TrimSpace(rawSource)
		if source == "" || isRemoteImageSource(source) {
			return match
		}

		resolvedPath, err := resolveLocalImagePath(source, baseDirAbs, assetsRootAbs)
		if err != nil {
			resolveErr = err
			return match
		}

		info, err := os.Stat(resolvedPath)
		if err != nil {
			resolveErr = fmt.Errorf("resolve local image %q: %w", source, err)
			return match
		}
		if info.IsDir() {
			resolveErr = fmt.Errorf("resolve local image %q: %q is a directory", source, resolvedPath)
			return match
		}

		filename := filepath.Base(resolvedPath)
		key := strings.ToLower(filename)
		if prev, ok := seenByFilename[key]; ok && filepath.Clean(prev) != filepath.Clean(resolvedPath) {
			resolveErr = fmt.Errorf("duplicate image filename %q from %q and %q; rename one file", filename, prev, resolvedPath)
			return match
		}
		if _, ok := seenByFilename[key]; !ok {
			seenByFilename[key] = resolvedPath
			localImageAssets = append(localImageAssets, Asset{
				Filename:   filename,
				SourcePath: resolvedPath,
			})
		}

		return `<ac:image` + attrs + `><ri:attachment ri:filename="` +
			stdhtml.EscapeString(filename) +
			`" /></ac:image>`
	})

	if resolveErr != nil {
		return "", nil, resolveErr
	}

	return resolvedStorage, localImageAssets, nil
}

func resolveAssetsRootPath(rawRoot, fallback string) (string, error) {
	root := strings.TrimSpace(rawRoot)
	if root == "" {
		root = fallback
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve --assets-root: %w", err)
	}
	return filepath.Clean(rootAbs), nil
}

func isRemoteImageSource(source string) bool {
	lower := strings.ToLower(strings.TrimSpace(source))
	return strings.HasPrefix(lower, "http://") ||
		strings.HasPrefix(lower, "https://") ||
		strings.HasPrefix(lower, "data:") ||
		strings.HasPrefix(lower, "//")
}

func resolveLocalImagePath(source, baseDirAbs, assetsRootAbs string) (string, error) {
	if strings.HasPrefix(source, "/") {
		rel := filepath.Clean(filepath.FromSlash(strings.TrimLeft(source, "/")))
		if rel == "." || rel == "" {
			return "", fmt.Errorf("invalid root-based image path %q", source)
		}
		resolved := filepath.Clean(filepath.Join(assetsRootAbs, rel))
		if !isPathWithinRoot(resolved, assetsRootAbs) {
			return "", fmt.Errorf("image path %q escapes --assets-root %q", source, assetsRootAbs)
		}
		return resolved, nil
	}

	normalized := filepath.FromSlash(source)
	if filepath.IsAbs(normalized) {
		return filepath.Clean(normalized), nil
	}
	return filepath.Clean(filepath.Join(baseDirAbs, normalized)), nil
}

func isPathWithinRoot(path, root string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
