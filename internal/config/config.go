// Package config stores which Journal instance the CLI talks to and the session
// it holds for it.
package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// DefaultURL is where the suite's shared Journal instance listens. Every Facile
// app ships here, so this is the one default a CLI can justify — unlike a
// per-deployment tool, there is exactly one canonical Journal.
const DefaultURL = "https://journal.facile.studio"

// Config is the whole stored state. It is small on purpose — everything else
// lives in the instance.
type Config struct {
	URL   string `json:"url"`
	Token string `json:"token,omitempty"`
}

// Dir is the configuration directory, honouring XDG_CONFIG_HOME.
func Dir() string {
	if base := os.Getenv("XDG_CONFIG_HOME"); base != "" {
		return filepath.Join(base, "journal")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".journal"
	}
	return filepath.Join(home, ".config", "journal")
}

// Path is the configuration file.
func Path() string { return filepath.Join(Dir(), "config.json") }

// Load reads the configuration, returning defaults when none exists yet.
func Load() (Config, error) {
	cfg := Config{URL: DefaultURL}

	data, err := os.ReadFile(Path())
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	if cfg.URL == "" {
		cfg.URL = DefaultURL
	}
	return cfg, nil
}

// Save writes the configuration with owner-only permissions. The session token
// is a bearer credential, so the file must not be group- or world-readable, and
// the directory is created 0700 for the same reason.
func Save(cfg Config) error {
	if err := os.MkdirAll(Dir(), 0o700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(Path(), append(data, '\n'), 0o600)
}

// Clear removes the stored session but keeps the instance URL, so logging out
// does not also make the user retype where their Journal is.
func Clear() error {
	cfg, err := Load()
	if err != nil {
		return err
	}
	cfg.Token = ""
	return Save(cfg)
}

// NormalizeURL trims a trailing slash and supplies a scheme, so `journal login
// journal.facile.studio` works as typed.
func NormalizeURL(raw string) string {
	trimmed := strings.TrimRight(strings.TrimSpace(raw), "/")
	if trimmed == "" {
		return ""
	}
	if !strings.HasPrefix(trimmed, "http://") && !strings.HasPrefix(trimmed, "https://") {
		return "https://" + trimmed
	}
	return trimmed
}
