package screens

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/key"
	"github.com/rw3iss/claude-viewer/internal/events"
	"github.com/rw3iss/claude-viewer/internal/components"
	"github.com/rw3iss/claude-viewer/internal/config"
	"github.com/rw3iss/claude-viewer/internal/data"
	"github.com/rw3iss/claude-viewer/internal/keys"
	"github.com/rw3iss/claude-viewer/internal/theme"
)

// AllOrgs renders every enabled ClaudeDir as a column on one screen.
type AllOrgs struct {
	repo   data.Repository
	cfg    *config.Config
	theme  theme.Theme
	keys   keys.Map
	width  int
	height int

	dirs    []data.ClaudeDir
	cols    [][]data.Session // sessions per dir
	colIdx  int              // focused column
	rowIdx  []int            // selected row per column

	alert       components.Alert
	helpVisible bool
}

// NewAllOrgs builds the multi-column screen.
func NewAllOrgs(repo data.Repository, cfg *config.Config, t theme.Theme, k keys.Map) *AllOrgs {
	a := &AllOrgs{repo: repo, cfg: cfg, theme: t, keys: k}
	a.refresh()
	return a
}

func (a *AllOrgs) refresh() {
	a.dirs = a.repo.EnabledDirs()
	a.cols = make([][]data.Session, len(a.dirs))
	a.rowIdx = make([]int, len(a.dirs))
	for i, d := range a.dirs {
		s, _ := a.repo.Sessions(d)
		a.cols[i] = s
	}
	if a.colIdx >= len(a.dirs) {
		a.colIdx = 0
	}
}

func (a *AllOrgs) Init() tea.Cmd        { return nil }
func (a *AllOrgs) SetSize(w, h int)     { a.width, a.height = w, h }

func (a *AllOrgs) Update(msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if a.helpVisible {
			if key.Matches(msg, a.keys.Help) || key.Matches(msg, a.keys.Esc) {
				a.helpVisible = false
			}
			return a, nil
		}
		switch {
		case key.Matches(msg, a.keys.Help):
			a.helpVisible = true
			return a, nil
		case key.Matches(msg, a.keys.Esc):
			return a, func() tea.Msg { return events.SwitchScreenMsg{To: events.ScreenMenu} }
		case key.Matches(msg, a.keys.Quit):
			return a, func() tea.Msg { return events.QuitAppMsg{} }
		case key.Matches(msg, a.keys.Left):
			if a.colIdx > 0 {
				a.colIdx--
			}
		case key.Matches(msg, a.keys.Right):
			if a.colIdx < len(a.dirs)-1 {
				a.colIdx++
			}
		case key.Matches(msg, a.keys.Up):
			if a.rowIdx[a.colIdx] > 0 {
				a.rowIdx[a.colIdx]--
			}
		case key.Matches(msg, a.keys.Down):
			if a.rowIdx[a.colIdx] < len(a.cols[a.colIdx])-1 {
				a.rowIdx[a.colIdx]++
			}
		case key.Matches(msg, a.keys.Enter):
			if a.colIdx < len(a.cols) && a.rowIdx[a.colIdx] < len(a.cols[a.colIdx]) {
				s := a.cols[a.colIdx][a.rowIdx[a.colIdx]]
				d := a.dirs[a.colIdx]
				return a, func() tea.Msg {
					return events.SwitchScreenMsg{To: events.ScreenChat, Session: &s, Dir: &d}
				}
			}
		case key.Matches(msg, a.keys.Reload):
			a.refresh()
		}
	}
	return a, nil
}

func (a *AllOrgs) View() string {
	if a.width < 20 || a.height < 8 {
		return a.theme.Dim().Render("claude-viewer: initializing…")
	}
	if a.helpVisible {
		return components.RenderHelp(a.theme, components.HelpInput{
			Title:    "All Organizations — Help",
			Sections: helpForAllOrgs(),
			Width:    a.width, Height: a.height,
		})
	}
	hint := "←/→ column · ↑/↓ row · enter open · h help · esc back · ctrl+r reload"
	header := components.Header(a.theme, *a.cfg, components.HeaderInput{
		Title:   "All Organizations",
		HintRow: hint,
		Width:   a.width,
	})
	if len(a.dirs) == 0 {
		return header + "\n\n" + a.theme.Dim().Render("(no enabled dirs)")
	}

	colW := (a.width - len(a.dirs) - 1) / len(a.dirs)
	if colW < 20 {
		colW = 20
	}
	bodyH := a.height - 5
	if bodyH < 5 {
		bodyH = 5
	}

	cols := make([]string, len(a.dirs))
	for i, d := range a.dirs {
		title := d.Label
		if d.OrgName != "" {
			title += "  " + a.theme.AccentAlt().Render("@ "+d.OrgName)
		}
		cols[i] = components.SessionList(a.theme, components.SessionListInput{
			Title:       title,
			Sessions:    a.cols[i],
			SelectedIdx: a.rowIdx[i],
			Width:       colW,
			Height:      bodyH,
			ActiveTTL:   a.cfg.ActiveDuration(),
			IsFocused:   i == a.colIdx,
		})
	}

	// Pad each col to same height, then horizontally join.
	rendered := joinHorizontal(cols, bodyH, colW, a.theme)
	return header + "\n\n" + rendered + "\n" + components.RenderAlert(a.theme, a.alert)
}

func joinHorizontal(cols []string, height, colW int, t theme.Theme) string {
	splits := make([][]string, len(cols))
	for i, c := range cols {
		splits[i] = strings.Split(c, "\n")
		// pad to height
		for len(splits[i]) < height {
			splits[i] = append(splits[i], strings.Repeat(" ", colW))
		}
	}
	var lines []string
	for r := range height {
		var parts []string
		for i := range cols {
			parts = append(parts, splits[i][r])
		}
		lines = append(lines, strings.Join(parts, " "+t.Border().Render("│")+" "))
	}
	return strings.Join(lines, "\n")
}
