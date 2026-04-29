// Package debug is a tiny global logger gated by a runtime flag.
//
// When Enabled is false (default), all calls are no-ops. When Enabled,
// timestamped lines are written to:
//   - stderr (visible at startup before tea takes over the screen, and
//     after tea exits)
//   - a log file at $XDG_CACHE_HOME/claude-viewer/debug.log
//
// Activate with `claude-viewer --debug`. The log path is printed to stderr
// at init so you know where to tail it during a TUI session.
package debug

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
	"time"
)

var (
	Enabled bool
	logger  *log.Logger
	logPath string
	logFile *os.File
	startup = time.Now()
)

// Init sets up the logger. Safe to call once at startup. If enabled is false
// it's a no-op (and Logf etc remain no-ops).
func Init(enabled bool) error {
	Enabled = enabled
	if !enabled {
		return nil
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return fmt.Errorf("user cache dir: %w", err)
	}
	dir := filepath.Join(cache, "claude-viewer")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	logPath = filepath.Join(dir, "debug.log")
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open %s: %w", logPath, err)
	}
	logFile = f
	mw := io.MultiWriter(os.Stderr, f)
	logger = log.New(mw, "[cv] ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
	logger.Printf("=== claude-viewer debug log starting ===")
	fmt.Fprintln(os.Stderr, "claude-viewer: debug log →", logPath)
	return nil
}

// LogPath returns the active log file path (empty if not enabled).
func LogPath() string { return logPath }

// Close flushes and closes the underlying file.
func Close() {
	if logFile != nil {
		_ = logFile.Close()
	}
}

// Logf writes a debug line. Cheap when disabled.
func Logf(format string, args ...any) {
	if !Enabled || logger == nil {
		return
	}
	// Frame skipped so log.Lshortfile shows the caller.
	_ = logger.Output(2, fmt.Sprintf(format, args...))
}

// Errf logs an error with context. Returns the same error untouched
// (handy for `if err := ...; err != nil { return debug.Errf("scan: %w", err) }`).
func Errf(format string, args ...any) error {
	err := fmt.Errorf(format, args...)
	if Enabled && logger != nil {
		_ = logger.Output(2, "ERROR: "+err.Error())
	}
	return err
}

// Section logs a "─── label ───" divider for readability.
func Section(label string) {
	if !Enabled || logger == nil {
		return
	}
	_ = logger.Output(2, "──── "+label+" ──── (+"+time.Since(startup).Truncate(time.Millisecond).String()+")")
}

// Recover is meant to be deferred at goroutine boundaries. It logs the
// panic + stack and re-panics so bubbletea can still tear down cleanly.
func Recover(where string) {
	if r := recover(); r != nil {
		stack := debug.Stack()
		if Enabled && logger != nil {
			_ = logger.Output(2, fmt.Sprintf("PANIC in %s: %v\n%s", where, r, stack))
		}
		fmt.Fprintf(os.Stderr, "\nclaude-viewer: panic in %s: %v\n", where, r)
		if Enabled {
			fmt.Fprintf(os.Stderr, "stack trace written to %s\n", logPath)
		} else {
			fmt.Fprintf(os.Stderr, "%s\nRe-run with --debug for full trace.\n", stack)
		}
		// re-panic so bubbletea's own recover restores the terminal state
		panic(r)
	}
}
