#!/usr/bin/env bash
set -e

REPO="Shashwat-CODING/adx"
INSTALL_DIR="/usr/local/bin"

if [ ! -w "$INSTALL_DIR" ]; then
    INSTALL_DIR="$HOME/.local/bin"
    mkdir -p "$INSTALL_DIR"
fi

# Detect OS
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$OS" in
    darwin) OS="darwin" ;;
    linux) OS="linux" ;;
    *) echo "Unsupported OS: $OS"; exit 1 ;;
esac

# Detect Architecture
ARCH="$(uname -m)"
case "$ARCH" in
    x86_64|amd64) ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

echo "⚡ Installing ADX for ${OS}/${ARCH}..."

# Get latest release tag
LATEST_TAG=$(curl -s "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
if [ -z "$LATEST_TAG" ]; then
    LATEST_TAG="v1.0.0"
fi

TAR_URL="https://github.com/${REPO}/releases/download/${LATEST_TAG}/adx_${LATEST_TAG}_${OS}_${ARCH}.tar.gz"
TEMP_DIR=$(mktemp -d)

echo "Downloading from ${TAR_URL}..."
if curl -sSL -f "$TAR_URL" -o "${TEMP_DIR}/adx.tar.gz"; then
    tar -xzf "${TEMP_DIR}/adx.tar.gz" -C "$TEMP_DIR"
    # Find extracted adx binary
    BIN_PATH=$(find "$TEMP_DIR" -type f -name "adx" | head -n 1)
    if [ -n "$BIN_PATH" ]; then
        mv "$BIN_PATH" "${INSTALL_DIR}/adx"
        chmod +x "${INSTALL_DIR}/adx"
        echo "✔ Successfully installed adx to ${INSTALL_DIR}/adx"
    else
        echo "Error: adx binary not found in archive"
        exit 1
    fi
else
    echo "Release asset not available yet. Falling back to Go install..."
    if command -v go >/dev/null 2>&1; then
        go install "github.com/${REPO}@latest"
        echo "✔ Installed via 'go install github.com/${REPO}@latest'"
    else
        echo "Failed to download prebuilt binary. Please install Go or download from: https://github.com/${REPO}/releases"
        exit 1
    fi
fi

rm -rf "$TEMP_DIR"

echo "Run 'adx doctor' or 'adx' to get started!"
