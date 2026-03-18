package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

var windowsReservedPageNames = map[string]struct{}{
	"AUX":  {},
	"COM1": {},
	"COM2": {},
	"COM3": {},
	"COM4": {},
	"COM5": {},
	"COM6": {},
	"COM7": {},
	"COM8": {},
	"COM9": {},
	"CON":  {},
	"LPT1": {},
	"LPT2": {},
	"LPT3": {},
	"LPT4": {},
	"LPT5": {},
	"LPT6": {},
	"LPT7": {},
	"LPT8": {},
	"LPT9": {},
	"NUL":  {},
	"PRN":  {},
}

func normalizePageTitle(title string) (string, error) {
	trimmed := strings.TrimSpace(title)
	if trimmed == "" {
		return "", errors.New("title must not be empty")
	}
	return trimmed, nil
}

func resolvePageNewTarget(workdir string, title string, pathArg string) (string, string, error) {
	if strings.TrimSpace(pathArg) != "" {
		return validatePageNewPath(workdir, pathArg)
	}

	filename, err := defaultPageNewFilename(title)
	if err != nil {
		return "", "", err
	}
	return validatePageNewPath(workdir, filename)
}

func defaultPageNewFilename(title string) (string, error) {
	stem := sanitizePageNewFilename(title)
	if stem == "" {
		return "", errors.New("title cannot be converted into a default filename; pass --path")
	}
	if isWindowsReservedPageName(stem) {
		return "", fmt.Errorf("title %q cannot be used to derive a default filename on Windows; pass --path", title)
	}
	return stem + ".md", nil
}

func validatePageNewPath(workdir string, pathArg string) (string, string, error) {
	trimmed := strings.TrimSpace(pathArg)
	if trimmed == "" {
		return "", "", errors.New("--path must not be empty")
	}

	display := filepath.Clean(trimmed)
	resolved := resolvePath(workdir, trimmed)
	if !strings.EqualFold(filepath.Ext(resolved), ".md") {
		return "", "", errors.New("--path must point to a markdown file ending in .md")
	}

	stem := strings.TrimSuffix(filepath.Base(resolved), filepath.Ext(resolved))
	if strings.Trim(stem, ". ") == "" {
		return "", "", errors.New("--path must include a filename before .md")
	}

	info, err := os.Stat(resolved)
	switch {
	case err == nil && info.IsDir():
		return "", "", fmt.Errorf("target path %q is a directory", display)
	case err == nil:
		return "", "", fmt.Errorf("target file %q already exists", display)
	case !errors.Is(err, os.ErrNotExist):
		return "", "", err
	}

	parent := filepath.Dir(resolved)
	parentInfo, err := os.Stat(parent)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", "", fmt.Errorf("parent directory %q does not exist", filepath.Dir(display))
		}
		return "", "", err
	}
	if !parentInfo.IsDir() {
		return "", "", fmt.Errorf("parent path %q is not a directory", filepath.Dir(display))
	}

	return resolved, display, nil
}

func sanitizePageNewFilename(title string) string {
	var b strings.Builder
	lastWasSpace := false
	for _, r := range title {
		switch {
		case unicode.IsControl(r):
			if !lastWasSpace {
				b.WriteByte(' ')
				lastWasSpace = true
			}
		case unicode.IsSpace(r):
			if !lastWasSpace {
				b.WriteByte(' ')
				lastWasSpace = true
			}
		case strings.ContainsRune(`/\:*?"<>|`, r):
			if !lastWasSpace {
				b.WriteByte(' ')
				lastWasSpace = true
			}
		default:
			b.WriteRune(r)
			lastWasSpace = false
		}
	}

	sanitized := strings.TrimSpace(b.String())
	return strings.TrimRight(sanitized, ". ")
}

func isWindowsReservedPageName(stem string) bool {
	_, ok := windowsReservedPageNames[strings.ToUpper(stem)]
	return ok
}
