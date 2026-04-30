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
