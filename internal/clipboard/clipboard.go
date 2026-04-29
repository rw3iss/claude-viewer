// Package clipboard wraps OS-native clipboard tools.
//
// Linux:   xclip -sel clip   (X11)   |   wl-copy   (Wayland)
// macOS:   pbcopy
// Windows: clip.exe
package clipboard

import (
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// Copy writes the given text to the system clipboard.
func Copy(text string) error {
	tools := candidates()
	if len(tools) == 0 {
		return errors.New("no clipboard tool available for " + runtime.GOOS)
	}
	var lastErr error
	for _, t := range tools {
		if _, err := exec.LookPath(t.bin); err != nil {
			lastErr = err
			continue
		}
		cmd := exec.Command(t.bin, t.args...)
		cmd.Stdin = strings.NewReader(text)
		if err := cmd.Run(); err != nil {
			lastErr = err
			continue
		}
		return nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no working clipboard tool: %v", names(tools))
	}
	return lastErr
}

// Available reports whether at least one tool exists.
func Available() bool {
	for _, t := range candidates() {
		if _, err := exec.LookPath(t.bin); err == nil {
			return true
		}
	}
	return false
}

type tool struct {
	bin  string
	args []string
}

func candidates() []tool {
	switch runtime.GOOS {
	case "darwin":
		return []tool{{bin: "pbcopy"}}
	case "windows":
		return []tool{{bin: "clip.exe"}, {bin: "clip"}}
	default: // linux + bsd
		return []tool{
			{bin: "wl-copy"},
			{bin: "xclip", args: []string{"-selection", "clipboard"}},
			{bin: "xsel", args: []string{"--clipboard", "--input"}},
		}
	}
}

func names(ts []tool) []string {
	out := make([]string, len(ts))
	for i, t := range ts {
		out[i] = t.bin
	}
	return out
}
