#!/usr/bin/env bash
# scripts/test-binary.sh — L3: Binary execution tests
set -euo pipefail

BINARY="bin/gh-ghent"
PASS=0
FAIL=0

pass() { PASS=$((PASS + 1)); echo "  ✓ $1"; }
fail() { FAIL=$((FAIL + 1)); echo "  ✗ $1"; }

check() {
    if eval "$1" 2>&1; then
        pass "$2"
    else
        fail "$2"
    fi
}

echo "=== ghent Binary Execution Tests ==="

# Build if not already built
if [ ! -f "$BINARY" ]; then
    echo "--- Building ---"
    make build
fi

# --- Basic CLI tests ---
echo ""
echo "--- Version ---"
check "$BINARY --version" "version"

echo "--- Help ---"
check "$BINARY --help | grep -q comments" "help lists comments"
check "$BINARY --help | grep -q checks" "help lists checks"
check "$BINARY --help | grep -q resolve" "help lists resolve"
check "$BINARY --help | grep -q reply" "help lists reply"
check "$BINARY --help | grep -q summary" "help lists summary"

echo "--- Subcommand help ---"
COMMENTS_HELP=$("$BINARY" comments --help 2>&1)
if echo "$COMMENTS_HELP" | grep -q "review threads"; then pass "comments help"; else fail "comments help"; fi
CHECKS_HELP=$("$BINARY" checks --help 2>&1)
if echo "$CHECKS_HELP" | grep -q "check status"; then pass "checks help"; else fail "checks help"; fi

# --- Real repo tests (require gh auth) ---
echo ""
echo "--- Real Repo Tests ---"

# Comments: tbgs PR #1 has 2 unresolved threads
echo "  Testing comments (indrasvat/tbgs #1)..."
OUTPUT=$("$BINARY" comments -R indrasvat/tbgs --pr 1 --format json 2>&1) || true
if echo "$OUTPUT" | python3 -m json.tool > /dev/null 2>&1; then
    pass "comments JSON valid"
    UNRESOLVED=$(echo "$OUTPUT" | python3 -c "import json,sys; print(json.load(sys.stdin)['unresolved_count'])")
    if [ "$UNRESOLVED" -ge 1 ]; then
        pass "comments found unresolved threads ($UNRESOLVED)"
    else
        fail "comments expected unresolved threads, got $UNRESOLVED"
    fi
else
    fail "comments JSON invalid (may need gh auth)"
fi

# Comments: no ANSI in pipe output
if echo "$OUTPUT" | grep -qP '\x1b\[' 2>/dev/null; then
    fail "comments pipe has ANSI codes"
else
    pass "comments pipe has no ANSI codes"
fi

# Comments: doot PR #1 has 0 unresolved (1 resolved) — exit code 0
if "$BINARY" comments -R indrasvat/doot --pr 1 --format json > /dev/null 2>&1; then
    pass "comments exit 0 when no unresolved (doot)"
else
    fail "comments expected exit 0 (doot)"
fi

# XML format
XML_OUTPUT=$("$BINARY" comments -R indrasvat/tbgs --pr 1 --format xml 2>&1) || true
if echo "$XML_OUTPUT" | python3 -c "import xml.etree.ElementTree as ET, sys; ET.fromstring(sys.stdin.read())" 2>/dev/null; then
    pass "comments XML well-formed"
else
    fail "comments XML malformed"
fi

# Markdown format
MD_OUTPUT=$("$BINARY" comments -R indrasvat/tbgs --pr 1 --format md 2>&1) || true
if echo "$MD_OUTPUT" | grep -q "Review"; then
    pass "comments markdown has content"
else
    fail "comments markdown empty or missing header"
fi

# --- Summary ---
echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="
if [ "$FAIL" -gt 0 ]; then
    exit 1
fi
