// Package components holds reusable UI fragments. Each component is a
// pure function (or small struct) that takes its inputs + a Theme and
// returns a styled string.
package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/rw3iss/claude-viewer/internal/config"
	"github.com/rw3iss/claude-viewer/internal/data"
	"github.com/rw3iss/claude-viewer/internal/theme"
)

// HeaderInput is everything the header may want to display.
type HeaderInput struct {
	Title   string // primary screen title (optional, for menu/settings)
	Session *data.Session
	Dir     *data.ClaudeDir
	HintRow string // second line, dim
	Width   int
}

// Header renders a two-line styled header that fills Width columns.
func Header(t theme.Theme, cfg config.Config, in HeaderInput) string {
	left := buildLeft(t, cfg, in)
	right := buildRight(t, cfg, in)

	w := in.Width
	if w < 20 {
		w = 20
	}
	pad := w - lipgloss.Width(left) - lipgloss.Width(right)
	if pad < 2 {
		pad = 2
	}

	line1 := left + strings.Repeat(" ", pad) + right
	if in.HintRow == "" {
		return line1
	}
	return line1 + "\n" + t.Dim().Render(in.HintRow)
}

func buildLeft(t theme.Theme, cfg config.Config, in HeaderInput) string {
	var parts []string
	if in.Title != "" {
		parts = append(parts, t.Title().Render(in.Title))
	}
	if s := in.Session; s != nil {
		if cfg.HeaderShowName && s.CustomName != "" {
			parts = append(parts, t.Highlight().Render(s.CustomName))
		}
		if cfg.HeaderShowDir {
			if path := s.ProjectPath(); path != "" {
				parts = append(parts, t.Accent().Render(path))
			}
		}
	}
	return strings.Join(parts, "  ")
}

func buildRight(t theme.Theme, cfg config.Config, in HeaderInput) string {
	var parts []string
	if in.Dir != nil {
		if cfg.HeaderShowOrg && in.Dir.OrgName != "" {
			parts = append(parts, t.AccentAlt().Render("@ "+in.Dir.OrgName))
		}
		if cfg.HeaderShowCfg {
			parts = append(parts, t.Dim().Render(in.Dir.Label))
		}
	}
	if in.Session != nil && cfg.HeaderShowUUID {
		parts = append(parts, t.Dim().Render(in.Session.ShortUUID()))
	}
	sep := t.Dim().Render(" · ")
	return strings.Join(parts, sep)
}
