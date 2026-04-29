// Package screens defines the screen interface and implementations.
//
// Each screen is a struct with Init / Update / View matching tea.Model.
// They communicate up to the app via SwitchScreenMsg / QuitAppMsg in the
// app package.
package screens

import tea "github.com/charmbracelet/bubbletea"

// Screen is the local interface for in-app screens. Same shape as tea.Model
// but renamed for clarity.
type Screen interface {
	Init() tea.Cmd
	Update(tea.Msg) (Screen, tea.Cmd)
	View() string

	// SetSize is called whenever the terminal resizes.
	SetSize(width, height int)
}
