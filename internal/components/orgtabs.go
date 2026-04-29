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

// OrgTabs renders a horizontal strip of bordered tabs, one per ClaudeDir.
// Each tab shows the dir label inside a rounded (or thick, when selected)
// border, with the org name centered on the line above the box.
//
//	rw3iss@gmail.com's Org      rw3iss@gmail.com's Org           Vendidit
//	╭─ .claude ─╮               ┏━━ .claude-2 ━━┓               ╭─ .claude-work ─╮
//
// Caller is responsible for placing the strip and providing height.
func OrgTabs(t theme.Theme, in OrgTabsInput) string {
	if len(in.Dirs) == 0 {
		return ""
	}

	tabs := make([]string, len(in.Dirs))
	for i, d := range in.Dirs {
		tabs[i] = renderOrgTab(t, d, i == in.SelectedIdx)
	}

	// Interleave a 2-space separator between tabs.
	parts := make([]string, 0, 2*len(tabs)-1)
	for i, tab := range tabs {
		if i > 0 {
			parts = append(parts, "  ")
		}
		parts = append(parts, tab)
	}
	return lipgloss.JoinHorizontal(lipgloss.Bottom, parts...)
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
