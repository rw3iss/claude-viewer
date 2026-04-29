package data

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ClaudeDir represents one Claude config directory (~/.claude, .claude-2,
// .claude-work, or a custom one added by the user).
type ClaudeDir struct {
	// Path is the absolute path of the config dir (e.g. /home/u/.claude-2).
	Path string
	// Label is the dirname (e.g. ".claude-2") used for display.
	Label string
	// OrgName is read from <Path>/.claude.json oauthAccount.organizationName.
	OrgName string
	// Custom is true for user-added (non-default) entries.
	Custom bool
	// Disabled flag (from config).
	Disabled bool
}

// ProjectsRoot returns the projects/ subdir, where session JSONLs live.
func (c ClaudeDir) ProjectsRoot() string {
	return filepath.Join(c.Path, "projects")
}

// IsValid checks that the path looks like a Claude config dir (has either
// .claude.json, .credentials.json, or projects/).
func (c ClaudeDir) IsValid() bool {
	for _, name := range []string{".claude.json", ".credentials.json", "projects"} {
		if _, err := os.Stat(filepath.Join(c.Path, name)); err == nil {
			return true
		}
	}
	return false
}

// DiscoverClaudeDirs walks $HOME for matching config directories.
// Pattern: ~/.claude*  (excluding obvious files like .claude.json).
func DiscoverClaudeDirs() ([]ClaudeDir, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(home)
	if err != nil {
		return nil, err
	}
	var out []ClaudeDir
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasPrefix(name, ".claude") {
			continue
		}
		p := filepath.Join(home, name)
		d := ClaudeDir{Path: p, Label: name}
		if !d.IsValid() {
			continue
		}
		d.OrgName = readOrgName(d.Path)
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Label < out[j].Label })
	return out, nil
}

// AsCustomDir builds a ClaudeDir from an arbitrary path, for user-added entries.
func AsCustomDir(path string) ClaudeDir {
	d := ClaudeDir{
		Path:   path,
		Label:  filepath.Base(path),
		Custom: true,
	}
	d.OrgName = readOrgName(path)
	return d
}

func readOrgName(p string) string {
	f, err := os.Open(filepath.Join(p, ".claude.json"))
	if err != nil {
		return ""
	}
	defer f.Close()
	var meta struct {
		OauthAccount struct {
			OrganizationName string `json:"organizationName"`
		} `json:"oauthAccount"`
	}
	if err := json.NewDecoder(f).Decode(&meta); err != nil {
		return ""
	}
	return meta.OauthAccount.OrganizationName
}

// MergeWithConfig produces the final ordered list of dirs to expose to
// the UI: auto-detected dirs (filtered by disabled list) + custom-added dirs,
// in stable order.
func MergeWithConfig(detected []ClaudeDir, customPaths, disabledPaths []string) []ClaudeDir {
	disabled := map[string]bool{}
	for _, d := range disabledPaths {
		disabled[d] = true
	}
	var out []ClaudeDir
	for _, d := range detected {
		d.Disabled = disabled[d.Path]
		out = append(out, d)
	}
	// Custom dirs that aren't already in detected
	seen := map[string]bool{}
	for _, d := range out {
		seen[d.Path] = true
	}
	for _, cp := range customPaths {
		if seen[cp] {
			continue
		}
		d := AsCustomDir(cp)
		d.Disabled = disabled[cp]
		out = append(out, d)
	}
	return out
}
