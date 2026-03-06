package page

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

type hashCache struct {
	Entries map[string]string `json:"entries"`
}

type attachmentCache struct {
	PageID  string            `json:"page_id"`
	Entries map[string]string `json:"entries"`
}

func fileSHA256(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func textSHA256(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func mermaidCachePath(markdownPath string) string {
	return markdownPath + ".mermaid-cache.json"
}

func attachmentCachePath(markdownPath string) string {
	return markdownPath + ".attachment-cache.json"
}

func loadHashCache(path string) (hashCache, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return hashCache{Entries: make(map[string]string)}, nil
		}
		return hashCache{}, err
	}
	var cache hashCache
	if err := json.Unmarshal(data, &cache); err == nil {
		if cache.Entries == nil {
			cache.Entries = make(map[string]string)
		}
		return cache, nil
	}
	return hashCache{Entries: make(map[string]string)}, nil
}

func saveHashCache(path string, cache hashCache) error {
	if cache.Entries == nil {
		cache.Entries = make(map[string]string)
	}
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func loadAttachmentCache(path string) (attachmentCache, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return attachmentCache{Entries: make(map[string]string)}, nil
		}
		return attachmentCache{}, err
	}
	var cache attachmentCache
	if err := json.Unmarshal(data, &cache); err == nil {
		if cache.Entries == nil {
			cache.Entries = make(map[string]string)
		}
		return cache, nil
	}
	return attachmentCache{Entries: make(map[string]string)}, nil
}

func saveAttachmentCache(path string, cache attachmentCache) error {
	if cache.Entries == nil {
		cache.Entries = make(map[string]string)
	}
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
