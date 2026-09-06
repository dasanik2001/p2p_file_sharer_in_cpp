#!/bin/sh
# Transfera CLI — One-Line Installer for macOS & Linux
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/dasanik2001/transfera-client/main/install.sh | bash
#
# What this does:
#   1. Detects your OS (macOS / Linux) and architecture (amd64 / arm64)
#   2. Downloads the correct binary from the latest GitHub Release
#   3. Installs it to ~/.local/bin/transfera
#   4. Adds ~/.local/bin to your PATH if not already there

set -e

REPO="dasanik2001/transfera-client"
INSTALL_DIR="$HOME/.local/bin"
BINARY_NAME="transfera"

# --- Colors ---
RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
YELLOW='\033[1;33m'
BOLD='\033[1m'
NC='\033[0m' # No Color

info()  { printf "  ${CYAN}[*]${NC} %s\n" "$1"; }
ok()    { printf "  ${GREEN}[OK]${NC} %s\n" "$1"; }
err()   { printf "  ${RED}[!]${NC} %s\n" "$1"; }

echo ""
echo "  ${CYAN}============================================${NC}"
echo "    Transfera CLI — Installer"
echo "  ${CYAN}============================================${NC}"
echo ""

# 1. Detect OS and architecture
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$OS" in
    linux)  OS="linux" ;;
    darwin) OS="darwin" ;;
    *)      err "Unsupported OS: $OS"; exit 1 ;;
esac

case "$ARCH" in
    x86_64|amd64)   ARCH="amd64" ;;
    arm64|aarch64)  ARCH="arm64" ;;
    *)              err "Unsupported architecture: $ARCH"; exit 1 ;;
esac

ASSET_NAME="transfera-${OS}-${ARCH}"
info "Detected platform: ${OS}/${ARCH}"

# 2. Find the latest release
info "Finding latest release..."
if command -v curl >/dev/null 2>&1; then
    RELEASE_JSON=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" 2>/dev/null) || {
        err "Could not fetch latest release. Check your internet connection."
        exit 1
    }
elif command -v wget >/dev/null 2>&1; then
    RELEASE_JSON=$(wget -qO- "https://api.github.com/repos/$REPO/releases/latest" 2>/dev/null) || {
        err "Could not fetch latest release. Check your internet connection."
        exit 1
    }
else
    err "Neither curl nor wget found. Please install one of them."
    exit 1
fi

# Parse tag name
TAG=$(echo "$RELEASE_JSON" | grep '"tag_name"' | head -1 | sed 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')
if [ -z "$TAG" ]; then
    err "Could not determine latest version."
    exit 1
fi
ok "Latest version: $TAG"

# Parse download URL for our asset
DOWNLOAD_URL=$(echo "$RELEASE_JSON" | grep "browser_download_url" | grep "$ASSET_NAME" | head -1 | sed 's/.*"browser_download_url"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')
if [ -z "$DOWNLOAD_URL" ]; then
    err "Binary '$ASSET_NAME' not found in release $TAG."
    err "Visit https://github.com/$REPO/releases/latest for available downloads."
    exit 1
fi

# 3. Create install directory
mkdir -p "$INSTALL_DIR"

# 4. Download the binary
DEST_PATH="$INSTALL_DIR/$BINARY_NAME"
info "Downloading transfera $TAG..."

if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$DOWNLOAD_URL" -o "$DEST_PATH"
else
    wget -qO "$DEST_PATH" "$DOWNLOAD_URL"
fi

chmod +x "$DEST_PATH"
ok "Installed to $DEST_PATH"

# 5. Add to PATH if needed
add_to_path() {
    local rc_file="$1"
    local export_line="export PATH=\"$INSTALL_DIR:\$PATH\""
    local marker="# Added by Transfera CLI"

    if [ -f "$rc_file" ] && grep -q "$INSTALL_DIR" "$rc_file" 2>/dev/null; then
        return 0  # Already configured
    fi

    echo "" >> "$rc_file"
    echo "$marker" >> "$rc_file"
    echo "$export_line" >> "$rc_file"
    return 1
}

PATH_ADDED=false

# Check if already in PATH
case ":$PATH:" in
    *":$INSTALL_DIR:"*)
        ok "Already in PATH."
        ;;
    *)
        info "Adding to PATH..."

        # Detect current shell and configure RC file
        CURRENT_SHELL="$(basename "$SHELL" 2>/dev/null || echo "bash")"

        case "$CURRENT_SHELL" in
            zsh)
                add_to_path "$HOME/.zshrc" && ok "Already configured in ~/.zshrc" || ok "Added to ~/.zshrc"
                ;;
            bash)
                if [ "$OS" = "darwin" ]; then
                    add_to_path "$HOME/.bash_profile" && ok "Already configured in ~/.bash_profile" || ok "Added to ~/.bash_profile"
                else
                    add_to_path "$HOME/.bashrc" && ok "Already configured in ~/.bashrc" || ok "Added to ~/.bashrc"
                fi
                ;;
            fish)
                FISH_CONFIG="$HOME/.config/fish/config.fish"
                mkdir -p "$(dirname "$FISH_CONFIG")"
                if ! grep -q "$INSTALL_DIR" "$FISH_CONFIG" 2>/dev/null; then
                    echo "" >> "$FISH_CONFIG"
                    echo "# Added by Transfera CLI" >> "$FISH_CONFIG"
                    echo "set -gx PATH $INSTALL_DIR \$PATH" >> "$FISH_CONFIG"
                fi
                ok "Added to $FISH_CONFIG"
                ;;
            *)
                add_to_path "$HOME/.profile" && ok "Already configured in ~/.profile" || ok "Added to ~/.profile"
                ;;
        esac
        PATH_ADDED=true
        ;;
esac

# 6. Done!
echo ""
echo "  ${GREEN}============================================${NC}"
echo "    Transfera $TAG installed successfully!"
echo "  ${GREEN}============================================${NC}"
echo ""

if [ "$PATH_ADDED" = true ]; then
    echo "  ${YELLOW}Restart your terminal or run:${NC}"
    echo "    source ~/.$(basename "$SHELL")rc"
    echo ""
fi

echo "  Then type:"
echo "    ${BOLD}transfera${NC}"
echo ""
