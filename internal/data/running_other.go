//go:build !linux

package data

// runningSessionPaths is a no-op on non-Linux platforms for now.
// macOS could use `lsof -c claude` but it's slow; Windows is harder.
// Sessions will fall back to mtime-based "recent" categorization.
func runningSessionPaths() map[string]bool { return nil }
