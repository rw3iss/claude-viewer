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
// of each rendered tab "block" (max of org-line and box widths). Use those
// widths for placing aligned content (usage meters) directly underneath.
func OrgTabsWithWidths(t theme.Theme, in OrgTabsInput) (string, []int) {
	if len(in.Dirs) == 0 {
		return "", nil
	}

	// First pass: compute wrapped org lines per tab + max line count so
	// every tab gets the same vertical space (shorter orgs get top-padded
	// with an empty line).
	orgLinesByTab := make([][]string, len(in.Dirs))
	maxOrgLines := 1
	for i, d := range in.Dirs {
		orgLinesByTab[i] = wrapOrgName(d.OrgName)
		if n := len(orgLinesByTab[i]); n > maxOrgLines {
			maxOrgLines = n
		}
	}
	for i := range orgLinesByTab {
		for len(orgLinesByTab[i]) < maxOrgLines {
			orgLinesByTab[i] = append([]string{""}, orgLinesByTab[i]...)
		}
	}

	tabs := make([]string, len(in.Dirs))
	widths := make([]int, len(in.Dirs))
	for i, d := range in.Dirs {
		s, w := renderOrgTab(t, d, i == in.SelectedIdx, orgLinesByTab[i])
		tabs[i] = s
		widths[i] = w
	}

	// Two leading cols + two cols between each tab (uniform 2-col margin).
	const tabSep = "  "
	parts := make([]string, 0, 2*len(tabs)+1)
	parts = append(parts, tabSep)
	for i, tab := range tabs {
		if i > 0 {
			parts = append(parts, tabSep)
		}
		parts = append(parts, tab)
	}
	return lipgloss.JoinHorizontal(lipgloss.Bottom, parts...), widths
}

// orgNameMaxLineWidth caps how wide a single org-name line can be — beyond
// this we wrap (if multi-word) or truncate.
const orgNameMaxLineWidth = 30

// orgWrapThreshold — orgs at or below this width stay on one line.
const orgWrapThreshold = 20

// wrapOrgName splits a long org name across two lines at the most balanced
// space. Returns 1 line for short or single-word names. Each line is
// truncated to orgNameMaxLineWidth (with ellipsis) as a final guard.
func wrapOrgName(name string) []string {
	name = strings.TrimSpace(name)
	if name == "" {
		return []string{"—"}
	}
	if lipgloss.Width(name) <= orgWrapThreshold {
		return []string{truncOrg(name)}
	}
	words := strings.Fields(name)
	if len(words) < 2 {
		return []string{truncOrg(name)}
	}

	// Find break point that minimizes max(line1Width, line2Width).
	bestIdx := -1
	bestMax := 1 << 30
	for i := 1; i < len(words); i++ {
		l1 := strings.Join(words[:i], " ")
		l2 := strings.Join(words[i:], " ")
		w := lipgloss.Width(l1)
		if w2 := lipgloss.Width(l2); w2 > w {
			w = w2
		}
		if w < bestMax {
			bestMax = w
			bestIdx = i
		}
	}
	if bestIdx < 0 {
		return []string{truncOrg(name)}
	}
	return []string{
		truncOrg(strings.Join(words[:bestIdx], " ")),
		truncOrg(strings.Join(words[bestIdx:], " ")),
	}
}

func truncOrg(s string) string {
	if lipgloss.Width(s) <= orgNameMaxLineWidth {
		return s
	}
	return s[:orgNameMaxLineWidth-1] + "…"
}

// JoinTabRow takes per-tab strings (e.g. usage meters) and joins them with
// the same 2-col leading + 2-col separator OrgTabs uses so they stay
// aligned beneath the tab strip. Multi-line entries are stacked.
func JoinTabRow(parts []string) string {
	const tabSep = "  "
	out := make([]string, 0, 2*len(parts)+1)
	out = append(out, tabSep)
	for i, p := range parts {
		if i > 0 {
			out = append(out, tabSep)
		}
		out = append(out, p)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, out...)
}

// renderOrgTab returns the tab block (one or more org lines + bordered box)
// and its block width — the max of all org-line widths and box width.
// Every line is centered within that block so the column is uniform.
//
// orgLines is a pre-wrapped, top-padded slice from OrgTabsWithWidths so all
// tabs in the strip share the same height regardless of org-name length.
func renderOrgTab(t theme.Theme, d data.ClaudeDir, selected bool, orgLines []string) (string, int) {
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

	// Style each org line.
	styled := make([]string, len(orgLines))
	maxOrgW := 0
	for i, line := range orgLines {
		if line == "" {
			styled[i] = ""
			continue
		}
		if selected {
			styled[i] = t.AccentAlt().Bold(true).Render(line)
		} else {
			styled[i] = t.Dim().Render(line)
		}
		if w := lipgloss.Width(styled[i]); w > maxOrgW {
			maxOrgW = w
		}
	}

	boxW := lipgloss.Width(box)
	blockW := boxW
	if maxOrgW > blockW {
		blockW = maxOrgW
	}

	// Center every org line and the box within blockW.
	center := lipgloss.NewStyle().Width(blockW).Align(lipgloss.Center)
	var b strings.Builder
	for _, l := range styled {
		b.WriteString(center.Render(l))
		b.WriteString("\n")
	}
	b.WriteString(center.Render(box))
	return b.String(), blockW
}
