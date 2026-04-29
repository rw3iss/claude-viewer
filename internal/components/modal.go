package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/rw3iss/claude-viewer/internal/theme"
)

// Modal renders a centered bordered box with title + body, sized to width/height.
func Modal(t theme.Theme, title, body string, width, height int) string {
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#666666")).
		Padding(1, 2)

	titleLine := t.Title().Render(title)
	content := titleLine + "\n\n" + body
	rendered := box.Render(content)

	w := lipgloss.Width(rendered)
	h := lipgloss.Height(rendered)
	xPad := (width - w) / 2
	yPad := (height - h) / 2
	if xPad < 0 {
		xPad = 0
	}
	if yPad < 0 {
		yPad = 0
	}

	var b strings.Builder
	for range yPad {
		b.WriteString(strings.Repeat(" ", width) + "\n")
	}
	pad := strings.Repeat(" ", xPad)
	for _, line := range strings.Split(rendered, "\n") {
		b.WriteString(pad + line + "\n")
	}
	return b.String()
}
