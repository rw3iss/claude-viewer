package data

import (
	"encoding/json"
	"errors"
	"hash/fnv"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// CacheEntry is the cached form of a session list for one ClaudeDir.
type CacheEntry struct {
	ClaudeDirPath string    `json:"claude_dir"`
	Sessions      []Session `json:"sessions"`
	GeneratedAt   time.Time `json:"generated_at"`
}

// CacheStore is the disk-backed cache (one file per ClaudeDir hash).
type CacheStore struct {
	root string
	mu   sync.Mutex
}

// NewCache returns a CacheStore rooted at $XDG_CACHE_HOME/claude-viewer (or
// the OS-appropriate equivalent).
func NewCache() (*CacheStore, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return nil, err
	}
	root := filepath.Join(dir, "claude-viewer")
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}
	return &CacheStore{root: root}, nil
}

// Root returns the cache root path (for diagnostics / clear).
func (c *CacheStore) Root() string { return c.root }

func (c *CacheStore) keyFor(claudeDirPath string) string {
	h := fnv.New64a()
	h.Write([]byte(claudeDirPath))
	return filepath.Join(c.root, "sessions-"+toHex(h.Sum64())+".json")
}

func toHex(u uint64) string {
	const hexDigits = "0123456789abcdef"
	out := make([]byte, 16)
	for i := 15; i >= 0; i-- {
		out[i] = hexDigits[u&0xF]
		u >>= 4
	}
	return string(out)
}

// Read loads a cached session list. ErrNotExist is returned if missing.
func (c *CacheStore) Read(claudeDirPath string) (*CacheEntry, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	p := c.keyFor(claudeDirPath)
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var e CacheEntry
	if err := json.NewDecoder(f).Decode(&e); err != nil {
		return nil, err
	}
	return &e, nil
}

// Write persists the cached list (atomic via temp+rename).
func (c *CacheStore) Write(e *CacheEntry) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	p := c.keyFor(e.ClaudeDirPath)
	tmp, err := os.CreateTemp(c.root, "sessions-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	enc := json.NewEncoder(tmp)
	enc.SetIndent("", "  ")
	if err := enc.Encode(e); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), p)
}

// Stale returns true if a cache entry is older than ttl, or doesn't exist.
func (c *CacheStore) Stale(claudeDirPath string, ttl time.Duration) bool {
	p := c.keyFor(claudeDirPath)
	info, err := os.Stat(p)
	if err != nil {
		return errors.Is(err, os.ErrNotExist) || true
	}
	return time.Since(info.ModTime()) > ttl
}

// Clear deletes all cached entries.
func (c *CacheStore) Clear() error {
	entries, err := os.ReadDir(c.root)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".json" {
			os.Remove(filepath.Join(c.root, e.Name()))
		}
	}
	return nil
}
