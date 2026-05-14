#!/usr/bin/env bash

set -euo pipefail

REPO="ExiroStudio/cfmux"

echo "==> Detecting platform..."

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
    x86_64)
        ARCH="amd64"
        ;;
    aarch64|arm64)
        ARCH="arm64"
        ;;
    *)
        echo "Unsupported architecture: $ARCH"
        exit 1
        ;;
esac

BINARY="cfmux-${OS}-${ARCH}"

echo "==> Fetching latest release..."

VERSION=$(curl -fsSL \
    "https://api.github.com/repos/${REPO}/releases/latest" \
    | grep '"tag_name":' \
    | sed -E 's/.*"([^"]+)".*/\1/')

if [[ -z "$VERSION" ]]; then
    echo "Failed to fetch latest version"
    exit 1
fi

echo "==> Latest version: $VERSION"

DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${BINARY}"

TMP_FILE="$(mktemp)"

echo "==> Downloading ${BINARY}..."

curl -fL "$DOWNLOAD_URL" -o "$TMP_FILE"

chmod +x "$TMP_FILE"

echo "==> Installing to /usr/local/bin/cfmux"

sudo mv "$TMP_FILE" /usr/local/bin/cfmux

echo
echo "cfmux installed successfully!"
echo

cfmux version || true