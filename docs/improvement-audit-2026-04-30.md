# Improvement Audit — 2026-04-30

## 1. Summary

- **Project**: claude-viewer (TUI browser for Claude Code session history)
- **Working directory**: `/home/rw3iss/Sites/tools/claude-viewer`
- **Lines of Go**: ~4,800 across 29 files
- **Stack**: Go 1.22+, bubbletea/bubbles/lipgloss, fsnotify, BurntSushi/toml
- **Build**: `make build` (no test suite yet)
- **Total findings**: 13 (UI: 4, styling: 4, architecture: 5)

The project is a single-user TUI in active development with a clean
package layout (`internal/{app,screens,components,data,config,theme,
keys,debug,clipboard,events,version}`). No structural rot — most
findings are polish-level.

## 2. UI & UX improvements

### U1. Truncation helpers are inconsistent across screens
- **Where**: `screens/allorgs.go:21` (`truncateAnsi`), `components/orgtabs.go:128` (`truncOrg`), `components/sessionlist.go` (`smartTruncate`).
- **Problem**: three slightly different ellipsis-truncation helpers. `smartTruncate` is path-aware (tail-keep), the others are simple prefix-cut.
- **Fix**: a single `components/textutil.go` exposing `Truncate(s, w)` (prefix) and `TruncatePath(s, w)` (tail-aware). Each call site picks the right one. Risk: low (drop-ins).

### U2. Initial paint can show "claude-viewer: initializing…" briefly
- **Where**: every screen's `View()` early-returns when width<20.
- **Problem**: works, but identical text is duplicated in 4 places.
- **Fix**: a single `components.LoadingPlaceholder(theme)` helper. Risk: low.

### U3. Help overlay vs Modal — duplicated centering math
- **Where**: `components/help.go:78-92`, `components/modal.go:24-40`.
- **Problem**: both compute `xPad = (W - rendered.W)/2`, then build a manual top-pad with newlines, then prefix every line with the left-pad string. Identical pattern, two implementations.
- **Fix**: extract a `placeCenter(content, width, height)` helper. Both call it. Risk: low.

### U4. The all-orgs column header still hand-truncates
- **Where**: `screens/allorgs.go:147-201`.
- **Problem**: builds the header by manually slicing strings; doesn't use `smartTruncate`. After U1 it can use the unified helper.
- **Fix**: route through the shared truncate util. Risk: low.

## 3. Styling & design system

### S1. Inline `lipgloss.Color("#…")` outside theme/
- **Where**:
  - `components/modal.go:14` — `#666666` (border)
  - `components/help.go:65` — `#666` (border)
  - `components/usagemeter.go:134` — `#e5c07b` (yellow warn)
  - `components/usagemeter.go:136` — `#ffb055` (orange warn)
- **Problem**: the rest of the codebase reaches colors via `theme.Theme` getters. These four hardcoded values bypass that — themes can't override them.
- **Fix**: add `Theme.BorderSubtle()` (gray), `Theme.BarWarn()` (yellow at 70+%), `Theme.BarHot()` (orange at 50–69%) and route the existing references through them. Risk: low (visual identical).

### S2. Magic min-pad / min-size constants scattered
- **Where**: `components/sessionlist.go` (`width-13`, `lastActiveColWidth=28`, etc), `components/usagemeter.go` (`meterContentWidth=16`), `components/orgtabs.go` (`orgWrapThreshold=20`, `orgNameMaxLineWidth=30`).
- **Problem**: layout constants are spread across files. Hard to tune the design as a whole.
- **Fix**: a `components/layout.go` with all numeric constants grouped, documented. Risk: low (constant rename).

### S3. No light-mode palette
- **Where**: `theme/default.go`.
- **Problem**: only one theme registered; default works on dark terminals but is unreadable on light backgrounds.
- **Fix**: add `theme/light.go` with a paired light palette. Wire `Theme` switcher (config `theme=light` already supported by `theme.Get`). Risk: low (additive).

### S4. Hardcoded `\033[2m` / `\033[0m` ANSI sequences
- **Where**: not present in the Go code (everything goes through lipgloss). ✓ already clean. No action needed.

## 4. Architecture & code quality

### A1. Duplicate usage state in Menu and AllOrgs
- **Where**: `screens/menu.go:31-32`, `screens/allorgs.go:34-35`.
- **Problem**: both screens hold `usage map[string]*data.Usage` + `usageErr map[string]string` with identical Update handling for `UsageMsg`. Adding a third screen would dup it again.
- **Fix**: extract a `screens.usageState` struct with `Apply(msg UsageMsg)`, `Render(dir)`, `FetchCmd(repo, dirs, force)`. Both screens embed it. Risk: medium (touches two screens; needs careful ordering of Update).

### A2. `chat.renderList` is 100+ lines and does 6 things
- **Where**: `screens/chat.go:430-540`.
- **Problem**: builds blocks, applies cursor highlight, sums heights, scrolls, joins. Hard to test in isolation.
- **Fix**: extract `buildPromptBlock(p, …)` and `selectVisibleBlocks(blocks, h)` helpers. Risk: medium (moderate refactor of the busiest screen).

### A3. `if x < min { x = min }` — modernize with builtins
- **Where**: 16 occurrences across `internal/`.
- **Problem**: pre-1.21 Go style. Modern Go has `min()`/`max()` builtins.
- **Fix**: `s/if x < N { x = N }/x = max(x, N)/g`. Mechanical, identical behavior. Risk: low (compiler verifies).

### A4. `data/repo.go` — `Repository` interface getting wide
- **Where**: `data/repo.go:14-46`.
- **Problem**: `Repository` has 9 methods now (`Dirs`, `EnabledDirs`, `Sessions`, `SessionsRefresh`, `LookupForCwd`, `AddCustom`, `SetDisabled`, `RemoveCustom`, `PrefetchAll`, `Usage`, `UsageRefresh`). Borderline ISP violation — chat screen only uses 1 (`SessionsRefresh` indirectly via `LoadPrompts`); menu uses 4; settings uses 4.
- **Fix**: split into focused interfaces (`DirRegistry`, `SessionStore`, `UsageStore`) in the same file; `Repository` aggregates them. Callers can depend on the narrow ones. Risk: medium (touches all screens' constructors).

### A5. No tests
- **Where**: project root.
- **Problem**: zero `_test.go` files. Important pure functions (`smartTruncate`, `wrapOrgName`, `formatDelta`, `formatLastActive`, `relativeAgo`, `cleanText`, `meterContentWidth` clamp logic) are testable.
- **Fix**: add `_test.go` files for the pure helpers. Risk: low (pure additive).

## 5. Recommended execution plan

### Phase A — applied automatically (low risk, low blast radius)
- [x] **A3**: modernize `if x < y { x = y }` to `max()`/`min()` in 16 sites. Verified: `go build ./...` passes, `go vet ./...` clean.
- [x] **S1**: extract the 4 inline colors to `theme.BorderSubtle()` / `theme.BarWarn()` / `theme.BarHot()` and route through them. Verified: build passes; visual identical.
- [x] **U2**: collapse `claude-viewer: initializing…` early-returns into `components.LoadingPlaceholder(theme)`. Verified: build passes.

### Phase B — pending user approval (medium risk)
- **U1+U4**: unify truncation helpers in `components/textutil.go` and route `allorgs` header through it.
- **U3**: extract centered-modal placement into a shared helper consumed by `help` and `modal`.
- **S2**: collect layout magic numbers into `components/layout.go`.
- **A1**: extract `usageState` mixin used by Menu and AllOrgs.
- **A5**: add unit tests for pure helpers (`smartTruncate`, `wrapOrgName`, `formatDelta`, `formatLastActive`, `cleanText`).

### Phase C — planned, deferred
- **A2**: refactor `chat.renderList` into smaller composable functions. Worth a dedicated planning session — touches the busiest screen.
- **A4**: split `Repository` into focused interfaces. Touches every screen's constructor + `app.New`. Worth a plan.
- **S3**: light-mode theme. Pure addition but design-judgment-heavy. Worth a separate session.
