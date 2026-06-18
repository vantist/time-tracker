#!/bin/sh
set -e

REPO="vantist/time-checker"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
BIN="$INSTALL_DIR/tt"

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
mkdir -p "$INSTALL_DIR"
curl -fsSL "$URL" -o /tmp/tt
chmod +x /tmp/tt
mv /tmp/tt "$BIN"

# ponytail: suppress xattr error if attr not present (fresh binary won't have it)
xattr -d com.apple.quarantine "$BIN" 2>/dev/null || true

case ":$PATH:" in
  *":$INSTALL_DIR:"*) ;;
  *) echo "Add to PATH: export PATH=\"\$HOME/.local/bin:\$PATH\"" ;;
esac

echo "Installed: $("$BIN" version)"
