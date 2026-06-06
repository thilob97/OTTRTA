#!/bin/sh
set -e

# Repository details
OWNER="handyfun97"
REPO="ottrta"

# Detect OS and Arch
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

case "$OS" in
    linux) OS="linux" ;;
    darwin) OS="darwin" ;;
    *) echo "Unsupported OS: $OS"; exit 1 ;;
esac

# Get latest release tag from GitHub API
echo "Checking latest release of $OWNER/$REPO..."
LATEST_RELEASE_URL="https://api.github.com/repos/$OWNER/$REPO/releases/latest"
# Fallback to curl/grep if jq is not installed
if command -v jq >/dev/null 2>&1; then
    TAG=$(curl -fsSL "$LATEST_RELEASE_URL" | jq -r .tag_name)
else
    TAG=$(curl -fsSL "$LATEST_RELEASE_URL" | grep -m 1 '"tag_name":' | cut -d'"' -f4)
fi

if [ -z "$TAG" ] || [ "$TAG" = "null" ]; then
    # Fallback to query redirect if API limit is reached
    TAG=$(curl -fsSL -o /dev/null -w "%{url_effective}" "https://github.com/$OWNER/$REPO/releases/latest" | sed 's|.*/||')
fi

if [ -z "$TAG" ]; then
    echo "Could not detect latest release version."
    exit 1
fi

VERSION="${TAG#v}"
echo "Downloading ottrta $TAG for ${OS}/${ARCH}..."

# Construct download URL
FILENAME="ottrta_${VERSION}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/$OWNER/$REPO/releases/download/$TAG/$FILENAME"

# Download to temp directory
TEMP_DIR=$(mktemp -d)
CLEANUP() {
    rm -rf "$TEMP_DIR"
}
trap CLEANUP EXIT

curl -fsSL "$URL" -o "$TEMP_DIR/$FILENAME"

# Extract binary
tar -xzf "$TEMP_DIR/$FILENAME" -C "$TEMP_DIR" ottrta

# Determine install directory
INSTALL_DIR="/usr/local/bin"
if [ ! -w "$INSTALL_DIR" ]; then
    # Fallback to home bin if /usr/local/bin is not writable and sudo is not available
    if command -v sudo >/dev/null 2>&1; then
        echo "Installing to $INSTALL_DIR (requires sudo)..."
        sudo mv "$TEMP_DIR/ottrta" "$INSTALL_DIR/ottrta"
        sudo chmod +x "$INSTALL_DIR/ottrta"
    else
        INSTALL_DIR="$HOME/.local/bin"
        mkdir -p "$INSTALL_DIR"
        echo "Installing to $INSTALL_DIR..."
        mv "$TEMP_DIR/ottrta" "$INSTALL_DIR/ottrta"
        chmod +x "$INSTALL_DIR/ottrta"
        echo "Please ensure $INSTALL_DIR is in your PATH."
    fi
else
    echo "Installing to $INSTALL_DIR..."
    mv "$TEMP_DIR/ottrta" "$INSTALL_DIR/ottrta"
    chmod +x "$INSTALL_DIR/ottrta"
fi

echo "Successfully installed ottrta!"
ottrta version || true
