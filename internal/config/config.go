package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Profile represents a Confluence connection profile.
type Profile struct {
	Name       string `toml:"name"`
	Domain     string `toml:"domain"`
	User       string `toml:"user"`
	SpaceKey   string `toml:"space_key"`
	AssetsRoot string `toml:"assets_root"`
	Output     string `toml:"output"`
}

// Config represents the top-level configuration file.
type Config struct {
	Current  string    `toml:"current"`
	Profiles []Profile `toml:"profiles"`
}

// Dir returns the configuration directory path.
// Uses $XDG_CONFIG_HOME/cflcli if set, otherwise ~/.config/cflcli.
func Dir() (string, error) {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("get home directory: %w", err)
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "cflcli"), nil
}

// Path returns the configuration file path.
func Path() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.toml"), nil
}

// Load reads the configuration file and returns the parsed Config.
// If the file does not exist, returns a zero-value Config (no error).
func Load() (*Config, error) {
	p, err := Path()
	if err != nil {
		return nil, err
	}
	return LoadFrom(p)
}

// LoadFrom reads the configuration from the specified path.
func LoadFrom(path string) (*Config, error) {
	cfg := &Config{}
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("stat config %s: %w", path, err)
	}
	if _, err := toml.DecodeFile(path, cfg); err != nil {
		return nil, fmt.Errorf("decode config %s: %w", path, err)
	}
	return cfg, nil
}

// Save writes the configuration to the default config file path.
func (c *Config) Save() error {
	p, err := Path()
	if err != nil {
		return err
	}
	return c.SaveTo(p)
}

// SaveTo writes the configuration to the specified path.
func (c *Config) SaveTo(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("open config file: %w", err)
	}
	defer func() { _ = f.Close() }()

	if err := toml.NewEncoder(f).Encode(c); err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	return nil
}

// AddProfile adds a new profile. If it is the first profile, it is set as current.
// Returns an error if a profile with the same name already exists.
func (c *Config) AddProfile(p *Profile) error {
	for _, existing := range c.Profiles {
		if existing.Name == p.Name {
			return fmt.Errorf("profile %q already exists", p.Name)
		}
	}
	c.Profiles = append(c.Profiles, *p)
	if c.Current == "" {
		c.Current = p.Name
	}
	return nil
}

// FindProfile returns the profile with the given name, or nil if not found.
func (c *Config) FindProfile(name string) *Profile {
	for i := range c.Profiles {
		if c.Profiles[i].Name == name {
			return &c.Profiles[i]
		}
	}
	return nil
}

// SetCurrent sets the current profile to the given name.
// Returns an error if the profile is not found.
func (c *Config) SetCurrent(name string) error {
	if c.FindProfile(name) == nil {
		return fmt.Errorf("profile %q not found", name)
	}
	c.Current = name
	return nil
}

// DeleteProfile removes the profile with the given name.
// Returns an error if the profile is not found.
func (c *Config) DeleteProfile(name string) error {
	for i := range c.Profiles {
		if c.Profiles[i].Name == name {
			c.Profiles = append(c.Profiles[:i], c.Profiles[i+1:]...)
			if c.Current == name {
				c.Current = ""
			}
			return nil
		}
	}
	return fmt.Errorf("profile %q not found", name)
}

// CurrentProfile returns the current active profile, or nil if none is set.
func (c *Config) CurrentProfile() *Profile {
	if c.Current == "" {
		return nil
	}
	return c.FindProfile(c.Current)
}
