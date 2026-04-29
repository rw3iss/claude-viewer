package data

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/rw3iss/claude-viewer/internal/config"
	dbg "github.com/rw3iss/claude-viewer/internal/debug"
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

	// PrefetchAll runs SessionsRefresh on every enabled dir concurrently.
	// Used at startup to warm the cache in parallel.
	PrefetchAll()
}

type repo struct {
	cfg   *config.Config
	dirs  []ClaudeDir
	cache *CacheStore

	// running-session detection — populated by detectRunning(); cached
	// briefly to avoid /proc thrash if Sessions() is called repeatedly.
	runningCache map[string]bool
	runningAt    time.Time
}

// NewRepo constructs a Repository from the user's config.
func NewRepo(cfg *config.Config) (Repository, error) {
	cache, err := NewCache()
	if err != nil {
		return nil, fmt.Errorf("init cache: %w", err)
	}
	dbg.Logf("cache root=%s", cache.Root())
	r := &repo{cfg: cfg, cache: cache}
	r.refreshDirs()
	dbg.Logf("repo: %d dirs detected", len(r.dirs))
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

// cacheTTL is intentionally generous — first paint comes from cache and a
// re-scan is triggered on user-initiated reload (ctrl+r) or when a session
// file is modified during a chat session (fsnotify).
const cacheTTL = 5 * time.Minute

func (r *repo) Sessions(d ClaudeDir) ([]Session, error) {
	if entry, err := r.cache.Read(d.Path); err == nil && time.Since(entry.GeneratedAt) <= cacheTTL {
		dbg.Logf("Sessions: cache HIT  dir=%s age=%s n=%d", d.Label, time.Since(entry.GeneratedAt).Truncate(time.Millisecond), len(entry.Sessions))
		return r.markRunning(entry.Sessions), nil
	}
	dbg.Logf("Sessions: cache MISS dir=%s — refreshing", d.Label)
	return r.SessionsRefresh(d)
}

// markRunning sets s.Running for any session whose path is held open by a
// live process (Linux: /proc walk). Refreshes its internal cache every 3s.
func (r *repo) markRunning(sessions []Session) []Session {
	if r.runningCache == nil || time.Since(r.runningAt) > 3*time.Second {
		r.runningCache = runningSessionPaths()
		r.runningAt = time.Now()
		dbg.Logf("running detection: %d sessions held open", len(r.runningCache))
	}
	for i := range sessions {
		sessions[i].Running = r.runningCache[sessions[i].Path]
	}
	return sessions
}

func (r *repo) SessionsRefresh(d ClaudeDir) ([]Session, error) {
	t := time.Now()
	sessions, err := LoadSessions(d)
	if err != nil {
		return nil, fmt.Errorf("load sessions for %s: %w", d.Path, err)
	}
	dbg.Logf("LoadSessions: dir=%s n=%d in=%s", d.Label, len(sessions), time.Since(t).Truncate(time.Millisecond))
	if err := r.cache.Write(&CacheEntry{
		ClaudeDirPath: d.Path,
		Sessions:      sessions,
		GeneratedAt:   time.Now(),
	}); err != nil {
		dbg.Logf("cache.Write error (non-fatal): %v", err)
	}
	return r.markRunning(sessions), nil
}

func (r *repo) LookupForCwd(cwd string) (Session, ClaudeDir, bool) {
	cwd = filepath.Clean(cwd)
	dbg.Logf("LookupForCwd: cwd=%q", cwd)
	type cand struct {
		session  Session
		dir      ClaudeDir
		matchLen int
	}
	var best *cand
	scanned := 0
	for _, d := range r.EnabledDirs() {
		sessions, err := r.Sessions(d)
		if err != nil {
			dbg.Logf("LookupForCwd: skip dir=%s err=%v", d.Label, err)
			continue
		}
		for _, s := range sessions {
			scanned++
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
		dbg.Logf("LookupForCwd: no match (%d sessions scanned)", scanned)
		return Session{}, ClaudeDir{}, false
	}
	dbg.Logf("LookupForCwd: match dir=%s session=%s project=%s (matchLen=%d)",
		best.dir.Label, best.session.UUID, best.session.ProjectDir, best.matchLen)
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

// PrefetchAll concurrently refreshes Sessions for every enabled dir. Errors
// are logged (debug only) — they don't bubble up because individual dir
// failures shouldn't fail the prefetch.
func (r *repo) PrefetchAll() {
	dirs := r.EnabledDirs()
	var wg sync.WaitGroup
	wg.Add(len(dirs))
	for _, d := range dirs {
		go func(d ClaudeDir) {
			defer wg.Done()
			if !r.cache.Stale(d.Path, cacheTTL) {
				return
			}
			if _, err := r.SessionsRefresh(d); err != nil {
				dbg.Logf("PrefetchAll: dir=%s err=%v", d.Label, err)
			}
		}(d)
	}
	wg.Wait()
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
