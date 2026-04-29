package data

import (
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/rw3iss/claude-viewer/internal/config"
)

// Repository is the small surface UI uses for data access. It wraps disk
// scanning + caching. Tests can swap in a fake.
type Repository interface {
	// Dirs returns the resolved list of ClaudeDirs (auto-detected merged
	// with custom + disabled config).
	Dirs() []ClaudeDir

	// EnabledDirs returns Dirs() minus the ones marked disabled.
	EnabledDirs() []ClaudeDir

	// Sessions returns the session list for a single ClaudeDir, using cache
	// if fresh.
	Sessions(claudeDir ClaudeDir) ([]Session, error)

	// SessionsRefresh forces a re-scan, ignoring cache.
	SessionsRefresh(claudeDir ClaudeDir) ([]Session, error)

	// LookupForCwd returns the most-recently-modified session whose
	// project dir is equal to or an ancestor of cwd, across all enabled
	// dirs. Returns the empty Session and false if nothing matches.
	LookupForCwd(cwd string) (Session, ClaudeDir, bool)

	// AddCustom appends a custom path to config.Custom (after validating
	// it's a plausible Claude config dir). Returns an error otherwise.
	AddCustom(path string) error

	// SetDisabled persists the disabled flag for a dir.
	SetDisabled(path string, disabled bool) error

	// RemoveCustom deletes a custom-added dir from config.
	RemoveCustom(path string) error
}

type repo struct {
	cfg   *config.Config
	dirs  []ClaudeDir
	cache *CacheStore
}

// NewRepo constructs a Repository from the user's config.
func NewRepo(cfg *config.Config) (Repository, error) {
	cache, err := NewCache()
	if err != nil {
		return nil, err
	}
	r := &repo{cfg: cfg, cache: cache}
	r.refreshDirs()
	return r, nil
}

func (r *repo) refreshDirs() {
	detected, _ := DiscoverClaudeDirs()
	r.dirs = MergeWithConfig(detected, r.cfg.Custom, r.cfg.Disabled)
}

func (r *repo) Dirs() []ClaudeDir { return r.dirs }

func (r *repo) EnabledDirs() []ClaudeDir {
	out := make([]ClaudeDir, 0, len(r.dirs))
	for _, d := range r.dirs {
		if !d.Disabled {
			out = append(out, d)
		}
	}
	return out
}

const cacheTTL = 5 * time.Second

func (r *repo) Sessions(d ClaudeDir) ([]Session, error) {
	if entry, err := r.cache.Read(d.Path); err == nil && time.Since(entry.GeneratedAt) <= cacheTTL {
		return entry.Sessions, nil
	}
	return r.SessionsRefresh(d)
}

func (r *repo) SessionsRefresh(d ClaudeDir) ([]Session, error) {
	sessions, err := LoadSessions(d)
	if err != nil {
		return nil, err
	}
	_ = r.cache.Write(&CacheEntry{
		ClaudeDirPath: d.Path,
		Sessions:      sessions,
		GeneratedAt:   time.Now(),
	})
	return sessions, nil
}

func (r *repo) LookupForCwd(cwd string) (Session, ClaudeDir, bool) {
	cwd = filepath.Clean(cwd)
	type cand struct {
		session   Session
		dir       ClaudeDir
		matchLen  int
	}
	var best *cand
	for _, d := range r.EnabledDirs() {
		sessions, err := r.Sessions(d)
		if err != nil {
			continue
		}
		for _, s := range sessions {
			pd := s.ProjectDir
			if pd == "" {
				continue
			}
			pd = filepath.Clean(pd)
			if pd == cwd || strings.HasPrefix(cwd+"/", pd+"/") {
				ml := len(pd)
				if best == nil ||
					ml > best.matchLen ||
					(ml == best.matchLen && s.Mtime.After(best.session.Mtime)) {
					best = &cand{session: s, dir: d, matchLen: ml}
				}
			}
		}
	}
	if best == nil {
		return Session{}, ClaudeDir{}, false
	}
	return best.session, best.dir, true
}

func (r *repo) AddCustom(path string) error {
	d := AsCustomDir(path)
	if !d.IsValid() {
		return errInvalidClaudeDir(path)
	}
	for _, p := range r.cfg.Custom {
		if p == path {
			return nil // already present, no-op
		}
	}
	r.cfg.Custom = append(r.cfg.Custom, path)
	sort.Strings(r.cfg.Custom)
	if err := config.Save(r.cfg); err != nil {
		return err
	}
	r.refreshDirs()
	return nil
}

func (r *repo) SetDisabled(path string, disabled bool) error {
	// Strip from disabled list, then re-add if true.
	out := r.cfg.Disabled[:0]
	for _, p := range r.cfg.Disabled {
		if p != path {
			out = append(out, p)
		}
	}
	r.cfg.Disabled = out
	if disabled {
		r.cfg.Disabled = append(r.cfg.Disabled, path)
	}
	if err := config.Save(r.cfg); err != nil {
		return err
	}
	r.refreshDirs()
	return nil
}

func (r *repo) RemoveCustom(path string) error {
	out := r.cfg.Custom[:0]
	for _, p := range r.cfg.Custom {
		if p != path {
			out = append(out, p)
		}
	}
	r.cfg.Custom = out
	if err := config.Save(r.cfg); err != nil {
		return err
	}
	r.refreshDirs()
	return nil
}

type invalidClaudeDirErr struct{ path string }

func (e invalidClaudeDirErr) Error() string {
	return "not a Claude config directory: " + e.path
}

func errInvalidClaudeDir(p string) error { return invalidClaudeDirErr{p} }
