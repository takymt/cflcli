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
func SyncAttachmentsFromStorage(ctx context.Context, client Client, pageID string, markdownPath string, storage string, generatedMermaid map[string]string) error {
	filenames := AttachmentFilenamesFromStorage(storage)
	if len(filenames) == 0 {
		return nil
	}
	mermaidCacheFile, err := mermaidCachePath(markdownPath)
	if err != nil {
		return err
	}
	mermaidCache, err := loadHashCache(mermaidCacheFile)
	if err != nil {
		return err
	}

	type fileEntry struct {
		filename string
		path     string
		hash     string
	}
	entries := make([]fileEntry, 0, len(filenames))
	current := make(map[string]string, len(filenames))
	for _, filename := range filenames {
		entry := fileEntry{filename: filename}
		if generatedPath, ok := generatedMermaid[filename]; ok {
			entry.path = generatedPath
			hash, hashErr := fileSHA256(generatedPath)
			if hashErr != nil {
				return fmt.Errorf("hash attachment %q: %w", filename, hashErr)
			}
			entry.hash = hash
		} else if attachmentPath, resolveErr := resolveAttachmentPath(markdownPath, filename); resolveErr == nil {
			entry.path = attachmentPath
			hash, hashErr := fileSHA256(attachmentPath)
			if hashErr != nil {
				return fmt.Errorf("hash attachment %q: %w", filename, hashErr)
			}
			entry.hash = hash
		} else if hash, ok := mermaidCache.Entries[filename]; ok {
			entry.hash = hash
		} else {
			return fmt.Errorf("attachment file %q not found: %w", filename, os.ErrNotExist)
		}
		entries = append(entries, entry)
		current[filename] = entry.hash
	}

	cachePath, err := attachmentCachePath(markdownPath)
	if err != nil {
		return err
	}
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

	for _, entry := range toUpload {
		if entry.path == "" {
			return fmt.Errorf("attachment %q changed but render output is unavailable", entry.filename)
		}
		if err := client.PutAttachment(ctx, pageID, entry.path); err != nil {
			return fmt.Errorf("put attachment %q: %w", entry.filename, err)
		}
	}

	cache.PageID = pageID
	cache.Entries = current
	if err := saveAttachmentCache(cachePath, cache); err != nil {
		return err
	}
	return nil
}

func resolveAttachmentPath(markdownPath string, filename string) (string, error) {
	attachmentPath := filepath.Join(filepath.Dir(markdownPath), filename)
	if _, err := os.Stat(attachmentPath); err == nil {
		return attachmentPath, nil
	}
	return "", os.ErrNotExist
}
