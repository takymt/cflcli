package page

import "regexp"

var attachmentFilenameRE = regexp.MustCompile(`ri:attachment\s+ri:filename="([^"]+)"`)

// AttachmentFilenamesFromStorage extracts referenced attachment filenames in appearance order.
func AttachmentFilenamesFromStorage(storage string) []string {
	matches := attachmentFilenameRE.FindAllStringSubmatch(storage, -1)
	if len(matches) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(matches))
	filenames := make([]string, 0, len(matches))
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		name := m[1]
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		filenames = append(filenames, name)
	}
	return filenames
}
