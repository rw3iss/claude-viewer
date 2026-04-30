package components

import (
	"strconv"
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
	in.Width = max(in.Width, 10)
	var b strings.Builder
	if in.Title != "" {
		// Title is rendered as-is so callers can pre-style multiple parts
		// (label, subtitle, etc) and we don't risk overflowing colW with
		// auto-styling that hides truncation.
		b.WriteString(in.Title)
		b.WriteString("\n")
	}

	visible := in.Height - 2
	visible = max(visible, 1)
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
		isCursor := in.IsFocused && e.idx == in.SelectedIdx
		rows = append(rows, renderRow(t, e.s, in.Width, isCursor, e.fade))
	}
	if len(running) > 0 && len(rest) > 0 {
		bw := in.Width - 2
		bw = max(bw, 1)
		rows = append(rows, t.Border().Render(strings.Repeat("─", bw)))
	}
	for _, e := range rest {
		isCursor := in.IsFocused && e.idx == in.SelectedIdx
		rows = append(rows, renderRow(t, e.s, in.Width, isCursor, e.fade))
	}

	// Scroll: keep selected in view
	start := 0
	if in.SelectedIdx >= visible {
		start = in.SelectedIdx - visible + 1
	}
	start = max(start, 0)
	end := start + visible
	end = min(end, len(rows))
	b.WriteString(strings.Join(rows[start:end], "\n"))
	return b.String()
}

// lastActiveColWidth is the fixed reservation for the "- 7h 12m ago (Apr 2 14:23)"
// column. Sized to fit the worst case ("- 99d 23h ago (Jan 22 23:59)" ≈ 28 cols).
const lastActiveColWidth = 28

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
	la := formatLastActive(s.Mtime)

	// Width budget per row (no trailing space — joinHorizontal in the
	// all-orgs view already pads short rows to colW and the divider's
	// leading space provides visual separation from the uuid).
	//   marker(2) + left + pad + lastActive(28) + "  "(2) + uuid
	leftAvail := width - 2 - lastActiveColWidth - 2 - lipgloss.Width(right)

	var row string
	if leftAvail < 10 {
		// Terminal too narrow — drop the last-active column.
		//   marker(2) + leftW + pad(>=1) + uuid(8) = width
		//   → leftW + pad = width - 10  → max leftW = width - 11.
		leftAvail2 := width - 11
		leftAvail2 = max(leftAvail2, 1)
		left = smartTruncate(left, leftAvail2)
		pad := width - 10 - lipgloss.Width(left)
		pad = max(pad, 1)
		row = marker + left + strings.Repeat(" ", pad) + right
	} else {
		left = smartTruncate(left, leftAvail)
		leftPad := leftAvail - lipgloss.Width(left)
		leftPad = max(leftPad, 0)
		laPad := lastActiveColWidth - lipgloss.Width(la)
		laPad = max(laPad, 0)
		row = marker + left + strings.Repeat(" ", leftPad) +
			t.Dim().Render(strings.Repeat(" ", laPad)+la) +
			"  " + right
	}

	switch {
	case selected:
		return t.Selected().Width(width).Render(row)
	case fade:
		return t.Idle().Render(row)
	default:
		return row
	}
}

// smartTruncate shortens s to maxW visible cells. For path-like strings
// (containing "/") it keeps the tail visible — e.g.
// "~/Sites/ven/other/scheduler-invoke-lambda" → "…r-invoke-lambda" — since
// the trailing folder is the meaningful identifier. Non-path strings get
// the usual prefix-truncation with ellipsis.
func smartTruncate(s string, maxW int) string {
	if maxW <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= maxW {
		return s
	}
	if maxW < 2 {
		return s[:maxW]
	}
	if strings.Contains(s, "/") {
		// Keep the last (maxW-1) bytes of s + leading ellipsis.
		return "…" + s[len(s)-maxW+1:]
	}
	return s[:maxW-1] + "…"
}

// formatLastActive renders a session's mtime as a relative string with the
// absolute timestamp in parens, e.g. "- 7h 12m ago (Apr 29 18:30)".
func formatLastActive(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return "- " + relativeAgo(t) + " (" + t.Format("Jan 2 15:04") + ")"
}

func relativeAgo(t time.Time) string {
	d := time.Since(t)
	d = max(d, 0)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return formatRelMinutes(int(d.Minutes()))
	case d < 24*time.Hour:
		h := int(d.Hours())
		m := int(d.Minutes()) - h*60
		if m == 0 {
			return formatRelHours(h)
		}
		return formatRelHours(h) + " " + formatRelMinutes(m)
	default:
		days := int(d.Hours()) / 24
		h := int(d.Hours()) - days*24
		if h == 0 {
			return formatRelDays(days)
		}
		return formatRelDays(days) + " " + formatRelHours(h)
	}
}

func formatRelMinutes(m int) string { return fmtItoa(m) + "m ago" }
func formatRelHours(h int) string   { return fmtItoa(h) + "h" }
func formatRelDays(d int) string    { return fmtItoa(d) + "d" }
func fmtItoa(n int) string {
	n = max(n, 0)
	return strconv.Itoa(n)
}
