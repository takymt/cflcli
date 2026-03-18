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
	mermaidCache, err := loadMermaidCache(mermaidCacheFile)
	if err != nil {
		return err
	}

	current := make(map[string]string, len(filenames))

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

	entries, err := collectAttachmentEntries(markdownPath, filenames, generatedMermaid, mermaidCache)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		filename := entry.filename
		hash := entry.hash
		if cached, ok := cache.Entries[filename]; ok && cached == hash {
			current[filename] = hash
			continue
		}
		if entry.path == "" {
			return fmt.Errorf("attachment %q changed but render output is unavailable", filename)
		}
		if err := client.PutAttachment(ctx, pageID, entry.path); err != nil {
			return fmt.Errorf("put attachment %q: %w", filename, err)
		}
		current[filename] = hash
	}

	cache.PageID = pageID
	cache.Entries = current
	if err := saveAttachmentCache(cachePath, cache); err != nil {
		return err
	}
	return nil
}

type attachmentEntry struct {
	filename string
	path     string
	hash     string
}

func collectAttachmentEntries(markdownPath string, filenames []string, generatedMermaid map[string]string, mermaidCache mermaidCache) ([]attachmentEntry, error) {
	entries := make([]attachmentEntry, 0, len(filenames))
	for _, filename := range filenames {
		entry, err := buildAttachmentEntry(markdownPath, filename, generatedMermaid, mermaidCache)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func buildAttachmentEntry(markdownPath string, filename string, generatedMermaid map[string]string, mermaidCache mermaidCache) (attachmentEntry, error) {
	if generatedPath, ok := generatedMermaid[filename]; ok {
		hash, err := fileSHA256(generatedPath)
		if err != nil {
			return attachmentEntry{}, fmt.Errorf("hash attachment %q: %w", filename, err)
		}
		return attachmentEntry{filename: filename, path: generatedPath, hash: hash}, nil
	}

	if attachmentPath, err := resolveAttachmentPath(markdownPath, filename); err == nil {
		hash, hashErr := fileSHA256(attachmentPath)
		if hashErr != nil {
			return attachmentEntry{}, fmt.Errorf("hash attachment %q: %w", filename, hashErr)
		}
		return attachmentEntry{filename: filename, path: attachmentPath, hash: hash}, nil
	}

	if entry, ok := mermaidCache.Entries[filename]; ok && entry.File != "" {
		return attachmentEntry{filename: filename, hash: entry.File}, nil
	}
	return attachmentEntry{}, fmt.Errorf("attachment file %q not found: %w", filename, os.ErrNotExist)
}

func resolveAttachmentPath(markdownPath string, filename string) (string, error) {
	attachmentPath := filepath.Join(filepath.Dir(markdownPath), filename)
	if _, err := os.Stat(attachmentPath); err == nil {
		return attachmentPath, nil
	}
	return "", os.ErrNotExist
}
