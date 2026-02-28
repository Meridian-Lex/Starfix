#!/usr/bin/env bash
# Starfix installer
# Run from the cloned repository root.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BINARY_NAME="starfix"
INSTALL_BIN="$HOME/.local/bin"
CONFIG_DIR="$HOME/.config/starfix"
# IDENTITY-EXCEPTION: functional internal reference — not for public exposure
SETTINGS_FILE="$HOME/.claude/settings.json"

print_banner() {
    echo ""
    echo "  Starfix — Post-Compaction Context Restoration"
    echo ""
}

check_deps() {
    local missing=0
    for dep in go jq; do
        if ! command -v "$dep" &>/dev/null; then
            echo "  ERROR: $dep is required but not found in PATH"
            missing=1
        fi
    done
    if [[ "$missing" -eq 1 ]]; then
        echo ""
        echo "  Note: Go must be in PATH. Try: export PATH=\$PATH:/home/meridian/go/bin"
        exit 1
    fi
}

build_binary() {
    echo "  Building starfix binary..."
    (cd "$SCRIPT_DIR" && go build -o "$SCRIPT_DIR/bin/$BINARY_NAME" ./cmd/starfix/)
    echo "  Build complete."
}

install_files() {
    mkdir -p "$INSTALL_BIN"
    cp "$SCRIPT_DIR/bin/$BINARY_NAME" "$INSTALL_BIN/$BINARY_NAME"
    chmod +x "$INSTALL_BIN/$BINARY_NAME"

    cp "$SCRIPT_DIR/hooks/precompact.sh" "$INSTALL_BIN/starfix-precompact"
    cp "$SCRIPT_DIR/hooks/sessionstart.sh" "$INSTALL_BIN/starfix-sessionstart"
    cp "$SCRIPT_DIR/hooks/userpromptsubmit.sh" "$INSTALL_BIN/starfix-userpromptsubmit"
    chmod +x "$INSTALL_BIN/starfix-precompact" \
             "$INSTALL_BIN/starfix-sessionstart" \
             "$INSTALL_BIN/starfix-userpromptsubmit"

    mkdir -p "$CONFIG_DIR"
    if [[ ! -f "$CONFIG_DIR/starfix.cfg" ]]; then
        cp "$SCRIPT_DIR/config/starfix.cfg" "$CONFIG_DIR/starfix.cfg"
        echo "  Config written to $CONFIG_DIR/starfix.cfg"
    else
        echo "  Existing config preserved at $CONFIG_DIR/starfix.cfg"
    fi

    echo "  Files installed."
}

register_hooks() {
    local tmp
    tmp=$(mktemp)

    jq '
      .hooks.PreCompact = (.hooks.PreCompact // []) + [{"hooks": [{"type": "command", "command": "'"$INSTALL_BIN/starfix-precompact"'", "timeout": 30}]}] |
      .hooks.SessionStart = (.hooks.SessionStart // []) + [{"hooks": [{"type": "command", "command": "'"$INSTALL_BIN/starfix-sessionstart"'", "timeout": 15}]}] |
      .hooks.UserPromptSubmit = (.hooks.UserPromptSubmit // []) + [{"hooks": [{"type": "command", "command": "'"$INSTALL_BIN/starfix-userpromptsubmit"'", "timeout": 15}]}]
    ' "$SETTINGS_FILE" > "$tmp" && mv "$tmp" "$SETTINGS_FILE"

    echo "  Hooks registered in $SETTINGS_FILE"
}

remove_hooks() {
    local tmp
    tmp=$(mktemp)

    jq '
      .hooks.PreCompact = [.hooks.PreCompact[]? | select(.hooks[0].command | test("starfix") | not)] |
      .hooks.SessionStart = [.hooks.SessionStart[]? | select(.hooks[0].command | test("starfix") | not)] |
      .hooks.UserPromptSubmit = [.hooks.UserPromptSubmit[]? | select(.hooks[0].command | test("starfix") | not)]
    ' "$SETTINGS_FILE" > "$tmp" && mv "$tmp" "$SETTINGS_FILE"

    echo "  Hooks removed from $SETTINGS_FILE"
}

remove_files() {
    rm -f "$INSTALL_BIN/$BINARY_NAME" \
          "$INSTALL_BIN/starfix-precompact" \
          "$INSTALL_BIN/starfix-sessionstart" \
          "$INSTALL_BIN/starfix-userpromptsubmit"
    echo "  Binaries removed."

    read -r -p "  Remove config directory $CONFIG_DIR? [y/N] " answer
    if [[ "${answer,,}" == "y" ]]; then
        rm -rf "$CONFIG_DIR"
        echo "  Config removed."
    else
        echo "  Config preserved."
    fi
}

do_install() {
    echo ""
    check_deps
    build_binary
    install_files
    register_hooks
    echo ""
    echo "  Starfix installed. Restart your Lex session to activate hooks."
    echo ""
}

do_remove() {
    echo ""
    remove_hooks
    remove_files
    echo ""
    echo "  Starfix removed. Restart your Lex session to deactivate hooks."
    echo ""
}

# Main menu
print_banner

echo "  1) Install for current user"
echo "  2) Remove"
echo ""
read -r -p "  > " choice

case "$choice" in
    1) do_install ;;
    2) do_remove ;;
    *) echo "  Invalid choice."; exit 1 ;;
esac
