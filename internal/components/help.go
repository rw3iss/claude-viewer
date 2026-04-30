package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/rw3iss/claude-viewer/internal/theme"
)

// HelpEntry is one (key, description) pair.
type HelpEntry struct {
	Key, Desc string
}

// HelpInput configures a centered help modal.
type HelpInput struct {
	Title    string
	Subtitle string
	Sections []HelpSection
	Width    int
	Height   int
}

// HelpSection is a labeled group of HelpEntries.
type HelpSection struct {
	Title   string
	Entries []HelpEntry
}

// RenderHelp returns a centered, bordered help panel sized to fit Width/Height.
func RenderHelp(t theme.Theme, in HelpInput) string {
	var body strings.Builder
	if in.Subtitle != "" {
		body.WriteString(t.Dim().Render(in.Subtitle))
		body.WriteString("\n\n")
	}
	for i, sec := range in.Sections {
		if i > 0 {
			body.WriteString("\n")
		}
		if sec.Title != "" {
			body.WriteString(t.Subtitle().Render(sec.Title))
			body.WriteString("\n")
		}
		// Compute key column width
		maxKey := 0
		for _, e := range sec.Entries {
			if l := lipgloss.Width(e.Key); l > maxKey {
				maxKey = l
			}
		}
		for _, e := range sec.Entries {
			body.WriteString("  ")
			body.WriteString(t.Highlight().Render(e.Key))
			body.WriteString(strings.Repeat(" ", maxKey-lipgloss.Width(e.Key)+2))
			body.WriteString(t.Dim().Render(e.Desc))
			body.WriteString("\n")
		}
	}
	body.WriteString("\n")
	body.WriteString(t.Dim().Render("press h or esc to close"))

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.BorderSubtle().GetForeground()).
		Padding(1, 2)
	titleLine := t.Title().Render(in.Title)
	rendered := box.Render(titleLine + "\n\n" + body.String())

	w := lipgloss.Width(rendered)
	h := lipgloss.Height(rendered)
	xPad := max(0, (in.Width-w)/2)
	yPad := max(0, (in.Height-h)/2)

	var out strings.Builder
	for range yPad {
		out.WriteString(strings.Repeat(" ", in.Width) + "\n")
	}
	pad := strings.Repeat(" ", xPad)
	for line := range strings.SplitSeq(rendered, "\n") {
		out.WriteString(pad + line + "\n")
	}
	return out.String()
}
