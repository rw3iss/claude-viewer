package data

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Session is the metadata view of a single .jsonl session file.
type Session struct {
	UUID       string    // basename minus .jsonl
	Path       string    // full path to .jsonl
	ClaudeDir  string    // owning config dir path
	ProjectDir string    // resolved cwd from JSONL (or slug-derived)
	Slug       string    // dirname under projects/  (e.g. -home-rw3iss-Sites-blobs)
	CustomName string    // /rename value if any
	OrgName    string    // org of the owning ClaudeDir
	Mtime      time.Time // file mtime
	LineCount  int       // approx prompt count (cached)

	// Running is set at query time (not persisted) when a live process
	// holds the JSONL open. True ⇒ this session is actually running right
	// now. Linux only via /proc walk; nil/false elsewhere.
	Running bool `json:"-"`
}

// ShortUUID returns the first 8 chars of UUID.
func (s Session) ShortUUID() string {
	if len(s.UUID) >= 8 {
		return s.UUID[:8]
	}
	return s.UUID
}

// Display returns the preferred title for list rendering.
func (s Session) Display() string {
	switch {
	case s.CustomName != "":
		return s.CustomName
	case s.ProjectDir != "":
		return abbrevHome(s.ProjectDir)
	default:
		return s.Slug
	}
}

// IsActive returns true if mtime is within the cutoff (recently modified).
func (s Session) IsActive(cutoff time.Duration) bool {
	return time.Since(s.Mtime) <= cutoff
}

func abbrevHome(p string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return p
	}
	if rest, ok := strings.CutPrefix(p, home); ok {
		return "~" + rest
	}
	return p
}

// LoadSessions enumerates all .jsonl session files for a ClaudeDir, reading
// just enough of each to populate Session metadata.
func LoadSessions(c ClaudeDir) ([]Session, error) {
	root := c.ProjectsRoot()
	entries, err := os.ReadDir(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var out []Session
	for _, projDir := range entries {
		if !projDir.IsDir() {
			continue
		}
		dirPath := filepath.Join(root, projDir.Name())
		jsonls, err := os.ReadDir(dirPath)
		if err != nil {
			continue
		}
		for _, j := range jsonls {
			name := j.Name()
			if !strings.HasSuffix(name, ".jsonl") {
				continue
			}
			full := filepath.Join(dirPath, name)
			info, err := j.Info()
			if err != nil {
				continue
			}
			s := Session{
				UUID:      strings.TrimSuffix(name, ".jsonl"),
				Path:      full,
				ClaudeDir: c.Path,
				Slug:      projDir.Name(),
				OrgName:   c.OrgName,
				Mtime:     info.ModTime(),
			}
			ScanSessionMeta(&s)
			out = append(out, s)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Mtime.After(out[j].Mtime) })
	return out, nil
}

// ScanSessionMeta reads just enough of the JSONL to populate CustomName +
// ProjectDir without paying for a full-file scan on every list load.
//
//   - cwd is set in nearly every user/assistant entry, so the first 8 KB
//     almost always contains it.
//   - customTitle is appended by /rename; the latest one wins. Reading the
//     last 16 KB and taking the most recent customTitle line catches it.
//
// Falls back to slug-derived ProjectDir if the file is unreadable / has no
// cwd in its head.
func ScanSessionMeta(s *Session) {
	f, err := os.Open(s.Path)
	if err != nil {
		s.ProjectDir = strings.ReplaceAll(s.Slug, "-", "/")
		return
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		s.ProjectDir = strings.ReplaceAll(s.Slug, "-", "/")
		return
	}
	size := info.Size()

	const headBytes = 8 * 1024
	const tailBytes = 16 * 1024

	// HEAD: scan for cwd
	headLen := int64(headBytes)
	if size < headLen {
		headLen = size
	}
	head := make([]byte, headLen)
	if _, err := f.ReadAt(head, 0); err == nil {
		scanLines(head, s, false)
	}

	// TAIL: scan for the most-recent customTitle. Skip if file is small
	// (we already covered everything above).
	if size > headLen {
		off := size - tailBytes
		skipFirst := true
		if off < headLen { // overlap or tiny: read remainder, no skip needed
			off = headLen
			skipFirst = false
		}
		tail := make([]byte, size-off)
		if _, err := f.ReadAt(tail, off); err == nil {
			scanLines(tail, s, skipFirst)
		}
	}

	if s.ProjectDir == "" {
		s.ProjectDir = strings.ReplaceAll(s.Slug, "-", "/")
	}
}

// scanLines walks newline-separated entries in buf. If skipFirstPartial is
// true, the first incomplete line is dropped (because buf likely starts mid-line).
func scanLines(buf []byte, s *Session, skipFirstPartial bool) {
	start := 0
	if skipFirstPartial {
		if i := indexNewline(buf); i >= 0 {
			start = i + 1
		} else {
			return
		}
	}
	for start < len(buf) {
		end := start
		for end < len(buf) && buf[end] != '\n' {
			end++
		}
		if end > start {
			handleSessionMetaLine(s, buf[start:end])
		}
		start = end + 1
	}
}

func indexNewline(b []byte) int {
	for i, c := range b {
		if c == '\n' {
			return i
		}
	}
	return -1
}

func handleSessionMetaLine(s *Session, line []byte) {
	var probe struct {
		Type        string `json:"type"`
		CustomTitle string `json:"customTitle"`
		Cwd         string `json:"cwd"`
	}
	if err := json.Unmarshal(line, &probe); err != nil {
		return
	}
	switch probe.Type {
	case "custom-title":
		if probe.CustomTitle != "" {
			s.CustomName = probe.CustomTitle
		}
	case "user", "assistant", "system", "summary":
		if s.ProjectDir == "" && probe.Cwd != "" {
			s.ProjectDir = probe.Cwd
		}
	}
}
