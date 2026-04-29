// Package events holds the cross-screen tea.Msg types. Lives in its own
// package so app and screens can both import without a cycle.
package events

import "github.com/rw3iss/claude-viewer/internal/data"

// Screen is the routing identity of a screen.
type Screen int

const (
	ScreenMenu Screen = iota
	ScreenAllOrgs
	ScreenSettings
	ScreenChat
)

// SwitchScreenMsg is published by a screen when it wants the app to route.
type SwitchScreenMsg struct {
	To      Screen
	Session *data.Session
	Dir     *data.ClaudeDir
}

// QuitAppMsg signals graceful shutdown.
type QuitAppMsg struct{}
