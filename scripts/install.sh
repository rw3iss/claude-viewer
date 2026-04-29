#!/usr/bin/env bash
# claude-viewer installer
#
# Builds (or downloads) the binary and installs it to ~/.local/bin.
# Optionally adds a 'cv' alias to your shell rc.
#
# One-line usage:
#   curl -fsSL https://raw.githubusercontent.com/rw3iss/claude-viewer/main/scripts/install.sh | bash
#
# Locally:
#   ./scripts/install.sh

set -euo pipefail

BIN_NAME="claude-viewer"
DEST="${CLAUDE_VIEWER_BIN_DIR:-$HOME/.local/bin}"
REPO_URL="${CLAUDE_VIEWER_REPO:-https://github.com/rw3iss/claude-viewer}"

color()    { printf '\033[%sm%s\033[0m' "$1" "$2"; }
info()     { printf '%s %s\n' "$(color '1;36' '▸')" "$1"; }
ok()       { printf '%s %s\n' "$(color '1;32' '✓')" "$1"; }
warn()     { printf '%s %s\n' "$(color '1;33' '!')" "$1"; }
err()      { printf '%s %s\n' "$(color '1;31' '✗')" "$1"; }

ensure_dest() {
    mkdir -p "$DEST"
    case ":$PATH:" in
        *":$DEST:"*) ;;
        *) warn "$DEST is not on \$PATH. Add this to your shell rc:
        export PATH=\"$DEST:\$PATH\""
           ;;
    esac
}

build_local() {
    if ! command -v go >/dev/null 2>&1; then
        err "Go toolchain not found and no prebuilt binary; install Go (https://go.dev/dl) and re-run."
        exit 1
    fi
    info "Building from source (go 1.22+ required)..."
    (cd "$(dirname "$0")/.." && make build)
    install -m 0755 "$(dirname "$0")/../bin/$BIN_NAME" "$DEST/$BIN_NAME"
    ok "installed: $DEST/$BIN_NAME"
}

download_or_build() {
    # If we're inside a checkout, prefer building locally.
    if [ -f "$(dirname "$0")/../go.mod" ]; then
        build_local
        return
    fi
    # Otherwise: try `go install` from the public repo.
    if command -v go >/dev/null 2>&1; then
        info "Installing via 'go install $REPO_URL/cmd/$BIN_NAME@latest'..."
        GOBIN="$DEST" go install "$REPO_URL/cmd/$BIN_NAME@latest"
        ok "installed: $DEST/$BIN_NAME"
        return
    fi
    err "Cannot install: no Go toolchain and no local checkout."
    exit 1
}

# Set to 1 by prompt_alias when an alias gets installed (or already exists).
ALIAS_INSTALLED=0

prompt_alias() {
    # Skip if non-interactive (piped install)
    if ! [ -t 0 ] && ! [ -t 1 ]; then
        cat <<EOF

Skipping alias prompt (non-interactive install).
To enable a 'cv' shortcut, add this line to your shell rc:

    alias cv='$DEST/$BIN_NAME'

EOF
        return
    fi

    cat <<EOF

Optional: add a 'cv' alias so you can launch the viewer in any directory by
typing just 'cv' (and 'cv <dir>' to point at a different project).

EOF
    printf "Add 'cv' alias to your shell rc? [Y/n] "
    read -r answer
    case "$answer" in
        ""|y|Y|yes) ;;
        *) warn "Skipped. To add manually:  alias cv='$DEST/$BIN_NAME'"
           return ;;
    esac

    rc="$(detect_rc)"
    if [ -z "$rc" ]; then
        warn "Couldn't detect your shell rc. Add this line manually:
        alias cv='$DEST/$BIN_NAME'"
        return
    fi

    if grep -q "# claude-viewer alias start" "$rc" 2>/dev/null; then
        ok "alias already present in $rc"
        ALIAS_INSTALLED=1
        return
    fi

    cat >> "$rc" <<EOF

# claude-viewer alias start
alias cv='$DEST/$BIN_NAME'
# claude-viewer alias end
EOF
    ok "alias added to $rc — open a new shell or run: source $rc"
    ALIAS_INSTALLED=1
}

detect_rc() {
    case "${SHELL:-}" in
        */zsh)  echo "$HOME/.zshrc" ;;
        */bash)
            if [ -f "$HOME/.bash_aliases" ]; then echo "$HOME/.bash_aliases"
            else echo "$HOME/.bashrc"
            fi ;;
        *) echo "$HOME/.profile" ;;
    esac
}

main() {
    info "Installing $BIN_NAME to $DEST"
    ensure_dest
    download_or_build
    prompt_alias

    alias_line=""
    if [ "$ALIAS_INSTALLED" = "1" ]; then
        alias_line="    cv                         # use the cv alias instead
"
    fi

    cat <<EOF

$(color '1;32' 'All done.') Try it:

    $BIN_NAME              # auto-pick session for cwd
    $BIN_NAME --no-auto    # always show menu
    $BIN_NAME help         # all subcommands
${alias_line}
To remove later:  $BIN_NAME uninstall

EOF
}

main "$@"
