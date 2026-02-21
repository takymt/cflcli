package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

const repoConfigFilename = "cfl.toml"

// RepoConfig represents repository-level defaults loaded from cfl.toml.
type RepoConfig struct {
	Domain      string `toml:"domain"`
	SpaceKey    string `toml:"space_key"`
	SpaceID     string `toml:"space_id"`
	ContentRoot string `toml:"content_root"`
}

// ApplyRepoDomain returns a copy of profile with repo domain override, when provided.
func ApplyRepoDomain(profile *Profile, repoCfg *RepoConfig) *Profile {
	if profile == nil {
		return nil
	}
	merged := *profile
	if repoCfg == nil {
		return &merged
	}
	if domain := strings.TrimSpace(repoCfg.Domain); domain != "" {
		merged.Domain = domain
	}
	return &merged
}

// ResolveRepoSpaceSelectors returns normalized space selectors from repo config.
// space_id and space_key are mutually exclusive when both are set.
func ResolveRepoSpaceSelectors(repoCfg *RepoConfig) (string, string, error) {
	if repoCfg == nil {
		return "", "", nil
	}
	spaceID := strings.TrimSpace(repoCfg.SpaceID)
	spaceKey := strings.TrimSpace(repoCfg.SpaceKey)
	if spaceID != "" && spaceKey != "" {
		return "", "", fmt.Errorf("repo config sets both space_id and space_key; specify only one")
	}
	return spaceID, spaceKey, nil
}

// ResolveContentRoot resolves the effective root path for /-prefixed content paths.
// explicitRoot has priority over repo content_root.
func ResolveContentRoot(explicitRoot string, repoCfg *RepoConfig, repoConfigPath string) string {
	if strings.TrimSpace(explicitRoot) != "" || repoCfg == nil {
		return explicitRoot
	}
	contentRoot := strings.TrimSpace(repoCfg.ContentRoot)
	if contentRoot == "" {
		return explicitRoot
	}
	if filepath.IsAbs(contentRoot) {
		return contentRoot
	}
	if strings.TrimSpace(repoConfigPath) == "" {
		return contentRoot
	}
	return filepath.Join(filepath.Dir(repoConfigPath), contentRoot)
}

// DiscoverRepoConfig searches for cfl.toml from startPath up to filesystem root.
// If startPath is empty, the current working directory is used.
// Returns nil config and empty path when not found.
func DiscoverRepoConfig(startPath string) (*RepoConfig, string, error) {
	startDir, err := resolveRepoSearchStartDir(startPath)
	if err != nil {
		return nil, "", err
	}

	dir := startDir
	for {
		candidate := filepath.Join(dir, repoConfigFilename)
		info, statErr := os.Stat(candidate)
		switch {
		case statErr == nil:
			if info.IsDir() {
				return nil, "", fmt.Errorf("repo config %q is a directory", candidate)
			}
			cfg, err := LoadRepoConfigFrom(candidate)
			if err != nil {
				return nil, "", err
			}
			return cfg, candidate, nil
		case os.IsNotExist(statErr):
			parent := filepath.Dir(dir)
			if parent == dir {
				return nil, "", nil
			}
			dir = parent
		default:
			return nil, "", fmt.Errorf("stat repo config %s: %w", candidate, statErr)
		}
	}
}

// LoadRepoConfigFrom loads cfl.toml from a specific path.
func LoadRepoConfigFrom(path string) (*RepoConfig, error) {
	cfg := &RepoConfig{}
	if _, err := toml.DecodeFile(path, cfg); err != nil {
		return nil, fmt.Errorf("decode repo config %s: %w", path, err)
	}
	return cfg, nil
}

func resolveRepoSearchStartDir(startPath string) (string, error) {
	trimmed := strings.TrimSpace(startPath)
	if trimmed == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("get cwd: %w", err)
		}
		return cwd, nil
	}

	abs, err := filepath.Abs(trimmed)
	if err != nil {
		return "", fmt.Errorf("resolve start path: %w", err)
	}

	info, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return filepath.Dir(abs), nil
		}
		return "", fmt.Errorf("stat start path %q: %w", abs, err)
	}

	if info.IsDir() {
		return abs, nil
	}
	return filepath.Dir(abs), nil
}
