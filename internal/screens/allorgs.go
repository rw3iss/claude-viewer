package screens

import (
	"strings"

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

// truncateAnsi delegates to the shared ANSI-aware truncation helper.
// Kept as a local alias so the call sites in this file read naturally.
func truncateAnsi(s string, maxW int) string { return components.TruncateAnsi(s, maxW) }

// AllOrgs renders every enabled ClaudeDir as a column on one screen.
type AllOrgs struct {
	repo   data.Repository
	cfg    *config.Config
	theme  theme.Theme
	keys   keys.Map
	width  int
	height int

	dirs   []data.ClaudeDir
	cols   [][]data.Session // sessions per dir
	colIdx int              // focused column
	rowIdx []int            // selected row per column

	usage usageState // 5h/7d meter data + per-dir error tracking

	alert       components.Alert
	helpVisible bool
}

// NewAllOrgs builds the multi-column screen.
func NewAllOrgs(repo data.Repository, cfg *config.Config, t theme.Theme, k keys.Map) *AllOrgs {
	a := &AllOrgs{
		repo:  repo,
		cfg:   cfg,
		theme: t,
		keys:  k,
		usage: newUsageState(),
	}
	a.refresh()
	return a
}

// allOrgsFetchUsage batches usage fetches across enabled dirs (skipped when
// the meters setting is off).
func (a *AllOrgs) allOrgsFetchUsage(force bool) tea.Cmd {
	return usageFetchCmd(a.cfg.ShowUsageMeters, a.repo, a.dirs, force)
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

func (a *AllOrgs) Init() tea.Cmd    { return a.allOrgsFetchUsage(false) }
func (a *AllOrgs) SetSize(w, h int) { a.width, a.height = w, h }

func (a *AllOrgs) Update(msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case UsageMsg:
		a.usage.Apply(msg)
		return a, nil
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
		case key.Matches(msg, a.keys.AllOrgs):
			// 'a' toggles back to the regular per-org menu.
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
			return a, a.allOrgsFetchUsage(true)
		}
	}
	return a, nil
}

func (a *AllOrgs) View() string {
	if a.width < 20 || a.height < 8 {
		return components.LoadingPlaceholder(a.theme)
	}
	if a.helpVisible {
		return components.RenderHelp(a.theme, components.HelpInput{
			Title:    "All Organizations — Help",
			Sections: helpForAllOrgs(),
			Width:    a.width, Height: a.height,
		})
	}
	hint := "←/→ column · ↑/↓ row · enter open · a / esc back · r reload · h help"
	header := components.Header(a.theme, *a.cfg, components.HeaderInput{
		Title:   "All Organizations",
		HintRow: hint,
		Width:   a.width,
	})
	if len(a.dirs) == 0 {
		return header + "\n\n" + a.theme.Dim().Render("(no enabled dirs)")
	}

	colW := (a.width - len(a.dirs) - 1) / len(a.dirs)
	colW = max(colW, 20)
	bodyH := a.height - 5
	// Reserve 3 rows when meters are enabled (2 lines + 1 spacer per column).
	if a.cfg.ShowUsageMeters {
		bodyH -= 3
	}
	bodyH = max(bodyH, 5)

	cols := make([]string, len(a.dirs))
	for i, d := range a.dirs {
		// Build a styled, colW-truncated 1- or 2-line header so each
		// column's widest line stays at colW — otherwise JoinHorizontal
		// pads the column to the title's width and breaks alignment.
		labelStyle := a.theme.Subtitle()
		if i == a.colIdx {
			labelStyle = labelStyle.Bold(true)
		}
		title := truncateAnsi(labelStyle.Render(d.Label), colW)
		if d.OrgName != "" {
			org := "@ " + d.OrgName
			if lipgloss.Width(org) > colW {
				org = org[:colW-1] + "…"
			}
			title += "\n" + a.theme.AccentAlt().Render(org)
		}

		// Optional usage meter centered above the session list.
		if a.cfg.ShowUsageMeters {
			var meter string
			if errMsg := a.usage.Err(d.Path); errMsg != "" {
				meter = components.UsageMeterError(a.theme, errMsg, colW)
			} else {
				meter = components.UsageMeter(a.theme, a.usage.Get(d.Path), colW)
			}
			title += "\n" + meter
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
		// pad height with empty rows
		for len(splits[i]) < height {
			splits[i] = append(splits[i], "")
		}
		// pad each row to exactly colW visible cells so the column
		// divider lands at the same position on every row.
		for r, line := range splits[i] {
			if w := lipgloss.Width(line); w < colW {
				splits[i][r] = line + strings.Repeat(" ", colW-w)
			}
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
