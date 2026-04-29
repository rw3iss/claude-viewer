// Package keys defines the shared keymap. Bindings are exported as a struct
// so screens can reference the same keys consistently and config can override.
package keys

import "github.com/charmbracelet/bubbles/key"

// Map is the full key set used across screens.
type Map struct {
	Up       key.Binding
	Down     key.Binding
	Left     key.Binding
	Right    key.Binding
	Enter    key.Binding
	Esc      key.Binding
	Quit     key.Binding
	Tab      key.Binding
	Reload   key.Binding
	Settings key.Binding
	AllOrgs  key.Binding
	Search   key.Binding
	Copy     key.Binding
	Save     key.Binding
	Layout   key.Binding
	RowsUp   key.Binding
	RowsDown key.Binding
	PaneUp   key.Binding
	PaneDown key.Binding
	Home     key.Binding
	End      key.Binding
	PageUp   key.Binding
	PageDown key.Binding
	Add      key.Binding
	Toggle   key.Binding
	Delete   key.Binding
}

// Default returns the built-in keymap.
func Default() Map {
	return Map{
		Up:       key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:     key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		Left:     key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("←/h", "prev org")),
		Right:    key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("→/l", "next org")),
		Enter:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "open")),
		Esc:      key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
		Quit:     key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		Tab:      key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "toggle preview")),
		Reload:   key.NewBinding(key.WithKeys("ctrl+r"), key.WithHelp("ctrl+r", "reload")),
		Settings: key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "settings")),
		AllOrgs:  key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "all orgs")),
		Search:   key.NewBinding(key.WithKeys("ctrl+f", "/"), key.WithHelp("ctrl+f", "search")),
		Copy:     key.NewBinding(key.WithKeys("ctrl+y"), key.WithHelp("ctrl+y", "copy")),
		Save:     key.NewBinding(key.WithKeys("ctrl+o"), key.WithHelp("ctrl+o", "save")),
		Layout:   key.NewBinding(key.WithKeys("ctrl+l"), key.WithHelp("ctrl+l", "layout")),
		RowsUp:   key.NewBinding(key.WithKeys("ctrl+up"), key.WithHelp("ctrl+↑", "more rows")),
		RowsDown: key.NewBinding(key.WithKeys("ctrl+down"), key.WithHelp("ctrl+↓", "fewer rows")),
		PaneUp:   key.NewBinding(key.WithKeys("alt+up"), key.WithHelp("alt+↑", "bigger preview")),
		PaneDown: key.NewBinding(key.WithKeys("alt+down"), key.WithHelp("alt+↓", "smaller preview")),
		Home:     key.NewBinding(key.WithKeys("home", "g")),
		End:      key.NewBinding(key.WithKeys("end", "G")),
		PageUp:   key.NewBinding(key.WithKeys("pgup")),
		PageDown: key.NewBinding(key.WithKeys("pgdown")),
		Add:      key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "add dir")),
		Toggle:   key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "toggle")),
		Delete:   key.NewBinding(key.WithKeys("d", "delete"), key.WithHelp("d", "delete")),
	}
}
