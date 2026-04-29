# claude-viewer — Architecture

A bubbletea-based TUI for browsing Claude Code session history across multiple
config directories (orgs / accounts) and shipping as a cross-platform binary.

## Goals

1. **Performant** — sub-50ms cold start, smooth scroll, no flicker.
2. **Multi-org aware** — auto-detects every `~/.claude*` config and lets the
   user enable/disable / add custom dirs.
3. **Fast re-entry** — caches session lists in `~/.cache/claude-viewer/` so
   the main menu paints before disk hits finish.
4. **Cwd-aware** — when launched from inside a project dir, jumps straight
   into that session's chat view.
5. **Extensible** — config-driven keybinds + theme palette, room for plugin
   hooks later.
6. **Cross-platform** — single static Go binary, goreleaser ships
   linux/macOS/windows × amd64/arm64.

## Module layout

```
claude-viewer/
├── cmd/claude-viewer/        # CLI entry point + subcommands (uninstall, etc)
├── internal/
│   ├── app/                  # Top-level bubbletea Model — screen routing
│   ├── screens/              # One file per screen (Model implementing tea.Model)
│   │   ├── menu.go           #   Main paged-by-org session list
│   │   ├── allorgs.go        #   Multi-column view of every enabled org
│   │   ├── settings.go       #   Org enable/disable, add custom dir
│   │   └── chat.go           #   Session detail (port of the bash TUI)
│   ├── components/           # Reusable UI fragments
│   │   ├── header.go         #   Two-line styled header (configurable)
│   │   ├── footer.go         #   Status line + key hints
│   │   ├── alert.go          #   Transient toast/status (ctrl-y feedback etc)
│   │   ├── modal.go          #   Centered modal box
│   │   ├── sessionlist.go    #   Session list with active/idle styling
│   │   └── promptlist.go     #   Wrapped prompt list (chat screen)
│   ├── data/                 # Pure-data layer (no UI)
│   │   ├── claudedir.go      #   Detect & describe ~/.claude* dirs
│   │   ├── session.go        #   Session model + JSONL parser
│   │   ├── cache.go          #   On-disk cache (sessions / lists)
│   │   └── repo.go           #   Repository facade — interface over above
│   ├── config/               # User preferences (TOML)
│   ├── theme/                # Color palette interface + default theme
│   ├── keys/                 # Shared keymap
│   ├── clipboard/            # Cross-platform clipboard adapter
│   └── version/              # Build info (set via -ldflags)
├── scripts/install.sh        # One-line installer (curl | bash)
├── scripts/uninstall.sh      # Companion (also callable as `claude-viewer uninstall`)
├── docs/ARCHITECTURE.md      # This file
├── .goreleaser.yml           # Cross-compile + Homebrew tap + Scoop bucket
├── Makefile                  # build / install / lint
└── README.md
```

## Screen flow

```
                ┌──────────────────────────┐
                │       Main Menu          │
                │  (paged by org, ←/→)     │
                └────────────┬─────────────┘
        enter│               │tab            │o               │esc
             ▼               ▼               ▼                ▼
        ┌─────────┐    ┌──────────┐    ┌──────────┐     (exit app)
        │  Chat   │    │ All Orgs │    │ Settings │
        │ (esc)   │    │  (esc)   │    │  (esc)   │
        └─────────┘    └──────────┘    └──────────┘
```

When launched inside a project dir, `app` skips the menu and opens **Chat**
directly with the closest matching session — `esc` returns to the menu.

## Data flow

```
ClaudeDir scan ──► sessions ─► cache write ─► UI
                                 │
                              cache read ──► UI (next launch, fast paint)
                                 │
                              file watch (fsnotify) ─► refresh selected session
```

- **Cache**: keyed by `(claude_dir_hash, session_uuid)` and stored at
  `$XDG_CACHE_HOME/claude-viewer/sessions.json`. Refreshed in the background;
  UI repaints on completion.
- **Config**: `$XDG_CONFIG_HOME/claude-viewer/config.toml` holds enabled
  dirs, custom-added dirs, theme name, header layout, last view, keybind
  overrides.

## SOLID-ish principles applied

- **Single Responsibility**: each `screens/*.go` only owns its own model + view.
  Data fetching lives in `data/`, styling in `theme/`, key parsing in `keys/`.
- **Open/Closed**: screens implement `tea.Model`; adding a new screen means
  adding a file, not touching `app/`. `theme.Theme` is an interface — new
  themes don't require code changes elsewhere.
- **Liskov / Interface Segregation**: `data.Repository` interface is small
  (`Dirs()`, `Sessions(dir)`, `Prompts(session)`); UI never sees disk.
- **Dependency Inversion**: `app.New(deps)` injects repo + theme + clipboard
  so tests can swap any of them.

## Extension points

1. **Themes** — drop a file in `internal/theme/`, register in `theme.Registry`.
2. **Keybinds** — `config.toml` `[keys]` table overrides defaults.
3. **Header widgets** — `components/header.go` renders a slice of `Widget`s
   that can be reordered or disabled in config.
4. **Future plugin hooks** — `[hooks]` table with shell commands invoked on
   events (session-open, prompt-copy, etc).

## Cross-platform notes

- **Clipboard**: `internal/clipboard/` wraps `xclip`/`wl-copy` (Linux),
  `pbcopy` (macOS), `clip.exe` (Windows).
- **Path conventions**: `os.UserConfigDir()` / `os.UserCacheDir()` give the
  right defaults on each OS.
- **File watching**: `fsnotify` is cross-platform.
- **Distribution**: `goreleaser` produces signed releases with brew/scoop
  manifests in one CI run.
