package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/rw3iss/claude-viewer/internal/theme"
)

// Footer renders a dim status line. Hints is left-aligned, status is right-aligned.
func Footer(t theme.Theme, hints, status string, width int) string {
	hStyled := t.Dim().Render(hints)
	sStyled := status // status may have its own color
	pad := width - lipgloss.Width(hStyled) - lipgloss.Width(sStyled)
	pad = max(pad, 1)
	return hStyled + strings.Repeat(" ", pad) + sStyled
}
