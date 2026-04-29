package components

import (
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/rw3iss/claude-viewer/internal/data"
	"github.com/rw3iss/claude-viewer/internal/theme"
)

// SessionListInput configures a single column of sessions.
type SessionListInput struct {
	Title       string
	Sessions    []data.Session
	SelectedIdx int
	Width       int
	Height      int
	ActiveTTL   time.Duration // sessions modified within = "active"
	IsFocused   bool
}

// SessionList renders one column: title + active group + idle group.
// Active sessions appear at top; idle sessions are dimmed and below.
func SessionList(t theme.Theme, in SessionListInput) string {
	if in.Width < 10 {
		in.Width = 10
	}
	titleStyle := t.Subtitle()
	if in.IsFocused {
		titleStyle = titleStyle.Bold(true)
	}
	var b strings.Builder
	b.WriteString(titleStyle.Render(in.Title))
	b.WriteString("\n")

	visible := in.Height - 2
	if visible < 1 {
		visible = 1
	}
	if len(in.Sessions) == 0 {
		b.WriteString(t.Dim().Render("(no sessions)"))
		return b.String()
	}

	rows := make([]string, 0, len(in.Sessions))
	now := time.Now()
	cutoff := in.ActiveTTL
	if cutoff == 0 {
		cutoff = 30 * time.Minute
	}
	activeCount := 0
	for _, s := range in.Sessions {
		if now.Sub(s.Mtime) <= cutoff {
			activeCount++
		}
	}

	cur := 0
	for i, s := range in.Sessions {
		row := renderRow(t, s, in.Width, i == in.SelectedIdx, now.Sub(s.Mtime) <= cutoff)
		if i == activeCount && activeCount > 0 && activeCount < len(in.Sessions) {
			bw := in.Width - 2
			if bw < 1 {
				bw = 1
			}
			rows = append(rows, t.Border().Render(strings.Repeat("─", bw)))
			cur++
		}
		rows = append(rows, row)
		cur++
	}

	// Scroll: keep selected in view
	start := 0
	if in.SelectedIdx >= visible {
		start = in.SelectedIdx - visible + 1
	}
	if start < 0 {
		start = 0
	}
	end := start + visible
	if end > len(rows) {
		end = len(rows)
	}
	b.WriteString(strings.Join(rows[start:end], "\n"))
	return b.String()
}

func renderRow(t theme.Theme, s data.Session, width int, selected, active bool) string {
	left := s.Display()
	if s.CustomName != "" {
		left = s.CustomName
	}
	right := s.ShortUUID()
	pad := width - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if pad < 1 {
		pad = 1
	}
	row := left + strings.Repeat(" ", pad) + right + " "

	switch {
	case selected:
		return t.Selected().Width(width).Render(row)
	case !active:
		return t.Idle().Render(row)
	default:
		return row
	}
}
