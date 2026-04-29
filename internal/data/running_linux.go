//go:build linux

package data

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// runningSessionPaths returns the set of session JSONL paths currently held
// open by any process. On Linux this is cheap — /proc walk, no shell out.
//
// Concretely: any process that has /home/.../.claude*/projects/.../*.jsonl
// open as a file descriptor counts as "running" for that session.
func runningSessionPaths() map[string]bool {
	out := map[string]bool{}
	procEntries, err := os.ReadDir("/proc")
	if err != nil {
		return out
	}
	for _, p := range procEntries {
		if !p.IsDir() {
			continue
		}
		if _, err := strconv.Atoi(p.Name()); err != nil {
			continue
		}
		fdDir := filepath.Join("/proc", p.Name(), "fd")
		fds, err := os.ReadDir(fdDir)
		if err != nil {
			continue // probably permission denied for another user's pid
		}
		for _, fd := range fds {
			link, err := os.Readlink(filepath.Join(fdDir, fd.Name()))
			if err != nil {
				continue
			}
			if strings.HasSuffix(link, ".jsonl") && strings.Contains(link, "/projects/") {
				out[link] = true
			}
		}
	}
	return out
}
