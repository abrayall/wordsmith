#!/bin/bash

# Wordsmith Install Script
# Builds and installs wordsmith to /usr/local/bin

set -e

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

# Get script directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR"

# Build first
./build.sh

# Find the built binary for current platform
ARCH=$(uname -m)
if [ "$ARCH" = "x86_64" ]; then
    GOARCH="amd64"
elif [ "$ARCH" = "arm64" ]; then
    GOARCH="arm64"
else
    GOARCH="amd64"
fi

BINARY=$(ls build/wordsmith-*-darwin-${GOARCH} 2>/dev/null | head -1)
if [ -z "$BINARY" ]; then
    BINARY=$(ls build/wordsmith-*-linux-${GOARCH} 2>/dev/null | head -1)
fi

if [ -z "$BINARY" ]; then
    echo -e "${BLUE}No binary found for current platform${NC}"
    exit 1
fi

# Install to /usr/local/bin
INSTALL_DIR="/usr/local/bin"

echo ""
echo -e "${BLUE}Installing to ${INSTALL_DIR}...${NC}"

if [ -w "$INSTALL_DIR" ]; then
    cp "$BINARY" "$INSTALL_DIR/wordsmith"
else
    sudo cp "$BINARY" "$INSTALL_DIR/wordsmith"
fi

echo -e "${GREEN}✓ Installed wordsmith to ${INSTALL_DIR}/wordsmith${NC}"
echo ""
echo "Run 'wordsmith --help' from anywhere to get started"
echo ""
