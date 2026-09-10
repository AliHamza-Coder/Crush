#!/bin/bash
# CRUSH Installer - One command install
# Usage: curl -fsSL https://raw.githubusercontent.com/AliHamza-Coder/Crush/main/scripts/install.sh | bash

set -e

REPO="AliHamza-Coder/Crush"
INSTALL_DIR="$HOME/.local/bin"

echo ""
echo "  Installing CRUSH..."
echo ""

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$OS" in
    linux)
        case "$ARCH" in
            x86_64) ASSET_PATTERN="linux-x64.tar.gz" ;;
            aarch64) ASSET_PATTERN="linux-arm64.tar.gz" ;;
            *) echo "  Error: Unsupported architecture: $ARCH"; exit 1 ;;
        esac
        ;;
    darwin)
        case "$ARCH" in
            x86_64) ASSET_PATTERN="macos-x64.tar.gz" ;;
            arm64) ASSET_PATTERN="macos-arm64.tar.gz" ;;
            *) echo "  Error: Unsupported architecture: $ARCH"; exit 1 ;;
        esac
        ;;
    *)
        echo "  Error: Unsupported OS: $OS"
        exit 1
        ;;
esac

# Get latest version
LATEST=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest")
VERSION=$(echo "$LATEST" | grep '"tag_name"' | sed 's/.*"v\([^"]*\)".*/\1/')
ASSET_URL=$(echo "$LATEST" | grep -o "https://[^\"]*$ASSET_PATTERN" | head -1)

if [ -z "$ASSET_URL" ]; then
    echo "  Error: No build found for $OS $ARCH"
    exit 1
fi

echo "  Downloading v$VERSION..."
mkdir -p "$INSTALL_DIR"
curl -fsSL "$ASSET_URL" | tar xz -C "$INSTALL_DIR" --strip-components=1

# Add to PATH if needed
if ! echo "$PATH" | grep -q "$INSTALL_DIR"; then
    mkdir -p "$HOME/.local"
    echo "export PATH=\"\$PATH:$INSTALL_DIR\"" >> "$HOME/.bashrc"
    echo "export PATH=\"\$PATH:$INSTALL_DIR\"" >> "$HOME/.zshrc" 2>/dev/null || true
    export PATH="$PATH:$INSTALL_DIR"
fi

echo ""
echo "  Done! Run: crush"
echo ""
