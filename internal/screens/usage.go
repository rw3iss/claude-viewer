package screens

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/rw3iss/claude-viewer/internal/data"
)

// UsageMsg is delivered when an async usage fetch completes. Both the
// menu and all-orgs screens consume the same message type so the same
// fetch helpers can drive either.
type UsageMsg struct {
	DirPath string
	Usage   *data.Usage
	Err     error
}

// fetchUsageCmd builds a tea.Cmd for one dir. force=true bypasses the
// disk cache.
func fetchUsageCmd(repo data.Repository, d data.ClaudeDir, force bool) tea.Cmd {
	return func() tea.Msg {
		var u *data.Usage
		var err error
		if force {
			u, err = repo.UsageRefresh(d)
		} else {
			u, err = repo.Usage(d)
		}
		out := UsageMsg{DirPath: d.Path, Usage: u}
		if err != nil {
			out.Err = err
		}
		return out
	}
}

// fetchAllUsageCmd batches per-dir fetches. Returns nil when there's
// nothing to fetch (no dirs, or feature disabled at the call site).
func fetchAllUsageCmd(repo data.Repository, dirs []data.ClaudeDir, force bool) tea.Cmd {
	if len(dirs) == 0 {
		return nil
	}
	cmds := make([]tea.Cmd, 0, len(dirs))
	for _, d := range dirs {
		cmds = append(cmds, fetchUsageCmd(repo, d, force))
	}
	return tea.Batch(cmds...)
}

// usageState bundles the per-screen state that Menu and AllOrgs both
// maintain to track usage data + per-dir errors. Each screen embeds it.
type usageState struct {
	usage    map[string]*data.Usage
	usageErr map[string]string
}

// newUsageState builds an empty state.
func newUsageState() usageState {
	return usageState{
		usage:    map[string]*data.Usage{},
		usageErr: map[string]string{},
	}
}

// Apply records a UsageMsg into the state, clearing any previous error
// when the new fetch succeeded.
func (u *usageState) Apply(msg UsageMsg) {
	if msg.Err != nil {
		u.usageErr[msg.DirPath] = msg.Err.Error()
		return
	}
	delete(u.usageErr, msg.DirPath)
	u.usage[msg.DirPath] = msg.Usage
}

// Err returns the cached error message for dirPath, or "" if none.
func (u usageState) Err(dirPath string) string { return u.usageErr[dirPath] }

// Get returns the cached usage for dirPath, or nil if not loaded yet.
func (u usageState) Get(dirPath string) *data.Usage { return u.usage[dirPath] }

// FetchCmd returns a tea.Cmd that batches fetches for every dir, gated
// on the ShowUsageMeters config flag. Returns nil if disabled.
func usageFetchCmd(enabled bool, repo data.Repository, dirs []data.ClaudeDir, force bool) tea.Cmd {
	if !enabled {
		return nil
	}
	return fetchAllUsageCmd(repo, dirs, force)
}
