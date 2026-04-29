// Package theme defines the color palette + reusable styles. Themes are
// implementations of the Theme interface; new themes drop in as new files
// and register themselves in init().
package theme

import "github.com/charmbracelet/lipgloss"

// Theme is the swappable color palette. UI code asks for semantic styles
// (Title, Dim, Active, etc) so themes can change without UI changes.
type Theme interface {
	Name() string

	// Semantic styles
	Title() lipgloss.Style
	Subtitle() lipgloss.Style
	Dim() lipgloss.Style
	Accent() lipgloss.Style       // primary accent (project dir, selected)
	AccentAlt() lipgloss.Style    // secondary (org name)
	Active() lipgloss.Style       // active session marker
	Idle() lipgloss.Style         // idle session
	Border() lipgloss.Style       // pane borders
	Selected() lipgloss.Style     // currently-selected list row
	Highlight() lipgloss.Style    // emphasis (custom-name)
	Error() lipgloss.Style
	Success() lipgloss.Style

	// Status feedback (alerts/toasts)
	AlertOK() lipgloss.Style
	AlertWarn() lipgloss.Style
	AlertErr() lipgloss.Style
}

var registry = map[string]Theme{}

// Register adds a theme so it can be looked up by name.
func Register(t Theme) { registry[t.Name()] = t }

// Get returns the named theme or the default if missing/empty.
func Get(name string) Theme {
	if t, ok := registry[name]; ok {
		return t
	}
	return registry["default"]
}

// Names returns all registered theme names.
func Names() []string {
	out := make([]string, 0, len(registry))
	for n := range registry {
		out = append(out, n)
	}
	return out
}
