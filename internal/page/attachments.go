package page

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

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

// SyncAttachmentsFromStorage uploads all referenced attachments before page body update.
func SyncAttachmentsFromStorage(ctx context.Context, client Client, pageID string, markdownPath string, storage string) error {
	for _, filename := range AttachmentFilenamesFromStorage(storage) {
		attachmentPath := filepath.Join(filepath.Dir(markdownPath), filename)
		if _, err := os.Stat(attachmentPath); err != nil {
			return fmt.Errorf("attachment file %q not found: %w", filename, err)
		}
		if err := client.PutAttachment(ctx, pageID, attachmentPath); err != nil {
			return fmt.Errorf("put attachment %q: %w", filename, err)
		}
	}
	return nil
}
