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
// meterContentWidth — minimum width for the meter content. Each meter
// is rendered at this width and centered inside the surrounding tab block.
const meterContentWidth = 16

// UsageMeter renders a 2-line meter sized to fit inside blockWidth. The
// label on each line is the time remaining until that period resets — so
// the user sees how much budget time is left, per account, at a glance:
//   "4h12m ▓▓▓ 5%"     ← top row = 5-hour window
//   "3d 4h ░░░ 61%"    ← bottom row = 7-day window
// When no reset-time is known yet (cache miss / first paint), the label
// falls back to "5h" / "7d" so the slot still indicates which period.
func UsageMeter(t theme.Theme, u *data.Usage, blockWidth int) string {
	mw := meterContentWidth
	if u == nil {
		return centerLine(t.Dim().Render("…"), blockWidth) + "\n" + centerLine("", blockWidth)
	}
	fiveLabel := remainingOrFallback(u.FiveHourResetAt, "5h")
	sevenLabel := remainingOrFallback(u.SevenDayResetAt, "7d")

	// Right-pad labels to the same width so the bars align between rows.
	w := len(fiveLabel)
	if len(sevenLabel) > w {
		w = len(sevenLabel)
	}
	fiveLabel = padRight(fiveLabel, w)
	sevenLabel = padRight(sevenLabel, w)

	five := meterLine(t, fiveLabel, u.FiveHourPct, mw)
	seven := meterLine(t, sevenLabel, u.SevenDayPct, mw)
	return centerLine(five, blockWidth) + "\n" + centerLine(seven, blockWidth)
}

func remainingOrFallback(reset time.Time, fallback string) string {
	r := formatRemaining(reset)
	if r == "" || r == "—" {
		return fallback
	}
	return r
}

func padRight(s string, w int) string {
	if len(s) >= w {
		return s
	}
	return s + strings.Repeat(" ", w-len(s))
}

// UsageMeterError renders a 2-line dim error block, centered in blockWidth.
// The error message is preprocessed to strip outer wrapping and keep the
// most informative tail (e.g. "fetch usage for .claude-2: usage api 401:
// Unauthorized" → "401: Unauth…"), so the user sees the actual cause.
func UsageMeterError(t theme.Theme, err string, blockWidth int) string {
	mw := meterContentWidth
	msg := stripErrPrefix(err)
	if len(msg) > mw {
		// Tail-truncate: the actual cause is usually at the end.
		msg = "…" + msg[len(msg)-mw+1:]
	}
	return centerLine(t.AlertWarn().Render("usage err"), blockWidth) + "\n" + centerLine(t.Dim().Render(msg), blockWidth)
}

// stripErrPrefix strips known wrapping prefixes ("fetch usage for X: ",
// "usage api: ") so the deepest error reaches the screen.
func stripErrPrefix(s string) string {
	for {
		i := strings.Index(s, ": ")
		if i <= 0 || i > 40 {
			return s
		}
		s = s[i+2:]
	}
}

// meterLine renders one bar like "<label> ▓▓░░ <pct>%". The label IS the
// countdown to the period reset (or "5h"/"7d" if not yet known); the
// bar takes whatever cells remain after label and percentage.
func meterLine(t theme.Theme, label string, pct int, totalW int) string {
	if pct < 0 {
		pct = 0
	}
	if pct > 999 {
		pct = 999
	}
	pctStr := fmt.Sprintf("%d%%", pct)
	full := len(label) + 1 + 1 + 1 + len(pctStr)   // label + space + bar(>=1) + space + pct
	narrow := len(label) + 1 + len(pctStr)         // text-only fallback

	switch {
	case totalW >= full+1:
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
