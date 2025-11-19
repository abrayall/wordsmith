#!/bin/bash

# Wordsmith Build Script
# Builds the wordsmith CLI tool for multiple platforms

set -e  # Exit on error

echo "======================================"
echo "Wordsmith Build"
echo "======================================"
echo ""

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
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

# Get version from latest git tag
echo -e "${BLUE}Reading version from git tags...${NC}"
GIT_DESCRIBE=$(git describe --tags --match "v*.*.*" 2>/dev/null || echo "v0.1.0")

# Parse git describe output
# Format: v0.1.0 or v0.1.0-5-g1a2b3c4 (if commits exist after tag)
if [[ "$GIT_DESCRIBE" =~ ^v([0-9]+)\.([0-9]+)\.([0-9]+)(-([0-9]+)-g([0-9a-f]+))?$ ]]; then
    MAJOR="${BASH_REMATCH[1]}"
    MINOR="${BASH_REMATCH[2]}"
    MAINTENANCE="${BASH_REMATCH[3]}"
    COMMIT_COUNT="${BASH_REMATCH[5]}"

    # If there are commits after the tag, append commit count to maintenance
    if [[ -n "$COMMIT_COUNT" ]]; then
        MAINTENANCE="${MAINTENANCE}-${COMMIT_COUNT}"
        VERSION="${MAJOR}.${MINOR}.${MAINTENANCE}"
    else
        VERSION="${MAJOR}.${MINOR}.${MAINTENANCE}"
    fi
else
    # Fallback
    MAJOR=0
    MINOR=1
    MAINTENANCE=0
    VERSION="0.1.0"
fi

# Check for uncommitted local changes
if [[ -n $(git status --porcelain 2>/dev/null) ]]; then
    TIMESTAMP=$(date +"%m%d%H%M")
    MAINTENANCE="${MAINTENANCE}-${TIMESTAMP}"
    VERSION="${MAJOR}.${MINOR}.${MAINTENANCE}"
    echo -e "${BLUE}Detected uncommitted changes, appending timestamp${NC}"
fi

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
echo "======================================"
echo -e "${GREEN}Build Complete!${NC}"
echo "======================================"
echo ""
echo "Artifacts created in build/:"
ls -1 "$BUILD_DIR"
echo ""
