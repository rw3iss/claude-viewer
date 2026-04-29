package data

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
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

// ScanSessionMeta reads the JSONL to fill in CustomName, ProjectDir, etc.
// Stops early once both are found (or end of file).
func ScanSessionMeta(s *Session) {
	f, err := os.Open(s.Path)
	if err != nil {
		return
	}
	defer f.Close()

	br := bufio.NewReaderSize(f, 64*1024)
	lines := 0
	for {
		line, err := br.ReadBytes('\n')
		if len(line) > 0 {
			lines++
			handleSessionMetaLine(s, line)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}
	}
	s.LineCount = lines
	if s.ProjectDir == "" {
		// Slug-derived fallback
		s.ProjectDir = strings.ReplaceAll(s.Slug, "-", "/")
	}
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
