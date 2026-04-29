package screens

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
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

	alert components.Alert
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
		switch {
		case key(m.keys.Quit, msg):
			return m, func() tea.Msg { return events.QuitAppMsg{} }
		case key(m.keys.Esc, msg):
			return m, func() tea.Msg { return events.QuitAppMsg{} }
		case key(m.keys.Left, msg):
			if len(m.dirs) > 0 {
				m.pageIdx = (m.pageIdx - 1 + len(m.dirs)) % len(m.dirs)
				m.selected = 0
				m.refresh()
			}
		case key(m.keys.Right, msg):
			if len(m.dirs) > 0 {
				m.pageIdx = (m.pageIdx + 1) % len(m.dirs)
				m.selected = 0
				m.refresh()
			}
		case key(m.keys.Up, msg):
			if m.selected > 0 {
				m.selected--
			}
		case key(m.keys.Down, msg):
			if m.selected < len(m.sessions)-1 {
				m.selected++
			}
		case key(m.keys.Home, msg):
			m.selected = 0
		case key(m.keys.End, msg):
			m.selected = len(m.sessions) - 1
		case key(m.keys.Enter, msg):
			if m.selected < len(m.sessions) {
				s := m.sessions[m.selected]
				d := m.dirs[m.pageIdx]
				return m, func() tea.Msg {
					return events.SwitchScreenMsg{To: events.ScreenChat, Session: &s, Dir: &d}
				}
			}
		case key(m.keys.AllOrgs, msg):
			return m, func() tea.Msg { return events.SwitchScreenMsg{To: events.ScreenAllOrgs} }
		case key(m.keys.Settings, msg):
			return m, func() tea.Msg { return events.SwitchScreenMsg{To: events.ScreenSettings} }
		case key(m.keys.Reload, msg):
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
	hint := fmt.Sprintf("←/→ page · ↑/↓ select · enter open · a all-orgs · o settings · ctrl+r reload · q quit")
	var dirRef *data.ClaudeDir
	if m.pageIdx < len(m.dirs) {
		d := m.dirs[m.pageIdx]
		dirRef = &d
	}
	header := components.Header(m.theme, *m.cfg, components.HeaderInput{
		Title:   "Claude Viewer",
		Dir:     dirRef,
		HintRow: hint,
		Width:   m.width,
	})

	bodyHeight := m.height - 5
	if bodyHeight < 5 {
		bodyHeight = 5
	}

	title := "(no Claude dirs detected — press 'o' for settings)"
	body := ""
	if len(m.dirs) > 0 {
		d := m.dirs[m.pageIdx]
		title = fmt.Sprintf("Page %d/%d  ·  %s", m.pageIdx+1, len(m.dirs), m.theme.Subtitle().Render(d.Label))
		if d.OrgName != "" {
			title += "  " + m.theme.AccentAlt().Render("@ "+d.OrgName)
		}
		body = components.SessionList(m.theme, components.SessionListInput{
			Title:       title,
			Sessions:    m.sessions,
			SelectedIdx: m.selected,
			Width:       m.width - 2,
			Height:      bodyHeight,
			ActiveTTL:   30 * time.Minute,
			IsFocused:   true,
		})
	}

	footer := components.RenderAlert(m.theme, m.alert)
	return header + "\n\n" + body + "\n" + footer
}

// helper: matches key.Binding against a tea.KeyMsg
func key(b interface{ Keys() []string }, msg tea.KeyMsg) bool {
	s := msg.String()
	for _, k := range b.Keys() {
		if k == s {
			return true
		}
	}
	return false
}
