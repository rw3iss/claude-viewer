package screens

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
	"github.com/rw3iss/claude-viewer/internal/events"
	"github.com/rw3iss/claude-viewer/internal/components"
	"github.com/rw3iss/claude-viewer/internal/config"
	"github.com/rw3iss/claude-viewer/internal/data"
	"github.com/rw3iss/claude-viewer/internal/keys"
	"github.com/rw3iss/claude-viewer/internal/theme"
)

// Menu is the paged-by-org main session list.
type Menu struct {
	repo    data.Repository
	cfg     *config.Config
	theme   theme.Theme
	keys    keys.Map
	width   int
	height  int

	dirs      []data.ClaudeDir // enabled only
	pageIdx   int
	sessions  []data.Session // for current page
	selected  int

	usage    map[string]*data.Usage // keyed by ClaudeDir.Path
	usageErr map[string]string

	alert       components.Alert
	helpVisible bool
}

// NewMenu builds the main menu screen.
func NewMenu(repo data.Repository, cfg *config.Config, t theme.Theme, k keys.Map) *Menu {
	m := &Menu{
		repo:     repo,
		cfg:      cfg,
		theme:    t,
		keys:     k,
		usage:    map[string]*data.Usage{},
		usageErr: map[string]string{},
	}
	m.refresh()
	return m
}

// menuFetchAllUsage batches usage fetches across the menu's enabled dirs.
// Skipped (returns nil) when the feature is disabled in config.
func (m *Menu) menuFetchAllUsage(force bool) tea.Cmd {
	if !m.cfg.ShowUsageMeters {
		return nil
	}
	return fetchAllUsageCmd(m.repo, m.dirs, force)
}

func (m *Menu) refresh() {
	m.dirs = m.repo.EnabledDirs()
	if m.pageIdx >= len(m.dirs) {
		m.pageIdx = 0
	}
	if len(m.dirs) == 0 {
		m.sessions = nil
		return
	}
	sessions, err := m.repo.Sessions(m.dirs[m.pageIdx])
	if err != nil {
		m.alert = components.Alert{Text: err.Error(), Level: components.AlertErr}
	}
	m.sessions = sessions
	if m.selected >= len(m.sessions) {
		m.selected = 0
	}
}

// Init kicks off async usage fetches when meters are enabled.
func (m *Menu) Init() tea.Cmd { return m.menuFetchAllUsage(false) }

// SetSize updates dimensions.
func (m *Menu) SetSize(w, h int) { m.width, m.height = w, h }

// Update handles input.
func (m *Menu) Update(msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case UsageMsg:
		if msg.Err != nil {
			m.usageErr[msg.DirPath] = msg.Err.Error()
		} else {
			delete(m.usageErr, msg.DirPath)
			m.usage[msg.DirPath] = msg.Usage
		}
		return m, nil
	case components.AlertExpiredMsg:
		m.alert = components.Alert{}
	case tea.KeyMsg:
		// Help overlay swallows all keys except its toggle / esc.
		if m.helpVisible {
			if key.Matches(msg, m.keys.Help) || key.Matches(msg, m.keys.Esc) {
				m.helpVisible = false
			}
			return m, nil
		}
		switch {
		case key.Matches(msg, m.keys.Help):
			m.helpVisible = true
			return m, nil
		case key.Matches(msg, m.keys.Quit):
			return m, func() tea.Msg { return events.QuitAppMsg{} }
		case key.Matches(msg, m.keys.Esc):
			return m, func() tea.Msg { return events.QuitAppMsg{} }
		case key.Matches(msg, m.keys.Left):
			if len(m.dirs) > 0 {
				m.pageIdx = (m.pageIdx - 1 + len(m.dirs)) % len(m.dirs)
				m.selected = 0
				m.refresh()
			}
		case key.Matches(msg, m.keys.Right):
			if len(m.dirs) > 0 {
				m.pageIdx = (m.pageIdx + 1) % len(m.dirs)
				m.selected = 0
				m.refresh()
			}
		case key.Matches(msg, m.keys.Up):
			if m.selected > 0 {
				m.selected--
			}
		case key.Matches(msg, m.keys.Down):
			if m.selected < len(m.sessions)-1 {
				m.selected++
			}
		case key.Matches(msg, m.keys.Home):
			m.selected = 0
		case key.Matches(msg, m.keys.End):
			m.selected = len(m.sessions) - 1
		case key.Matches(msg, m.keys.Enter):
			if m.selected < len(m.sessions) {
				s := m.sessions[m.selected]
				d := m.dirs[m.pageIdx]
				return m, func() tea.Msg {
					return events.SwitchScreenMsg{To: events.ScreenChat, Session: &s, Dir: &d}
				}
			}
		case key.Matches(msg, m.keys.AllOrgs):
			return m, func() tea.Msg { return events.SwitchScreenMsg{To: events.ScreenAllOrgs} }
		case key.Matches(msg, m.keys.Settings):
			return m, func() tea.Msg { return events.SwitchScreenMsg{To: events.ScreenSettings} }
		case key.Matches(msg, m.keys.Reload):
			if len(m.dirs) > 0 {
				if _, err := m.repo.SessionsRefresh(m.dirs[m.pageIdx]); err != nil {
					m.alert = components.Alert{Text: err.Error(), Level: components.AlertErr, Expires: time.Now().Add(3 * time.Second)}
				} else {
					m.alert = components.Alert{Text: "reloaded", Level: components.AlertOK, Expires: time.Now().Add(2 * time.Second)}
				}
				m.refresh()
				return m, tea.Batch(
					components.AlertCmd(time.Now().UnixNano(), 2*time.Second),
					m.menuFetchAllUsage(true),
				)
			}
		}
	}
	return m, nil
}

// View renders the screen.
func (m *Menu) View() string {
	if m.width < 20 || m.height < 8 {
		return m.theme.Dim().Render("claude-viewer: initializing…")
	}
	if m.helpVisible {
		return components.RenderHelp(m.theme, components.HelpInput{
			Title:    "Main Menu — Help",
			Subtitle: "Browse Claude session history across orgs.",
			Sections: helpForMenu(),
			Width:    m.width, Height: m.height,
		})
	}
	hint := "←/→ org · ↑/↓ select · enter open · a all-orgs · o settings · h help · ctrl+r reload · q quit"
	header := components.Header(m.theme, *m.cfg, components.HeaderInput{
		Title:   "Claude Viewer",
		HintRow: hint,
		Width:   m.width,
	})

	if len(m.dirs) == 0 {
		return header + "\n\n" + m.theme.Dim().Render("(no Claude dirs detected — press 'o' for settings)")
	}

	tabs, tabWidths := components.OrgTabsWithWidths(m.theme, components.OrgTabsInput{
		Dirs:        m.dirs,
		SelectedIdx: m.pageIdx,
		Width:       m.width,
	})

	// Optional usage meters under each tab.
	var meterRow string
	if m.cfg.ShowUsageMeters {
		parts := make([]string, len(m.dirs))
		for i, d := range m.dirs {
			w := tabWidths[i]
			if errMsg, has := m.usageErr[d.Path]; has {
				parts[i] = components.UsageMeterError(m.theme, errMsg, w)
			} else if u := m.usage[d.Path]; u != nil {
				parts[i] = components.UsageMeter(m.theme, u, w)
			} else {
				// still loading
				parts[i] = components.UsageMeter(m.theme, nil, w)
			}
		}
		meterRow = "\n" + components.JoinTabRow(parts)
	}

	// Reserve vertical space dynamically: header(2) + spacer(1) + tabs +
	// meterRow + spacer(1) + footer(1).
	bodyHeight := m.height - 5 - lipgloss.Height(tabs)
	if meterRow != "" {
		bodyHeight -= lipgloss.Height(meterRow)
	}
	if bodyHeight < 5 {
		bodyHeight = 5
	}

	body := components.SessionList(m.theme, components.SessionListInput{
		Sessions:    m.sessions,
		SelectedIdx: m.selected,
		Width:       m.width - 2,
		Height:      bodyHeight,
		ActiveTTL:   m.cfg.ActiveDuration(),
		IsFocused:   true,
	})

	footer := components.RenderAlert(m.theme, m.alert)
	// If the focused org has a usage error, surface the full message in
	// the footer (the meter slot truncates it).
	if m.cfg.ShowUsageMeters && m.alert.Text == "" && m.pageIdx < len(m.dirs) {
		if errMsg, has := m.usageErr[m.dirs[m.pageIdx].Path]; has {
			footer = m.theme.AlertWarn().Render("usage error: ") + m.theme.Dim().Render(errMsg)
		}
	}
	return header + "\n\n" + tabs + meterRow + "\n\n" + body + "\n" + footer
}

