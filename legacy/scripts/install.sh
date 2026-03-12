#!/bin/bash
# SENTINEL V3 — Quick Install
# Usage: curl -sSL https://raw.githubusercontent.com/ward331/sentinel/main/scripts/install.sh | bash

set -e

VERSION="${SENTINEL_VERSION:-3.0.0}"
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

BINARY="sentinel-${OS}-${ARCH}"
URL="https://github.com/ward331/sentinel/releases/download/v${VERSION}/${BINARY}.tar.gz"

echo "Installing SENTINEL v${VERSION} (${OS}/${ARCH})..."
TMP=$(mktemp -d)
curl -sSL "$URL" -o "$TMP/sentinel.tar.gz"
tar xzf "$TMP/sentinel.tar.gz" -C "$TMP"
mkdir -p "$HOME/.local/bin"
mv "$TMP/$BINARY" "$HOME/.local/bin/sentinel"
chmod +x "$HOME/.local/bin/sentinel"
rm -rf "$TMP"

echo "Installed to $HOME/.local/bin/sentinel"
echo "Run: sentinel --wizard"
