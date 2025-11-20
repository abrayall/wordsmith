#!/bin/sh

# Wordsmith Test Script

set -e

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[38;2;59;130;246m'
NC='\033[0m'

echo "=============================================="
echo "Wordsmith Tests"
echo "=============================================="
echo ""

printf "${BLUE}Running unit tests...${NC}\n"
echo ""

if go test ./... -v; then
    echo ""
    printf "${GREEN}✓ All tests passed${NC}\n"
else
    echo ""
    printf "${RED}✗ Tests failed${NC}\n"
    exit 1
fi

echo ""
echo "=============================================="
printf "${GREEN}Test Complete!${NC}\n"
echo "=============================================="
