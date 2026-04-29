#!/usr/bin/env bash
# Standalone uninstaller — equivalent to `claude-viewer uninstall`.
set -euo pipefail

BIN_NAME="claude-viewer"
DEST="${CLAUDE_VIEWER_BIN_DIR:-$HOME/.local/bin}"

if [ -x "$DEST/$BIN_NAME" ]; then
    rm -f "$DEST/$BIN_NAME"
    echo "removed: $DEST/$BIN_NAME"
fi

for rc in "$HOME/.zshrc" "$HOME/.bashrc" "$HOME/.bash_aliases" "$HOME/.profile"; do
    [ -f "$rc" ] || continue
    if grep -q "# claude-viewer alias start" "$rc"; then
        # remove block between markers (works on macOS/BSD/Linux sed)
        awk '/# claude-viewer alias start/{f=1; next} /# claude-viewer alias end/{f=0; next} !f' "$rc" > "$rc.tmp"
        mv "$rc.tmp" "$rc"
        echo "cleaned alias from: $rc"
    fi
done

echo "done."
