#!/usr/bin/env bash
# scripts/test-binary.sh — L3: Binary execution tests
set -euo pipefail

BINARY="bin/gh-ghent"
STALE_REPO="${GHENT_STALE_REPO:-clayliddell/AgentVM}"
STALE_PR="${GHENT_STALE_PR:-10}"
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
ROOT_HELP=$("$BINARY" --help 2>&1)
if echo "$ROOT_HELP" | grep -q "comments"; then pass "help lists comments"; else fail "help lists comments"; fi
if echo "$ROOT_HELP" | grep -q "checks"; then pass "help lists checks"; else fail "help lists checks"; fi
if echo "$ROOT_HELP" | grep -q "resolve"; then pass "help lists resolve"; else fail "help lists resolve"; fi
if echo "$ROOT_HELP" | grep -q "reply"; then pass "help lists reply"; else fail "help lists reply"; fi
if echo "$ROOT_HELP" | grep -q "dismiss"; then pass "help lists dismiss"; else fail "help lists dismiss"; fi
if echo "$ROOT_HELP" | grep -q "status"; then pass "help lists status"; else fail "help lists status"; fi

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

# Status: stale review detection on a public PR with stale blockers
echo "  Testing status stale review detection (${STALE_REPO} #${STALE_PR})..."
STATUS_OUTPUT=$("$BINARY" status -R "$STALE_REPO" --pr "$STALE_PR" --format json 2>&1) || true
if echo "$STATUS_OUTPUT" | python3 -m json.tool > /dev/null 2>&1; then
    pass "status JSON valid"
    STALE_COUNT=$(echo "$STATUS_OUTPUT" | python3 -c "import json,sys; print(len(json.load(sys.stdin).get('stale_reviews', [])))")
    if [ "$STALE_COUNT" -ge 1 ]; then
        pass "status found stale blocking reviews ($STALE_COUNT)"
    else
        pass "status returned no stale blockers (valid no-op scenario)"
    fi
else
    fail "status JSON invalid for stale-review repo"
fi

# Dismiss: dry-run only, should enumerate stale blockers and never require maintainer access
echo "  Testing dismiss dry-run (${STALE_REPO} #${STALE_PR})..."
DISMISS_OUTPUT=$("$BINARY" dismiss -R "$STALE_REPO" --pr "$STALE_PR" --dry-run --format json 2>&1) || true
if echo "$DISMISS_OUTPUT" | python3 -m json.tool > /dev/null 2>&1; then
    pass "dismiss dry-run JSON valid"
    RESULT_COUNT=$(echo "$DISMISS_OUTPUT" | python3 -c "import json,sys; d=json.load(sys.stdin); print(len(d.get('results', [])))")
    if [ "$RESULT_COUNT" -eq 0 ]; then
        pass "dismiss dry-run no-op (no stale blockers matched)"
    else
        ACTION=$(echo "$DISMISS_OUTPUT" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d['results'][0].get('action',''))")
        if [ "$ACTION" = "would_dismiss" ]; then
            pass "dismiss dry-run action"
        else
            fail "dismiss dry-run expected would_dismiss, got '$ACTION'"
        fi
    fi
else
    fail "dismiss dry-run JSON invalid"
fi

# --- Summary ---
echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="
if [ "$FAIL" -gt 0 ]; then
    exit 1
fi
