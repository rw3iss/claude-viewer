package theme

import "github.com/charmbracelet/lipgloss"

// default theme: dark-friendly, low-saturation. Inspired by the existing
// bash TUI's color cues: cyan = location, magenta = org, yellow = name.
type defaultTheme struct{}

func (defaultTheme) Name() string { return "default" }

func (defaultTheme) Title() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#e5c07b"))
}
func (defaultTheme) Subtitle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#56b6c2"))
}
func (defaultTheme) Dim() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#777777"))
}
func (defaultTheme) Accent() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#56b6c2"))
}
func (defaultTheme) AccentAlt() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#c678dd"))
}
func (defaultTheme) Active() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#50c878"))
}
func (defaultTheme) Idle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#666666"))
}
func (defaultTheme) Border() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#444444"))
}
func (defaultTheme) Selected() lipgloss.Style {
	return lipgloss.NewStyle().
		Background(lipgloss.Color("#3a3a3a")).
		Foreground(lipgloss.Color("#ffffff")).
		Bold(true)
}
func (defaultTheme) Highlight() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#e5c07b"))
}
func (defaultTheme) Error() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#eb5757"))
}
func (defaultTheme) Success() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#50c878"))
}

func (defaultTheme) AlertOK() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#50c878")).
		Padding(0, 1)
}
func (defaultTheme) AlertWarn() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#e5c07b")).
		Padding(0, 1)
}
func (defaultTheme) AlertErr() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#eb5757")).
		Padding(0, 1)
}

func (defaultTheme) BorderSubtle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#666666"))
}

func (defaultTheme) BarHot() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#ffb055"))
}

func (defaultTheme) BarWarn() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#e5c07b"))
}

func init() { Register(defaultTheme{}) }
