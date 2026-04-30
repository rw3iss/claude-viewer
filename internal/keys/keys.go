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
	Swap     key.Binding
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
	Help     key.Binding
}

// Default returns the built-in keymap. Single-letter shortcuts are
// preferred for actions; vim-style nav aliases (h/j/k/l) are intentionally
// dropped so those letters can carry meaningful actions in the chat
// screen (l = layout, etc).
func Default() Map {
	return Map{
		Up:       key.NewBinding(key.WithKeys("up"), key.WithHelp("↑", "up")),
		Down:     key.NewBinding(key.WithKeys("down"), key.WithHelp("↓", "down")),
		Left:     key.NewBinding(key.WithKeys("left"), key.WithHelp("←", "prev org")),
		Right:    key.NewBinding(key.WithKeys("right"), key.WithHelp("→", "next org")),
		Enter:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "open")),
		Esc:      key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
		Quit:     key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		Tab:      key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "toggle preview")),
		Reload:   key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "reload")),
		Settings: key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "settings")),
		AllOrgs:  key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "all orgs")),
		Search:   key.NewBinding(key.WithKeys("f", "/"), key.WithHelp("f", "search")),
		Copy:     key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "copy")),
		Save:     key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "export")),
		Layout:   key.NewBinding(key.WithKeys("l"), key.WithHelp("l", "layout")),
		Swap:     key.NewBinding(key.WithKeys("]"), key.WithHelp("]", "swap panes")),
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
		Help:     key.NewBinding(key.WithKeys("h", "?"), key.WithHelp("h/?", "help")),
	}
}
