package page

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"golang.org/x/sync/errgroup"
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
	filenames := AttachmentFilenamesFromStorage(storage)
	if len(filenames) == 0 {
		return nil
	}

	type fileEntry struct {
		filename string
		path     string
		hash     string
	}
	entries := make([]fileEntry, 0, len(filenames))
	current := make(map[string]string, len(filenames))
	for _, filename := range filenames {
		attachmentPath := filepath.Join(filepath.Dir(markdownPath), filename)
		if _, err := os.Stat(attachmentPath); err != nil {
			return fmt.Errorf("attachment file %q not found: %w", filename, err)
		}
		hash, err := fileSHA256(attachmentPath)
		if err != nil {
			return fmt.Errorf("hash attachment %q: %w", filename, err)
		}
		entries = append(entries, fileEntry{
			filename: filename,
			path:     attachmentPath,
			hash:     hash,
		})
		current[filename] = hash
	}

	cachePath := attachmentCachePath(markdownPath)
	cache, err := loadAttachmentCache(cachePath)
	if err != nil {
		return err
	}
	if cache.PageID != pageID {
		cache = attachmentCache{
			PageID:  pageID,
			Entries: make(map[string]string),
		}
	}

	var toUpload []fileEntry
	for _, entry := range entries {
		if cached, ok := cache.Entries[entry.filename]; ok && cached == entry.hash {
			continue
		}
		toUpload = append(toUpload, entry)
	}

	eg, egCtx := errgroup.WithContext(ctx)
	sem := make(chan struct{}, 4)
	for _, entry := range toUpload {
		entry := entry
		eg.Go(func() error {
			select {
			case sem <- struct{}{}:
			case <-egCtx.Done():
				return egCtx.Err()
			}
			defer func() { <-sem }()
			if err := client.PutAttachment(egCtx, pageID, entry.path); err != nil {
				return fmt.Errorf("put attachment %q: %w", entry.filename, err)
			}
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return err
	}

	cache.PageID = pageID
	cache.Entries = current
	if err := saveAttachmentCache(cachePath, cache); err != nil {
		return err
	}
	return nil
}
