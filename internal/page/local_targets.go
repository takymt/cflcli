package page

import (
	"net/url"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

var windowsAbsPathRE = regexp.MustCompile(`^([A-Za-z]:[\\/]|\\\\)`)

func attachmentFilenameFromTarget(target string) string {
	localPath, _, ok := localPathFromTarget(target)
	if !ok {
		return ""
	}
	name := path.Base(strings.ReplaceAll(localPath, `\`, `/`))
	if name == "." || name == "/" || name == "" {
		return ""
	}
	return name
}

func isRelativeFilesystemTarget(target string) bool {
	_, absolute, ok := localPathFromTarget(target)
	return ok && !absolute
}

func localPathFromTarget(target string) (string, bool, bool) {
	target = strings.TrimSpace(target)
	if target == "" || strings.HasPrefix(target, "#") {
		return "", false, false
	}
	if windowsAbsPathRE.MatchString(target) || strings.HasPrefix(target, "//") {
		return target, true, true
	}

	parsed, err := url.Parse(target)
	if err != nil {
		return "", false, false
	}
	if parsed.Scheme != "" || parsed.Host != "" {
		return "", false, false
	}
	if parsed.Path == "" {
		return "", false, false
	}

	localPath := parsed.Path
	if decoded, err := url.PathUnescape(localPath); err == nil {
		localPath = decoded
	}
	if filepath.IsAbs(localPath) || strings.HasPrefix(parsed.Path, "/") {
		return localPath, true, true
	}
	return localPath, false, true
}
