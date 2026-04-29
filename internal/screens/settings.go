package screens

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/key"
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

	addingMode  bool
	editingPref string // "" = not editing; "active_minutes" = editing that pref
	input       textinput.Model
	alert       components.Alert
	helpVisible bool
}

// rowCount returns the total navigable rows = dirs + 1 prefs row (active window).
func (s *Settings) rowCount() int { return len(s.dirs) + 1 }

// isPrefRow returns true if s.row points to the prefs section.
func (s *Settings) isPrefRow() bool { return s.row >= len(s.dirs) }

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

	if s.editingPref != "" {
		var cmd tea.Cmd
		switch m := msg.(type) {
		case tea.KeyMsg:
			switch m.String() {
			case "esc":
				s.editingPref = ""
				s.input.Blur()
				s.input.SetValue("")
				return s, nil
			case "enter":
				raw := strings.TrimSpace(s.input.Value())
				if raw == "" {
					s.editingPref = ""
					s.input.Blur()
					return s, nil
				}
				switch s.editingPref {
				case "active_minutes":
					n, err := strconv.Atoi(raw)
					if err != nil || n < 1 || n > 60*24*30 {
						s.alert = components.Alert{Text: "enter a number between 1 and " + strconv.Itoa(60*24*30), Level: components.AlertErr, Expires: time.Now().Add(3 * time.Second)}
						return s, components.AlertCmd(time.Now().UnixNano(), 3*time.Second)
					}
					s.cfg.ActiveMinutes = n
					if err := config.Save(s.cfg); err != nil {
						s.alert = components.Alert{Text: err.Error(), Level: components.AlertErr, Expires: time.Now().Add(3 * time.Second)}
					} else {
						s.alert = components.Alert{Text: fmt.Sprintf("active window = %d min", n), Level: components.AlertOK, Expires: time.Now().Add(2 * time.Second)}
					}
				}
				s.editingPref = ""
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
		if s.helpVisible {
			if key.Matches(msg, s.keys.Help) || key.Matches(msg, s.keys.Esc) {
				s.helpVisible = false
			}
			return s, nil
		}
		switch {
		case key.Matches(msg, s.keys.Help):
			s.helpVisible = true
			return s, nil
		case key.Matches(msg, s.keys.Quit):
			return s, func() tea.Msg { return events.QuitAppMsg{} }
		case key.Matches(msg, s.keys.Esc):
			return s, func() tea.Msg { return events.SwitchScreenMsg{To: events.ScreenMenu} }
		case key.Matches(msg, s.keys.Up):
			if s.row > 0 {
				s.row--
			}
		case key.Matches(msg, s.keys.Down):
			if s.row < s.rowCount()-1 {
				s.row++
			}
		case key.Matches(msg, s.keys.Toggle):
			if !s.isPrefRow() {
				d := s.dirs[s.row]
				if err := s.repo.SetDisabled(d.Path, !d.Disabled); err != nil {
					s.alert = components.Alert{Text: err.Error(), Level: components.AlertErr, Expires: time.Now().Add(3 * time.Second)}
				}
				s.refresh()
				return s, components.AlertCmd(time.Now().UnixNano(), 3*time.Second)
			}
		case key.Matches(msg, s.keys.Enter):
			// Enter on a pref row enters edit mode.
			if s.isPrefRow() {
				s.editingPref = "active_minutes"
				s.input.SetValue(strconv.Itoa(s.cfg.ActiveMinutes))
				s.input.Placeholder = "minutes (1..43200)"
				s.input.Focus()
				return s, textinput.Blink
			}
		case key.Matches(msg, s.keys.Add):
			s.addingMode = true
			s.input.SetValue("")
			s.input.Placeholder = "/path/to/claude/dir"
			s.input.Focus()
			return s, textinput.Blink
		case key.Matches(msg, s.keys.Delete):
			if !s.isPrefRow() {
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
	if s.helpVisible {
		return components.RenderHelp(s.theme, components.HelpInput{
			Title:    "Settings — Help",
			Sections: helpForSettings(),
			Width:    s.width, Height: s.height,
		})
	}
	hint := "↑/↓ select · space toggle · enter edit · n add · d remove · h help · esc menu · q quit"
	header := components.Header(s.theme, *s.cfg, components.HeaderInput{
		Title:   "Settings",
		HintRow: hint,
		Width:   s.width,
	})

	var b strings.Builder
	b.WriteString(header)
	b.WriteString("\n\n")

	// === Directories section ===
	b.WriteString(s.theme.Subtitle().Render("Directories"))
	b.WriteString("\n")
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

	// === Preferences section ===
	b.WriteString("\n")
	b.WriteString(s.theme.Subtitle().Render("Preferences"))
	b.WriteString("\n")

	prefRow := func(label, value, desc string, focused bool) string {
		const labelWidth = 28
		l := label + strings.Repeat(" ", max(0, labelWidth-len(label)))
		row := fmt.Sprintf("  %s  %s  %s", l, s.theme.Highlight().Render(value), s.theme.Dim().Render(desc))
		if focused {
			row = s.theme.Selected().Render(row)
		}
		return row
	}

	const activeRowIdx = 0 // index within prefs section (only one for now)
	prefValue := fmt.Sprintf("%d min", s.cfg.ActiveMinutes)
	if s.cfg.ActiveMinutes <= 0 {
		prefValue = "60 min (default)"
	}
	b.WriteString(prefRow(
		"Active session window",
		prefValue,
		"sessions modified within this window count as active",
		s.row == len(s.dirs)+activeRowIdx,
	))
	b.WriteString("\n")

	if s.addingMode {
		b.WriteString("\n")
		b.WriteString(s.theme.Highlight().Render("New custom dir path:") + "\n")
		b.WriteString("  " + s.input.View() + "\n")
		b.WriteString(s.theme.Dim().Render("  enter to add · esc to cancel") + "\n")
	}
	if s.editingPref != "" {
		b.WriteString("\n")
		var prompt string
		switch s.editingPref {
		case "active_minutes":
			prompt = "Active session window (minutes):"
		}
		b.WriteString(s.theme.Highlight().Render(prompt) + "\n")
		b.WriteString("  " + s.input.View() + "\n")
		b.WriteString(s.theme.Dim().Render("  enter to save · esc to cancel") + "\n")
	}

	b.WriteString("\n")
	b.WriteString(components.RenderAlert(s.theme, s.alert))
	return b.String()
}
