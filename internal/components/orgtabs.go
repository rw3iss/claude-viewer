package components

import (
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
// of each rendered tab "block" (max of org-line and box widths). Use those
// widths for placing aligned content (usage meters) directly underneath.
func OrgTabsWithWidths(t theme.Theme, in OrgTabsInput) (string, []int) {
	if len(in.Dirs) == 0 {
		return "", nil
	}
	tabs := make([]string, len(in.Dirs))
	widths := make([]int, len(in.Dirs))
	for i, d := range in.Dirs {
		s, w := renderOrgTab(t, d, i == in.SelectedIdx)
		tabs[i] = s
		widths[i] = w
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

// renderOrgTab returns the tab block (org line + bordered box) and its block
// width — the max of the org-line and box visible widths. Both lines are
// centered within that block so the column is uniform end-to-end.
func renderOrgTab(t theme.Theme, d data.ClaudeDir, selected bool) (string, int) {
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
	const orgMax = 40
	if len(org) > orgMax {
		org = org[:orgMax-1] + "…"
	}
	var orgStyled string
	if selected {
		orgStyled = t.AccentAlt().Bold(true).Render(org)
	} else {
		orgStyled = t.Dim().Render(org)
	}

	boxW := lipgloss.Width(box)
	orgW := lipgloss.Width(orgStyled)
	blockW := boxW
	if orgW > blockW {
		blockW = orgW
	}

	// Center both the org line and the (multi-line) box within blockW so
	// the entire column is exactly blockW wide. Without this the box
	// hangs left when the org overflows, and meters underneath go crooked.
	center := lipgloss.NewStyle().Width(blockW).Align(lipgloss.Center)
	return center.Render(orgStyled) + "\n" + center.Render(box), blockW
}
