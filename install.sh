#!/bin/bash

# Wordsmith Install Script
# Builds locally if in git repo, otherwise downloads from GitHub releases

set -e

# Colors
GREEN='\033[0;32m'
BLUE='\033[38;5;69m'
WHITE='\033[1;37m'
RED='\033[0;31m'
NC='\033[0m'

REPO="abrayall/wordsmith"
INSTALL_DIR="/usr/local/bin"

# Detect platform and architecture
detect_platform() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)

    case "$OS" in
        darwin) OS="darwin" ;;
        linux) OS="linux" ;;
        *) echo -e "${RED}Unsupported OS: $OS${NC}"; exit 1 ;;
    esac

    case "$ARCH" in
        x86_64) ARCH="amd64" ;;
        amd64) ARCH="amd64" ;;
        arm64) ARCH="arm64" ;;
        aarch64) ARCH="arm64" ;;
        *) echo -e "${RED}Unsupported architecture: $ARCH${NC}"; exit 1 ;;
    esac
}

# Check if we're in the wordsmith git repo
is_in_repo() {
    if [ -f "go.mod" ] && grep -q "wordsmith" go.mod 2>/dev/null; then
        if [ -f "build.sh" ]; then
            return 0
        fi
    fi
    return 1
}

# Build from source
build_local() {
    ./build.sh >&2

    BINARY=$(ls build/wordsmith-*-${OS}-${ARCH} 2>/dev/null | head -1)
    if [ -z "$BINARY" ]; then
        echo -e "${RED}No binary found for ${OS}-${ARCH}${NC}" >&2
        exit 1
    fi

    echo "$BINARY"
}

# Download from GitHub releases
download_release() {
    echo -e "${BLUE}Fetching latest release...${NC}" >&2

    # Get latest release tag
    if command -v curl &> /dev/null; then
        LATEST=$(curl -sL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
    elif command -v wget &> /dev/null; then
        LATEST=$(wget -qO- "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
    else
        echo -e "${RED}Error: curl or wget is required${NC}" >&2
        exit 1
    fi

    if [ -z "$LATEST" ]; then
        echo -e "${RED}Failed to fetch latest release${NC}" >&2
        exit 1
    fi

    echo -e "${BLUE}Latest version: ${WHITE}${LATEST}${NC}" >&2
    echo "" >&2

    # Construct download URL
    VERSION="${LATEST#v}"
    FILENAME="wordsmith-${VERSION}-${OS}-${ARCH}"
    URL="https://github.com/${REPO}/releases/download/${LATEST}/${FILENAME}"

    # Create temp directory
    TMP_DIR=$(mktemp -d)
    BINARY="${TMP_DIR}/wordsmith"

    echo -e "${BLUE}Downloading ${FILENAME}...${NC}" >&2

    # Download binary
    if command -v curl &> /dev/null; then
        if ! curl -sL -f -o "$BINARY" "$URL"; then
            echo -e "${RED}Failed to download from ${URL}${NC}" >&2
            rm -rf "$TMP_DIR"
            exit 1
        fi
    else
        if ! wget -q -O "$BINARY" "$URL"; then
            echo -e "${RED}Failed to download from ${URL}${NC}" >&2
            rm -rf "$TMP_DIR"
            exit 1
        fi
    fi

    chmod +x "$BINARY"
    echo "$BINARY"
}

# Install binary
install_binary() {
    local BINARY="$1"

    echo -e "${BLUE}Installing to ${INSTALL_DIR}...${NC}"

    if [ -w "$INSTALL_DIR" ]; then
        cp "$BINARY" "$INSTALL_DIR/wordsmith"
        chmod +x "$INSTALL_DIR/wordsmith"
    else
        sudo cp "$BINARY" "$INSTALL_DIR/wordsmith"
        sudo chmod +x "$INSTALL_DIR/wordsmith"
    fi
}

# Main
detect_platform

if is_in_repo; then
    BINARY=$(build_local)
else
    BINARY=$(download_release)
fi

install_binary "$BINARY"

# Cleanup temp files if downloaded
if [[ "$BINARY" == /tmp/* ]] || [[ "$BINARY" == */tmp.* ]]; then
    rm -rf "$(dirname "$BINARY")"
fi

echo ""
echo -e "${GREEN}✓ Installed wordsmith to ${INSTALL_DIR}/wordsmith${NC}"
echo ""
echo "Run 'wordsmith --help' to get started"
echo ""
