#!/usr/bin/env bash
# scripts/test-binary.sh — L3: Binary execution tests
set -euo pipefail

BINARY="bin/gh-ghent"

echo "=== ghent Binary Execution Tests ==="

# Build if not already built
if [ ! -f "$BINARY" ]; then
    echo "--- Building ---"
    make build
fi

# Version
echo "--- Version ---"
$BINARY --version

# Help
echo "--- Help ---"
$BINARY --help

echo ""
echo "=== All binary tests passed ==="
