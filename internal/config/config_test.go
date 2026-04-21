package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	return path
}

func TestLoadFrom_Missing(t *testing.T) {
	cfg, err := LoadFrom(filepath.Join(t.TempDir(), "nope.toml"))
	if err != nil {
		t.Fatalf("missing file should not error: %v", err)
	}
	if cfg == nil || len(cfg.Profiles) != 0 || cfg.DefaultProfile != "" {
		t.Errorf("expected empty config, got %+v", cfg)
	}
}

func TestLoadFrom_FullProfile(t *testing.T) {
	path := writeConfig(t, `
default_profile = "dev"

[profile.dev]
host = "me@dev.example.com"
port = 2222
identity = "~/.ssh/id_ed25519"
jump = "bastion"
remote_path = "/home/me/.clipboard.{ext}"
ssh_opts = ["StrictHostKeyChecking=no", "UserKnownHostsFile=/dev/null"]
hook = "tmux:main"

[profile.prod]
host = "deploy@prod"
`)
	cfg, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DefaultProfile != "dev" {
		t.Errorf("default_profile = %q", cfg.DefaultProfile)
	}
	if len(cfg.Profiles) != 2 {
		t.Errorf("expected 2 profiles, got %d", len(cfg.Profiles))
	}

	dev := cfg.ProfileOrEmpty("dev")
	want := Profile{
		Host:       "me@dev.example.com",
		Port:       2222,
		Identity:   "~/.ssh/id_ed25519",
		Jump:       "bastion",
		RemotePath: "/home/me/.clipboard.{ext}",
		SSHOpts:    []string{"StrictHostKeyChecking=no", "UserKnownHostsFile=/dev/null"},
		Hook:       "tmux:main",
	}
	if !reflect.DeepEqual(dev, want) {
		t.Errorf("dev profile:\n  got:  %+v\n  want: %+v", dev, want)
	}
}

func TestProfileOrEmpty_Unknown(t *testing.T) {
	cfg := &Config{Profiles: map[string]Profile{"dev": {Host: "h"}}}
	got := cfg.ProfileOrEmpty("nope")
	if !reflect.DeepEqual(got, Profile{}) {
		t.Errorf("unknown profile should return zero value, got %+v", got)
	}
}

func TestProfileOrEmpty_Nil(t *testing.T) {
	var c *Config
	got := c.ProfileOrEmpty("dev")
	if !reflect.DeepEqual(got, Profile{}) {
		t.Errorf("nil config should return zero Profile, got %+v", got)
	}
}

func TestResolveProfileName_Precedence(t *testing.T) {
	t.Setenv("CLIPSH_PROFILE", "from-env")
	cfg := &Config{DefaultProfile: "from-config"}

	if got := cfg.ResolveProfileName("from-flag"); got != "from-flag" {
		t.Errorf("flag should win, got %q", got)
	}
	if got := cfg.ResolveProfileName(""); got != "from-env" {
		t.Errorf("env should beat config, got %q", got)
	}
	t.Setenv("CLIPSH_PROFILE", "")
	if got := cfg.ResolveProfileName(""); got != "from-config" {
		t.Errorf("config default should apply, got %q", got)
	}
}

func TestLoadFrom_MalformedTOML(t *testing.T) {
	path := writeConfig(t, `this is = invalid = toml`)
	if _, err := LoadFrom(path); err == nil {
		t.Fatal("expected parse error for malformed TOML")
	}
}
