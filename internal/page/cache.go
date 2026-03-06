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

func mermaidCachePath(markdownPath string) (string, error) {
	return cachePathFor("mermaid", markdownPath)
}

func attachmentCachePath(markdownPath string) (string, error) {
	return cachePathFor("attachments", markdownPath)
}

func cachePathFor(kind string, markdownPath string) (string, error) {
	base, key, err := cacheBaseAndKey(markdownPath)
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "cflcli", kind, key+".json"), nil
}

func cacheBaseAndKey(markdownPath string) (string, string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", "", err
	}
	abs := markdownPath
	if resolved, absErr := filepath.Abs(markdownPath); absErr == nil {
		abs = resolved
	}
	return base, textSHA256(abs), nil
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
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
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
