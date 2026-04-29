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

// SessionList renders one column: title + running group + everything else.
//
// Sessions where a live process holds the JSONL open (Session.Running == true)
// appear at the top with a green ● marker. Everything else follows below a
// divider, dimmed if the mtime is older than ActiveTTL.
func SessionList(t theme.Theme, in SessionListInput) string {
	if in.Width < 10 {
		in.Width = 10
	}
	var b strings.Builder
	if in.Title != "" {
		titleStyle := t.Subtitle()
		if in.IsFocused {
			titleStyle = titleStyle.Bold(true)
		}
		b.WriteString(titleStyle.Render(in.Title))
		b.WriteString("\n")
	}

	visible := in.Height - 2
	if visible < 1 {
		visible = 1
	}
	if len(in.Sessions) == 0 {
		b.WriteString(t.Dim().Render("(no sessions)"))
		return b.String()
	}

	// Reorder: running first (preserving relative mtime order), then the
	// rest. Then mark divider position.
	now := time.Now()
	cutoff := in.ActiveTTL
	if cutoff == 0 {
		cutoff = 30 * time.Minute
	}

	type entry struct {
		s    data.Session
		idx  int // original index for SelectedIdx tracking
		fade bool
	}
	var running, rest []entry
	for i, s := range in.Sessions {
		fade := now.Sub(s.Mtime) > cutoff && !s.Running
		e := entry{s: s, idx: i, fade: fade}
		if s.Running {
			running = append(running, e)
		} else {
			rest = append(rest, e)
		}
	}

	rows := make([]string, 0, len(in.Sessions)+1)
	for _, e := range running {
		rows = append(rows, renderRow(t, e.s, in.Width, e.idx == in.SelectedIdx, e.fade))
	}
	if len(running) > 0 && len(rest) > 0 {
		bw := in.Width - 2
		if bw < 1 {
			bw = 1
		}
		rows = append(rows, t.Border().Render(strings.Repeat("─", bw)))
	}
	for _, e := range rest {
		rows = append(rows, renderRow(t, e.s, in.Width, e.idx == in.SelectedIdx, e.fade))
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

func renderRow(t theme.Theme, s data.Session, width int, selected, fade bool) string {
	// Leading marker column: green ● when a live process holds the file
	// open, two spaces otherwise. Always 2 cells wide so columns align.
	marker := "  "
	if s.Running {
		marker = t.Active().Render("●") + " "
	}

	left := s.Display()
	if s.CustomName != "" {
		left = s.CustomName
	}
	right := s.ShortUUID()
	pad := width - 2 - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if pad < 1 {
		pad = 1
	}
	row := marker + left + strings.Repeat(" ", pad) + right + " "

	switch {
	case selected:
		return t.Selected().Width(width).Render(row)
	case fade:
		return t.Idle().Render(row)
	default:
		return row
	}
}
