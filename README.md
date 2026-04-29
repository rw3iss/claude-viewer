# claude-viewer

A fast, multi-org TUI browser for Claude Code session history.

Auto-detects every `~/.claude*` config dir on your machine, lets you page
through their sessions, and drops you straight into the right session when
you launch from inside a project directory.

```
~/Sites/blobs                                @ Vendidit · .claude-work · b3cc1052
↑↓ nav · enter open · a all-orgs · o settings · ctrl+r reload · q quit

Page 2/3  ·  .claude-work  @ Vendidit
  ▌ blobs                                                          b3cc1052
    ~/Sites/ven/api-server                                         ecdc8014
    ~/Sites/ven/new                                                4ab800c2
    ─────────────────────────────────────────────────────────
    ~/Sites/ven/static-dev-alt-vendidit-com    (idle)              90c4ea9c
```

## Install

**One-line (any OS with Go 1.22+):**
```sh
curl -fsSL https://raw.githubusercontent.com/rw3iss/claude-viewer/main/scripts/install.sh | bash
```

**From a clone:**
```sh
git clone https://github.com/rw3iss/claude-viewer ~/Sites/tools/claude-viewer
cd ~/Sites/tools/claude-viewer
./scripts/install.sh
```

The installer asks if you want a `cv` alias added to your shell rc. Decline
and it prints the line to add manually.

## Update

```sh
claude-viewer update
```

Shells out to `go install github.com/rw3iss/claude-viewer/cmd/claude-viewer@latest`,
which fetches the newest tagged release (or `main` if no tag), builds it, and
replaces the binary in `$GOBIN` / `$GOPATH/bin` / `~/.local/bin`. Requires Go
in PATH; if not available, the command prints the manual install one-liner.

For binary-only users (no Go toolchain), re-run the installer:
```sh
curl -fsSL https://raw.githubusercontent.com/rw3iss/claude-viewer/main/scripts/install.sh | bash
```

## Uninstall

```sh
claude-viewer uninstall
# or
~/Sites/tools/claude-viewer/scripts/uninstall.sh
```

Removes the binary and any `cv` alias block previously added.

## Usage

```sh
claude-viewer            # auto-open the session matching $PWD; menu otherwise
claude-viewer --no-auto  # always show the menu
claude-viewer --dir /path/to/project   # open the session matching that dir
claude-viewer help       # subcommands
cv                       # alias for `claude-viewer`
```

### Screens

| Screen        | Enter via   | What it shows                                              |
| ------------- | ----------- | ---------------------------------------------------------- |
| **Menu**      | (default)   | Sessions for one config dir; `←/→` cycles through dirs.    |
| **All Orgs**  | `a`         | Every enabled dir as a side-by-side column.                |
| **Settings**  | `o`         | Enable/disable detected dirs, add custom paths.            |
| **Chat**      | `enter`     | Session prompts (newest first) + full content preview.    |

### Keys (chat screen)

| Key            | Action                                  |
| -------------- | --------------------------------------- |
| `↑/↓`          | navigate prompts                        |
| `pgup/pgdn`    | jump 10                                 |
| `home/end`     | first/last                              |
| `ctrl+f` / `/` | toggle search filter                    |
| `ctrl+y`       | copy highlighted prompt to clipboard    |
| `ctrl+o`       | save highlighted prompt to `$CWD/...`   |
| `ctrl+l`       | toggle bottom ↔ right preview layout    |
| `ctrl+↑/↓`     | wrap rows per prompt (1–8)              |
| `alt+↑/↓`      | grow/shrink the preview pane            |
| `ctrl+r`       | reload from disk                        |
| `esc`          | back to menu                            |
| `q`            | quit                                    |

Live-reload via `fsnotify` is automatic — new prompts appear within ~300ms.

## Multi-org behavior

claude-viewer scans `~/.claude*/projects/` automatically. Each detected dir
is one "page" in the menu (←/→ cycles). The settings screen lets you:

- toggle individual dirs on/off (`space`)
- add a custom dir not under `~/.claude*` (`n` — input field)
- remove custom dirs (`d`)

State persists in `$XDG_CONFIG_HOME/claude-viewer/config.toml`.

## Caching

Session lists per dir are cached at `$XDG_CACHE_HOME/claude-viewer/sessions-*.json`
(short TTL — 5s). The first paint after launch comes from cache; the
background re-scan replaces it when ready. Clear with:

```sh
claude-viewer reset-cache
```

<details>
<summary><b>Configuration file</b></summary>

```toml
# ~/.config/claude-viewer/config.toml

theme         = "default"
preview_rows  = 2
preview_size  = 60       # %
layout        = "bottom"  # or "right"

# Dirs the user explicitly hid (overrides auto-detect):
disabled = []

# User-added custom dirs (non-default paths):
custom = []

# Header widget toggles:
header_show_name = true   # /rename custom title
header_show_dir  = true   # ~/Sites/...
header_show_org  = true   # @ Org Name
header_show_cfg  = true   # .claude-2 / .claude-work
header_show_uuid = true   # short uuid
```

</details>

<details>
<summary><b>Architecture (for contributors)</b></summary>

See [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md). Short version:

- `internal/data` — pure data layer (no UI). Detect dirs, parse JSONL, cache.
- `internal/screens` — one file per screen, each a tea.Model.
- `internal/components` — reusable UI fragments (header, footer, alert, modal, lists).
- `internal/theme` — palette interface; default theme registered in `init()`.
  Drop a new theme in this dir and call `theme.Register()`.
- `internal/keys` — shared keymap.
- `internal/clipboard` — OS-aware copy adapter (xclip / wl-copy / pbcopy / clip.exe).
- `internal/events` — cross-screen `tea.Msg` types (avoids import cycle).
- `internal/app` — root tea.Model that routes between screens.
- `cmd/claude-viewer` — CLI entry + subcommands.

Build / test:
```sh
make build           # bin/claude-viewer
make install         # → ~/.local/bin
make dev             # go run
make test            # go test
make cross           # cross-compile dist/* for linux/macOS/windows
```

</details>

<details>
<summary><b>Cross-platform notes</b></summary>

- **Linux**: clipboard via `xclip` (X11) → falls back to `wl-copy` (Wayland) → `xsel`.
- **macOS**: clipboard via `pbcopy`. Single binary; `brew install` once goreleaser is wired up.
- **Windows**: clipboard via `clip.exe`. Run from any terminal that supports
  ANSI (Windows Terminal, ConEmu, etc.). A `.cmd` launcher could be wrapped
  later — see roadmap.

</details>

<details>
<summary><b>Roadmap</b></summary>

- [ ] goreleaser CI pipeline (Homebrew tap + Scoop bucket)
- [ ] Mouse drag-to-resize the preview divider (bubbletea exposes drag events)
- [ ] Theme switcher UI (currently config-only)
- [ ] Plugin hooks via `[hooks]` table — shell out on session-open/copy/etc
- [ ] macOS .app bundle (`platypus` or AppleScript wrapper)
- [ ] Windows .exe launcher (just opens Windows Terminal pointing at the binary)

</details>

## License

MIT.
