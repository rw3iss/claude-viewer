package screens

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/key"
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

	alert       components.Alert
	helpVisible bool
}

// NewMenu builds the main menu screen.
func NewMenu(repo data.Repository, cfg *config.Config, t theme.Theme, k keys.Map) *Menu {
	m := &Menu{repo: repo, cfg: cfg, theme: t, keys: k}
	m.refresh()
	return m
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

// Init satisfies tea.Model. No initial cmd needed.
func (m *Menu) Init() tea.Cmd { return nil }

// SetSize updates dimensions.
func (m *Menu) SetSize(w, h int) { m.width, m.height = w, h }

// Update handles input.
func (m *Menu) Update(msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {
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
				return m, components.AlertCmd(time.Now().UnixNano(), 2*time.Second)
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

	tabs := components.OrgTabs(m.theme, components.OrgTabsInput{
		Dirs:        m.dirs,
		SelectedIdx: m.pageIdx,
		Width:       m.width,
	})

	// Header is 2 lines (line + hint), blank, tabs are 4 lines (org+border+label+border),
	// blank, alert footer 1 line.
	bodyHeight := m.height - 12
	if bodyHeight < 5 {
		bodyHeight = 5
	}

	body := components.SessionList(m.theme, components.SessionListInput{
		Sessions:    m.sessions,
		SelectedIdx: m.selected,
		Width:       m.width - 2,
		Height:      bodyHeight,
		ActiveTTL:   30 * time.Minute,
		IsFocused:   true,
	})

	footer := components.RenderAlert(m.theme, m.alert)
	return header + "\n\n" + tabs + "\n\n" + body + "\n" + footer
}

