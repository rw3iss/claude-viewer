// Package config persists user preferences to TOML.
//
// Location: $XDG_CONFIG_HOME/claude-viewer/config.toml (with OS-appropriate
// fallback via os.UserConfigDir).
package config

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
)

// Config is the persisted user preferences.
type Config struct {
	// Enabled holds the canonical paths of claude config directories the
	// user wants to see. Empty means "all auto-detected dirs".
	Enabled []string `toml:"enabled"`

	// Disabled holds dirs the user explicitly hid (takes precedence over
	// auto-detect / Enabled).
	Disabled []string `toml:"disabled"`

	// Custom holds manually-added dirs (typically non-standard paths).
	Custom []string `toml:"custom"`

	// LastSession is the path of the most recently opened session.jsonl,
	// used as a fallback when no cwd match is found.
	LastSession string `toml:"last_session"`

	// Theme name (resolved via theme.Get).
	Theme string `toml:"theme"`

	// PreviewRows is the default wrap-row count in chat view.
	PreviewRows int `toml:"preview_rows"`

	// PreviewSize is the default preview-pane size %.
	PreviewSize int `toml:"preview_size"`

	// Layout is the default chat-view layout: "bottom" or "right".
	Layout string `toml:"layout"`

	// HeaderShow* toggles header widgets.
	HeaderShowName bool `toml:"header_show_name"`
	HeaderShowDir  bool `toml:"header_show_dir"`
	HeaderShowOrg  bool `toml:"header_show_org"`
	HeaderShowCfg  bool `toml:"header_show_cfg"`
	HeaderShowUUID bool `toml:"header_show_uuid"`

	// ActiveMinutes — sessions with mtime within this window count as
	// "active" for the purpose of the menu/all-orgs list grouping.
	// Running sessions (process holds the file open) ignore this and are
	// always shown as active.
	ActiveMinutes int `toml:"active_minutes"`
}

// ActiveDuration returns the configured active-session window as a Duration,
// falling back to 60 minutes if unset.
func (c Config) ActiveDuration() time.Duration {
	if c.ActiveMinutes <= 0 {
		return 60 * time.Minute
	}
	return time.Duration(c.ActiveMinutes) * time.Minute
}

// Default returns sensible defaults.
func Default() Config {
	return Config{
		Theme:          "default",
		PreviewRows:    2,
		PreviewSize:    60,
		Layout:         "bottom",
		HeaderShowName: true,
		HeaderShowDir:  true,
		HeaderShowOrg:  true,
		HeaderShowCfg:  true,
		HeaderShowUUID: true,
		ActiveMinutes:  60,
	}
}

// Path returns the resolved config-file path.
func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "claude-viewer", "config.toml"), nil
}

var (
	mu     sync.Mutex
	loaded *Config
)

// Load reads (or creates) the config file. Subsequent calls return the cached
// value. Use Save() to persist mutations.
func Load() (*Config, error) {
	mu.Lock()
	defer mu.Unlock()
	if loaded != nil {
		return loaded, nil
	}
	p, err := Path()
	if err != nil {
		return nil, err
	}
	cfg := Default()
	if _, err := os.Stat(p); err == nil {
		if _, err := toml.DecodeFile(p, &cfg); err != nil {
			return nil, err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	loaded = &cfg
	return loaded, nil
}

// Save persists the current config.
func Save(c *Config) error {
	mu.Lock()
	defer mu.Unlock()
	p, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(p), ".config.*.toml")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if err := toml.NewEncoder(tmp).Encode(c); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp.Name(), p); err != nil {
		return err
	}
	loaded = c
	return nil
}

// Reset clears the in-memory cache (mainly for tests).
func Reset() {
	mu.Lock()
	loaded = nil
	mu.Unlock()
}
