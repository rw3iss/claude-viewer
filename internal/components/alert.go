package components

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rw3iss/claude-viewer/internal/theme"
)

// AlertLevel categorizes a transient status message.
type AlertLevel int

const (
	AlertOK AlertLevel = iota
	AlertWarn
	AlertErr
)

// Alert is a transient status (toast). Timeout=0 means no auto-clear.
type Alert struct {
	Text    string
	Level   AlertLevel
	Expires time.Time
}

// AlertExpiredMsg is emitted when an alert auto-clears.
type AlertExpiredMsg struct{ ID int64 }

// AlertCmd schedules an expiration message for the given timeout.
func AlertCmd(id int64, timeout time.Duration) tea.Cmd {
	if timeout <= 0 {
		return nil
	}
	return tea.Tick(timeout, func(t time.Time) tea.Msg { return AlertExpiredMsg{ID: id} })
}

// RenderAlert returns the styled alert string (empty if expired/empty).
func RenderAlert(t theme.Theme, a Alert) string {
	if a.Text == "" {
		return ""
	}
	if !a.Expires.IsZero() && time.Now().After(a.Expires) {
		return ""
	}
	switch a.Level {
	case AlertOK:
		return t.AlertOK().Render("✓ " + a.Text)
	case AlertWarn:
		return t.AlertWarn().Render("⚠ " + a.Text)
	case AlertErr:
		return t.AlertErr().Render("✗ " + a.Text)
	}
	return a.Text
}
