package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/rw3iss/claude-viewer/internal/data"
	"github.com/rw3iss/claude-viewer/internal/theme"
)

// OrgTabsInput configures the tab strip.
type OrgTabsInput struct {
	Dirs        []data.ClaudeDir
	SelectedIdx int
	Width       int // for overflow handling (currently informational)
}

// OrgTabs renders a horizontal strip of bordered tabs (see OrgTabsWithWidths
// for the signature that also returns per-tab pixel widths).
func OrgTabs(t theme.Theme, in OrgTabsInput) string {
	s, _ := OrgTabsWithWidths(t, in)
	return s
}

// OrgTabsWithWidths is like OrgTabs but additionally returns the visible width
// of each rendered tab — useful for placing aligned content (e.g. usage
// meters) directly underneath.
func OrgTabsWithWidths(t theme.Theme, in OrgTabsInput) (string, []int) {
	if len(in.Dirs) == 0 {
		return "", nil
	}
	tabs := make([]string, len(in.Dirs))
	widths := make([]int, len(in.Dirs))
	for i, d := range in.Dirs {
		tabs[i] = renderOrgTab(t, d, i == in.SelectedIdx)
		// Width of the box (last line of the rendered tab) — the org line
		// above is padded to match, so any line works.
		lines := strings.Split(tabs[i], "\n")
		widths[i] = lipgloss.Width(lines[len(lines)-1])
	}
	parts := make([]string, 0, 2*len(tabs)-1)
	for i, tab := range tabs {
		if i > 0 {
			parts = append(parts, "  ")
		}
		parts = append(parts, tab)
	}
	return lipgloss.JoinHorizontal(lipgloss.Bottom, parts...), widths
}

// JoinTabRow takes per-tab strings (e.g. usage meters) and joins them with
// the same 2-space separator OrgTabs uses, so they stay aligned beneath
// the tab strip. Multi-line entries are stacked correctly.
func JoinTabRow(parts []string) string {
	out := make([]string, 0, 2*len(parts)-1)
	for i, p := range parts {
		if i > 0 {
			out = append(out, "  ")
		}
		out = append(out, p)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, out...)
}

func renderOrgTab(t theme.Theme, d data.ClaudeDir, selected bool) string {
	labelText := " " + d.Label + " "
	if d.Custom {
		labelText = " " + d.Label + " ★"
	}

	var box string
	if selected {
		box = lipgloss.NewStyle().
			Border(lipgloss.ThickBorder()).
			BorderForeground(lipgloss.Color("#56b6c2")).
			Foreground(lipgloss.Color("#ffffff")).
			Bold(true).
			Render(labelText)
	} else {
		box = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#555")).
			Foreground(lipgloss.Color("#aaaaaa")).
			Render(labelText)
	}

	org := d.OrgName
	if org == "" {
		org = "—"
	}
	const orgMax = 32
	if len(org) > orgMax {
		org = org[:orgMax-1] + "…"
	}
	var orgStyled string
	if selected {
		orgStyled = t.AccentAlt().Bold(true).Render(org)
	} else {
		orgStyled = t.Dim().Render(org)
	}

	// Center the org above the box.
	boxW := lipgloss.Width(box)
	orgW := lipgloss.Width(orgStyled)
	pad := boxW - orgW
	if pad < 0 {
		// org is wider than the box — let the org overflow centered;
		// the box stays its natural width.
		pad = 0
	}
	leftPad := pad / 2
	rightPad := pad - leftPad
	orgLine := strings.Repeat(" ", leftPad) + orgStyled + strings.Repeat(" ", rightPad)

	return orgLine + "\n" + box
}
