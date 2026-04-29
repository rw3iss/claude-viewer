package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/rw3iss/claude-viewer/internal/data"
	"github.com/rw3iss/claude-viewer/internal/theme"
)

// Internal meter sizing: keep content readable but bounded so a wide org
// name doesn't stretch the bar across the whole tab block.
// meterContentWidth — the meter content (bar + label + pct + countdown)
// is rendered at a fixed 15-cols wide and centered inside the tab block.
const meterContentWidth = 15

// UsageMeter renders a 2-line meter (5h + 7d) sized to fit comfortably
// inside blockWidth, with side padding so it doesn't span edge-to-edge.
// Line format: `5h ████░░ 70% 4h12m`. Tightens (drops countdown, then
// shrinks bar) when content width is small.
func UsageMeter(t theme.Theme, u *data.Usage, blockWidth int) string {
	mw := meterContentWidth
	if u == nil {
		return centerLine(t.Dim().Render("…"), blockWidth) + "\n" + centerLine("", blockWidth)
	}
	five := meterLine(t, "5h", u.FiveHourPct, u.FiveHourResetAt, mw)
	seven := meterLine(t, "7d", u.SevenDayPct, u.SevenDayResetAt, mw)
	return centerLine(five, blockWidth) + "\n" + centerLine(seven, blockWidth)
}

// UsageMeterError renders a 2-line dim error block, centered in blockWidth.
func UsageMeterError(t theme.Theme, err string, blockWidth int) string {
	mw := meterContentWidth
	if len(err) > mw {
		err = err[:mw-1] + "…"
	}
	return centerLine(t.AlertWarn().Render("usage err"), blockWidth) + "\n" + centerLine(t.Dim().Render(err), blockWidth)
}

func meterLine(t theme.Theme, label string, pct int, resetAt time.Time, totalW int) string {
	if pct < 0 {
		pct = 0
	}
	if pct > 999 {
		pct = 999
	}
	pctStr := fmt.Sprintf("%d%%", pct)
	countdown := formatRemaining(resetAt)

	// Layouts in order of preference:
	//   "5h ████░ 70% 4h12m"   (full)
	//   "5h ████░ 70%"         (no countdown)
	//   "5h 70%"               (text-only, narrow)
	full := len(label) + 1 + 1 + 1 + len(pctStr) + 1 + len(countdown)
	med := len(label) + 1 + 1 + 1 + len(pctStr)
	narrow := len(label) + 1 + len(pctStr)

	switch {
	case totalW >= full+3:
		barW := totalW - len(label) - len(pctStr) - len(countdown) - 3
		bar := buildBar(t, pct, barW)
		s := fmt.Sprintf("%s %s %s %s", label, bar, t.Subtitle().Render(pctStr), t.Dim().Render(countdown))
		return centerLine(s, totalW)
	case totalW >= med+3:
		barW := totalW - len(label) - len(pctStr) - 2
		bar := buildBar(t, pct, barW)
		s := fmt.Sprintf("%s %s %s", label, bar, t.Subtitle().Render(pctStr))
		return centerLine(s, totalW)
	case totalW >= narrow+1:
		s := label + " " + t.Subtitle().Render(pctStr)
		return centerLine(s, totalW)
	default:
		return centerLine(t.Subtitle().Render(pctStr), totalW)
	}
}

func buildBar(t theme.Theme, pct, w int) string {
	if w < 1 {
		w = 1
	}
	filled := pct * w / 100
	if filled > w {
		filled = w
	}
	if filled < 0 {
		filled = 0
	}
	left := strings.Repeat("█", filled)
	right := strings.Repeat("░", w-filled)
	style := t.Success()
	switch {
	case pct >= 90:
		style = t.Error()
	case pct >= 70:
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("#e5c07b"))
	case pct >= 50:
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("#ffb055"))
	}
	return style.Render(left) + t.Dim().Render(right)
}

// formatRemaining returns a compact "Nd Mh" / "Nh Mm" / "Nm" string.
func formatRemaining(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	d := time.Until(t)
	if d <= 0 {
		return "now"
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		h := int(d.Hours())
		m := int(d.Minutes()) - h*60
		if m == 0 {
			return fmt.Sprintf("%dh", h)
		}
		return fmt.Sprintf("%dh%dm", h, m)
	}
	days := int(d.Hours()) / 24
	hrs := int(d.Hours()) - days*24
	if hrs == 0 {
		return fmt.Sprintf("%dd", days)
	}
	return fmt.Sprintf("%dd%dh", days, hrs)
}

// centerLine pads s with spaces to occupy exactly width visible cells, with
// roughly equal padding on each side.
func centerLine(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	left := (width - w) / 2
	right := width - w - left
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
}
