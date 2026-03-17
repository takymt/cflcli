package auth

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Store interface {
	Path() (string, error)
	Load() (Credentials, error)
	Save(Credentials) error
	Clear() error
}

type XDGConfigStore struct{}

func NewXDGConfigStore() *XDGConfigStore {
	return &XDGConfigStore{}
}

func (s *XDGConfigStore) Path() (string, error) {
	baseDir, err := configBaseDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(baseDir, "cflcli", "config.yml"), nil
}

func (s *XDGConfigStore) Load() (Credentials, error) {
	path, err := s.Path()
	if err != nil {
		return Credentials{}, err
	}

	config, err := loadConfigMap(path)
	if errors.Is(err, os.ErrNotExist) {
		return Credentials{}, nil
	}
	if err != nil {
		return Credentials{}, err
	}

	return Credentials{
		Domain:   stringValue(config["domain"]),
		Email:    stringValue(config["email"]),
		APIToken: stringValue(config["api_token"]),
	}, nil
}

func (s *XDGConfigStore) Save(creds Credentials) error {
	path, err := s.Path()
	if err != nil {
		return err
	}

	required, err := RequireCredentials(creds)
	if err != nil {
		return err
	}

	config, err := loadConfigMap(path)
	if errors.Is(err, os.ErrNotExist) {
		config = make(map[string]any)
	} else if err != nil {
		return err
	}

	config["domain"] = required.Domain
	config["email"] = required.Email
	config["api_token"] = required.APIToken

	return writeConfigMap(path, config)
}

func (s *XDGConfigStore) Clear() error {
	path, err := s.Path()
	if err != nil {
		return err
	}

	config, err := loadConfigMap(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(config) == 0 {
		return nil
	}

	delete(config, "domain")
	delete(config, "email")
	delete(config, "api_token")

	return writeConfigMap(path, config)
}

func configBaseDir() (string, error) {
	if xdgConfigHome := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME")); xdgConfigHome != "" {
		return xdgConfigHome, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".config"), nil
}

func loadConfigMap(path string) (map[string]any, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(string(raw)) == "" {
		return map[string]any{}, nil
	}

	var config map[string]any
	if err := yaml.Unmarshal(raw, &config); err != nil {
		return nil, fmt.Errorf("load auth config: %w", err)
	}
	if config == nil {
		return map[string]any{}, nil
	}
	return config, nil
}

func writeConfigMap(path string, config map[string]any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	if len(config) == 0 {
		return os.WriteFile(path, []byte{}, 0o600)
	}

	raw, err := yaml.Marshal(config)
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o600)
}

func stringValue(value any) string {
	s, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(s)
}
