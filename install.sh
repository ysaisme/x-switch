#!/bin/bash
set -e

REPO="user/mswitch"
INSTALL_DIR="${HOME}/.local/bin"
BINARY="mswitch"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo "unsupported architecture: $ARCH"; exit 1 ;;
esac

echo "installing mswitch for ${OS}-${ARCH}..."

mkdir -p "$INSTALL_DIR"

if command -v go &>/dev/null; then
    echo "building from source..."
    git clone https://github.com/${REPO}.git /tmp/mswitch-install 2>/dev/null || true
    cd /tmp/mswitch-install
    make build-all
    cp bin/mswitch "$INSTALL_DIR/$BINARY"
    rm -rf /tmp/mswitch-install
else
    echo "go not found, downloading prebuilt binary..."
    URL="https://github.com/${REPO}/releases/latest/download/mswitch-latest-${OS}-${ARCH}"
    curl -fSL "$URL" -o "$INSTALL_DIR/$BINARY"
fi

chmod +x "$INSTALL_DIR/$BINARY"

echo ""
echo "mswitch installed to $INSTALL_DIR/$BINARY"
echo ""

if ! echo "$PATH" | grep -q "$INSTALL_DIR"; then
    echo "add $INSTALL_DIR to your PATH:"
    echo "  echo 'export PATH=\$PATH:$INSTALL_DIR' >> ~/.bashrc"
    echo "  source ~/.bashrc"
fi

echo "run 'mswitch init' to get started."
