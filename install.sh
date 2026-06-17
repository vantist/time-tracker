#!/bin/sh
set -e

REPO="vantist/time-checker"
BIN="/usr/local/bin/tt"

OS=$(uname -s)
ARCH=$(uname -m)

case "$OS" in
  Darwin)
    case "$ARCH" in
      arm64)  ARTIFACT="tt-darwin-arm64" ;;
      x86_64) ARTIFACT="tt-darwin-amd64" ;;
      *)
        echo "Unsupported platform: darwin/$ARCH"
        exit 1
        ;;
    esac
    ;;
  *)
    echo "Unsupported platform: $OS/$ARCH"
    exit 1
    ;;
esac

TAG=$(curl -s "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')
URL="https://github.com/${REPO}/releases/download/${TAG}/${ARTIFACT}"

echo "Installing tt ${TAG} (${ARTIFACT})..."
curl -fsSL "$URL" -o /tmp/tt
chmod +x /tmp/tt
mv /tmp/tt "$BIN"

# ponytail: suppress xattr error if attr not present (fresh binary won't have it)
xattr -d com.apple.quarantine "$BIN" 2>/dev/null || true

echo "Installed: $("$BIN" version)"
