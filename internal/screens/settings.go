package screens

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/rw3iss/claude-viewer/internal/events"
	"github.com/rw3iss/claude-viewer/internal/components"
	"github.com/rw3iss/claude-viewer/internal/config"
	"github.com/rw3iss/claude-viewer/internal/data"
	"github.com/rw3iss/claude-viewer/internal/keys"
	"github.com/rw3iss/claude-viewer/internal/theme"
)

// Settings lets the user enable/disable detected dirs and add custom dirs.
type Settings struct {
	repo  data.Repository
	cfg   *config.Config
	theme theme.Theme
	keys  keys.Map

	width, height int
	dirs          []data.ClaudeDir
	row           int

	addingMode bool
	input      textinput.Model
	alert      components.Alert
}

// NewSettings constructs the screen.
func NewSettings(repo data.Repository, cfg *config.Config, t theme.Theme, k keys.Map) *Settings {
	ti := textinput.New()
	ti.Placeholder = "/path/to/claude/dir"
	ti.CharLimit = 4096
	ti.Width = 60

	s := &Settings{repo: repo, cfg: cfg, theme: t, keys: k, input: ti}
	s.refresh()
	return s
}

func (s *Settings) refresh() {
	s.dirs = s.repo.Dirs()
	if s.row >= len(s.dirs) {
		s.row = 0
	}
}

func (s *Settings) Init() tea.Cmd     { return nil }
func (s *Settings) SetSize(w, h int)  { s.width, s.height = w, h; s.input.Width = w - 8 }

func (s *Settings) Update(msg tea.Msg) (Screen, tea.Cmd) {
	if s.addingMode {
		var cmd tea.Cmd
		switch m := msg.(type) {
		case tea.KeyMsg:
			switch m.String() {
			case "esc":
				s.addingMode = false
				s.input.Blur()
				s.input.SetValue("")
				return s, nil
			case "enter":
				path := strings.TrimSpace(s.input.Value())
				if path == "" {
					s.addingMode = false
					s.input.Blur()
					return s, nil
				}
				if err := s.repo.AddCustom(path); err != nil {
					s.alert = components.Alert{Text: err.Error(), Level: components.AlertErr, Expires: time.Now().Add(3 * time.Second)}
				} else {
					s.alert = components.Alert{Text: "added " + path, Level: components.AlertOK, Expires: time.Now().Add(2 * time.Second)}
					s.refresh()
				}
				s.addingMode = false
				s.input.Blur()
				s.input.SetValue("")
				return s, components.AlertCmd(time.Now().UnixNano(), 3*time.Second)
			}
		}
		s.input, cmd = s.input.Update(msg)
		return s, cmd
	}

	switch msg := msg.(type) {
	case components.AlertExpiredMsg:
		s.alert = components.Alert{}
	case tea.KeyMsg:
		switch {
		case key(s.keys.Quit, msg):
			return s, func() tea.Msg { return events.QuitAppMsg{} }
		case key(s.keys.Esc, msg):
			return s, func() tea.Msg { return events.SwitchScreenMsg{To: events.ScreenMenu} }
		case key(s.keys.Up, msg):
			if s.row > 0 {
				s.row--
			}
		case key(s.keys.Down, msg):
			if s.row < len(s.dirs)-1 {
				s.row++
			}
		case key(s.keys.Toggle, msg):
			if s.row < len(s.dirs) {
				d := s.dirs[s.row]
				if err := s.repo.SetDisabled(d.Path, !d.Disabled); err != nil {
					s.alert = components.Alert{Text: err.Error(), Level: components.AlertErr, Expires: time.Now().Add(3 * time.Second)}
				}
				s.refresh()
				return s, components.AlertCmd(time.Now().UnixNano(), 3*time.Second)
			}
		case key(s.keys.Add, msg):
			s.addingMode = true
			s.input.Focus()
			return s, textinput.Blink
		case key(s.keys.Delete, msg):
			if s.row < len(s.dirs) {
				d := s.dirs[s.row]
				if !d.Custom {
					s.alert = components.Alert{Text: "only custom dirs can be removed", Level: components.AlertWarn, Expires: time.Now().Add(2 * time.Second)}
					return s, components.AlertCmd(time.Now().UnixNano(), 2*time.Second)
				}
				if err := s.repo.RemoveCustom(d.Path); err != nil {
					s.alert = components.Alert{Text: err.Error(), Level: components.AlertErr, Expires: time.Now().Add(3 * time.Second)}
				}
				s.refresh()
				return s, components.AlertCmd(time.Now().UnixNano(), 3*time.Second)
			}
		}
	}
	return s, nil
}

func (s *Settings) View() string {
	if s.width < 20 || s.height < 8 {
		return s.theme.Dim().Render("claude-viewer: initializing…")
	}
	hint := "↑/↓ select · space toggle · n add · d remove (custom only) · esc back · q quit"
	header := components.Header(s.theme, *s.cfg, components.HeaderInput{
		Title:   "Settings — Claude Directories",
		HintRow: hint,
		Width:   s.width,
	})

	var b strings.Builder
	b.WriteString(header)
	b.WriteString("\n\n")

	for i, d := range s.dirs {
		mark := s.theme.Success().Render("[✓]")
		if d.Disabled {
			mark = s.theme.Idle().Render("[ ]")
		}
		label := s.theme.Subtitle().Render(d.Label)
		if d.Custom {
			label += " " + s.theme.Dim().Render("(custom)")
		}
		org := ""
		if d.OrgName != "" {
			org = " " + s.theme.AccentAlt().Render("@ "+d.OrgName)
		}
		path := s.theme.Dim().Render(d.Path)
		row := fmt.Sprintf("  %s  %s%s  %s", mark, label, org, path)
		if i == s.row {
			row = s.theme.Selected().Render(row)
		}
		b.WriteString(row + "\n")
	}

	if s.addingMode {
		b.WriteString("\n")
		b.WriteString(s.theme.Highlight().Render("New custom dir path:") + "\n")
		b.WriteString("  " + s.input.View() + "\n")
		b.WriteString(s.theme.Dim().Render("  enter to add · esc to cancel") + "\n")
	}

	b.WriteString("\n")
	b.WriteString(components.RenderAlert(s.theme, s.alert))
	return b.String()
}
