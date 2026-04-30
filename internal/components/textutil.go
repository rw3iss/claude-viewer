package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Truncate shortens s to at most maxW visible cells, appending '…' if it
// had to cut. Prefix-truncation (the start is preserved). Use this for
// names, labels, and other "the beginning is the meaningful part" strings.
func Truncate(s string, maxW int) string {
	if maxW <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= maxW {
		return s
	}
	if maxW < 2 {
		return s[:maxW]
	}
	return s[:maxW-1] + "…"
}

// TruncatePath shortens s to at most maxW visible cells, preserving the
// TAIL — useful for filesystem paths where the trailing folder is the
// meaningful identifier:
//
//	"~/Sites/ven/other/scheduler-invoke-lambda" (41) at maxW=18
//	  →  "…er-invoke-lambda"
//
// Falls back to plain prefix-truncation when s contains no '/'.
func TruncatePath(s string, maxW int) string {
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
		// Keep the last (maxW-1) bytes + leading ellipsis.
		return "…" + s[len(s)-maxW+1:]
	}
	return s[:maxW-1] + "…"
}

// TruncateAnsi shortens a possibly-styled string to at most maxW visible
// cells, appending '…'. Walks down the byte length so that lipgloss.Width
// (which is ANSI-aware) lands in budget; the trailing escape sequences
// remain intact when they were leading-only.
func TruncateAnsi(s string, maxW int) string {
	if maxW <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= maxW {
		return s
	}
	for i := len(s); i > 0; i-- {
		cand := s[:i]
		if lipgloss.Width(cand) <= maxW-1 {
			return cand + "…"
		}
	}
	return ""
}
