// Command claude-viewer is a TUI for browsing Claude Code session history
// across multiple config directories.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/rw3iss/claude-viewer/internal/app"
	"github.com/rw3iss/claude-viewer/internal/config"
	"github.com/rw3iss/claude-viewer/internal/data"
	"github.com/rw3iss/claude-viewer/internal/keys"
	"github.com/rw3iss/claude-viewer/internal/theme"
	"github.com/rw3iss/claude-viewer/internal/version"
)

func main() {
	if len(os.Args) >= 2 {
		switch os.Args[1] {
		case "version", "--version", "-v":
			fmt.Println("claude-viewer", version.String())
			return
		case "uninstall":
			runUninstall()
			return
		case "help", "--help", "-h":
			printUsage()
			return
		case "reset-cache":
			runResetCache()
			return
		case "update":
			runUpdate()
			return
		}
	}

	startCwd := flag.String("dir", "", "open with the session matching this directory (defaults to $PWD)")
	noAuto := flag.Bool("no-auto", false, "skip auto-open of cwd's session; always show menu")
	flag.Parse()

	cwd := *startCwd
	if cwd == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			cwd = ""
		}
	}

	// Brief stderr feedback while we load config + scan dirs (alt-screen
	// hides whatever was on the terminal before, but the user sees this
	// hint during the cold start).
	fmt.Fprint(os.Stderr, "claude-viewer: scanning sessions…\r")

	cfg, err := config.Load()
	if err != nil {
		die("failed to load config: %v", err)
	}

	repo, err := data.NewRepo(cfg)
	if err != nil {
		die("failed to init repo: %v", err)
	}

	deps := app.Deps{
		Repo:  repo,
		Cfg:   cfg,
		Theme: theme.Get(cfg.Theme),
		Keys:  keys.Default(),
	}

	if !*noAuto && cwd != "" {
		fmt.Fprint(os.Stderr, "claude-viewer: matching cwd…           \r")
		if s, d, ok := repo.LookupForCwd(cwd); ok {
			deps.InitialSession = &s
			deps.InitialDir = &d
		}
	}

	// Clear the scratch line before tea takes over (alt-screen will hide
	// anything left, but this keeps non-alt-screen invocations clean).
	fmt.Fprint(os.Stderr, "                                       \r")

	m := app.New(deps)
	prog := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := prog.Run(); err != nil {
		die("tea error: %v", err)
	}
}

func die(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "claude-viewer: "+format+"\n", a...)
	os.Exit(1)
}

func printUsage() {
	fmt.Print(`claude-viewer — TUI browser for Claude Code session history

Usage:
  claude-viewer [flags]
  claude-viewer <subcommand>

Flags:
  --dir DIR      open with the session matching DIR (defaults to $PWD)
  --no-auto      skip auto-open of cwd's session; show the main menu

Subcommands:
  version        print version
  update         re-install the latest release via 'go install'
  reset-cache    clear the disk cache
  uninstall      remove the binary + alias from your shell rc
  help           this message
`)
}

// runUpdate refreshes the binary. Strategies, in order:
//   1) If a git checkout is found (via CLAUDE_VIEWER_SRC, or known dirs),
//      run 'git pull' + 'make install' there. Uses whatever auth git
//      already has (typically SSH), avoiding go install's HTTPS-auth issues.
//   2) Else 'go install <repo>@latest'. Public repos go through proxy and
//      need no auth; private repos need GOPRIVATE + git insteadOf rewrite.
//   3) Else print the manual install one-liner.
func runUpdate() {
	if src := findCheckout(); src != "" {
		fmt.Println("updating from git checkout:", src)
		if err := runIn(src, "git", "pull", "--ff-only"); err != nil {
			die("git pull: %v", err)
		}
		if err := runIn(src, "make", "install"); err != nil {
			die("make install: %v", err)
		}
		fmt.Println("✓ updated. Run 'claude-viewer version' to confirm.")
		return
	}
	if _, err := exec.LookPath("go"); err != nil {
		fmt.Fprintln(os.Stderr, "claude-viewer: 'go' not in PATH and no local checkout found.")
		fmt.Fprintln(os.Stderr, "  Install Go from https://go.dev/dl, or re-run the install script:")
		fmt.Fprintln(os.Stderr, "  curl -fsSL https://raw.githubusercontent.com/rw3iss/claude-viewer/main/scripts/install.sh | bash")
		os.Exit(1)
	}
	const pkg = "github.com/rw3iss/claude-viewer/cmd/claude-viewer@latest"
	fmt.Println("running: go install", pkg)
	cmd := exec.Command("go", "install", pkg)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Hint: if this repo is private, set GOPRIVATE and configure git to use SSH:")
		fmt.Fprintln(os.Stderr, "  export GOPRIVATE=github.com/rw3iss/*")
		fmt.Fprintln(os.Stderr, "  git config --global url.\"git@github.com:\".insteadOf \"https://github.com/\"")
		fmt.Fprintln(os.Stderr, "Or clone the repo and set CLAUDE_VIEWER_SRC=<path>; 'cv update' will then 'git pull && make install'.")
		die("update failed: %v", err)
	}
	fmt.Println("✓ updated. Run 'claude-viewer version' to confirm.")
}

// findCheckout returns the absolute path of a claude-viewer git checkout, or "".
func findCheckout() string {
	if p := os.Getenv("CLAUDE_VIEWER_SRC"); p != "" {
		if isCheckout(p) {
			return p
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	for _, p := range []string{
		home + "/Sites/tools/claude-viewer",
		home + "/src/claude-viewer",
		home + "/code/claude-viewer",
		home + "/Code/claude-viewer",
		home + "/dev/claude-viewer",
		home + "/projects/claude-viewer",
	} {
		if isCheckout(p) {
			return p
		}
	}
	return ""
}

func isCheckout(p string) bool {
	if _, err := os.Stat(p + "/.git"); err != nil {
		return false
	}
	if _, err := os.Stat(p + "/go.mod"); err != nil {
		return false
	}
	return true
}

func runIn(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runResetCache() {
	cache, err := data.NewCache()
	if err != nil {
		die("cache: %v", err)
	}
	if err := cache.Clear(); err != nil {
		die("clear: %v", err)
	}
	fmt.Println("cache cleared:", cache.Root())
}

func runUninstall() {
	// Best-effort: remove the binary that ran us, and try to clean
	// `cv` alias from common shell rc files.
	self, err := os.Executable()
	if err == nil && self != "" {
		_ = os.Remove(self)
		fmt.Println("removed binary:", self)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	for _, rc := range []string{".zshrc", ".bashrc", ".bash_aliases", ".profile"} {
		p := home + "/" + rc
		clean(p)
	}
	fmt.Println("(if you sourced this from a non-default profile, remove `alias cv=` manually)")
}

func clean(rcPath string) {
	data, err := os.ReadFile(rcPath)
	if err != nil {
		return
	}
	const marker = "# claude-viewer alias"
	if !contains(string(data), marker) {
		return
	}
	out := stripBlock(string(data), marker)
	_ = os.WriteFile(rcPath, []byte(out), 0o644)
	fmt.Println("cleaned alias from:", rcPath)
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (indexOf(haystack, needle) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

// stripBlock removes the block delimited by `# claude-viewer alias` markers
// (`# claude-viewer alias start` ... `# claude-viewer alias end`).
func stripBlock(in, marker string) string {
	start := indexOf(in, marker+" start")
	end := indexOf(in, marker+" end")
	if start < 0 || end < 0 || end < start {
		return in
	}
	end = end + len(marker+" end")
	for end < len(in) && in[end] != '\n' {
		end++
	}
	if end < len(in) {
		end++
	}
	return in[:start] + in[end:]
}
