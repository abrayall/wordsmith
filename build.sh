#!/bin/bash

# Wordsmith Build Script
# Builds the wordsmith CLI tool for multiple platforms

set -e  # Exit on error

echo "=============================================="
echo "Wordsmith Build"
echo "=============================================="
echo ""

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[38;2;59;130;246m'
NC='\033[0m' # No Color

# Get script directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR"

# Build directory
BUILD_DIR="$SCRIPT_DIR/build"

# Clean previous build
echo -e "${BLUE}Cleaning previous build...${NC}"
rm -rf "$BUILD_DIR"
mkdir -p "$BUILD_DIR"

# Get version using vermouth
echo -e "${BLUE}Reading version from git tags...${NC}"
VERSION=$(vermouth 2>/dev/null || curl -sfL https://raw.githubusercontent.com/abrayall/vermouth/refs/heads/main/vermouth.sh | sh -)

echo -e "${GREEN}Building version: ${VERSION}${NC}"
echo ""

# Build for multiple platforms
PLATFORMS=("darwin/amd64" "darwin/arm64" "linux/amd64" "linux/arm64" "windows/amd64")

for PLATFORM in "${PLATFORMS[@]}"; do
    GOOS="${PLATFORM%/*}"
    GOARCH="${PLATFORM#*/}"

    OUTPUT_NAME="wordsmith-${VERSION}-${GOOS}-${GOARCH}"
    if [ "$GOOS" = "windows" ]; then
        OUTPUT_NAME="${OUTPUT_NAME}.exe"
    fi

    echo -e "${BLUE}Building ${GOOS}/${GOARCH}...${NC}"

    GOOS=$GOOS GOARCH=$GOARCH go build \
        -ldflags "-X wordsmith/cmd.Version=${VERSION}" \
        -o "$BUILD_DIR/$OUTPUT_NAME" \
        .

    echo -e "${GREEN}✓ Created: ${OUTPUT_NAME}${NC}"
    echo ""
done

# Summary
echo ""
echo "=============================================="
echo -e "${GREEN}Build Complete!${NC}"
echo "=============================================="
echo ""
echo "Artifacts created in build/:"
ls -1 "$BUILD_DIR"
echo ""
