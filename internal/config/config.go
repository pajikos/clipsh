// Package config loads the clipsh TOML configuration file.
//
// The config has a flat shape: an optional default_profile key, and a map of
// named profiles under [profile.<name>] that each fill one or more defaults.
// Absence of the file is not an error — an empty Config is returned.
//
// Profile values never override explicit CLI flags. The caller merges them;
// see ProfileOrEmpty.
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Profile is one named set of defaults.
type Profile struct {
	Host       string   `toml:"host"`
	Port       int      `toml:"port"`
	Identity   string   `toml:"identity"`
	Jump       string   `toml:"jump"`
	RemotePath string   `toml:"remote_path"`
	SSHOpts    []string `toml:"ssh_opts"`
	Hook       string   `toml:"hook"`
}

// Config is the parsed config file.
type Config struct {
	DefaultProfile string             `toml:"default_profile"`
	Profiles       map[string]Profile `toml:"profile"`
}

// Path returns the expected config-file location, respecting $XDG_CONFIG_HOME.
func Path() string {
	if p := os.Getenv("XDG_CONFIG_HOME"); p != "" {
		return filepath.Join(p, "clipsh", "config.toml")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "clipsh/config.toml"
	}
	return filepath.Join(home, ".config", "clipsh", "config.toml")
}

// Load reads and parses the default config path. A missing file is not an
// error; an empty Config is returned instead.
func Load() (*Config, error) {
	return LoadFrom(Path())
}

// LoadFrom reads and parses a specific path. A missing file is not an error.
func LoadFrom(path string) (*Config, error) {
	data, err := os.ReadFile(path) // #nosec G304 - user-provided path is the point
	if errors.Is(err, fs.ErrNotExist) {
		return &Config{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	var cfg Config
	if _, err := toml.Decode(string(data), &cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	return &cfg, nil
}

// ProfileOrEmpty returns the named profile. An empty name or a config with
// no matching profile yields an empty profile rather than an error, so
// callers can unconditionally merge with CLI flags.
func (c *Config) ProfileOrEmpty(name string) Profile {
	if c == nil || name == "" {
		return Profile{}
	}
	if p, ok := c.Profiles[name]; ok {
		return p
	}
	return Profile{}
}

// ResolveProfileName picks a profile name using the documented precedence:
// explicit > CLIPSH_PROFILE env var > default_profile in config.
// Returns "" when no profile is selected.
func (c *Config) ResolveProfileName(explicit string) string {
	if explicit != "" {
		return explicit
	}
	if env := os.Getenv("CLIPSH_PROFILE"); env != "" {
		return env
	}
	if c != nil {
		return c.DefaultProfile
	}
	return ""
}
