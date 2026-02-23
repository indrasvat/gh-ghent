#!/usr/bin/env bash
# scripts/test-agent-workflow.sh — L5: Agent workflow tests
# Tests the full agent workflow: read comments → parse → verify formats
set -euo pipefail

BINARY="bin/gh-ghent"
PASS=0
FAIL=0

pass() { PASS=$((PASS + 1)); echo "  ✓ $1"; }
fail() { FAIL=$((FAIL + 1)); echo "  ✗ $1"; }

echo "=== ghent Agent Workflow Tests ==="

# Build if not already built
if [ ! -f "$BINARY" ]; then
    echo "--- Building ---"
    make build
fi

# --- Test repos ---
# indrasvat/tbgs #1: 2 unresolved threads, checks pass
# indrasvat/peek-it #2: 1 unresolved thread, checks fail
# indrasvat/doot #1: 0 unresolved, checks pass (merge-ready)

echo ""
echo "--- Workflow 1: Read comments, parse JSON ---"
THREADS=$("$BINARY" comments -R indrasvat/tbgs --pr 1 --format json 2>&1) || true
THREAD_ID=$(echo "$THREADS" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d['threads'][0]['id'])" 2>/dev/null) || true
if [ -n "$THREAD_ID" ] && [[ "$THREAD_ID" == PRRT_* ]]; then
    pass "parsed thread ID: $THREAD_ID"
else
    fail "could not parse thread ID from comments JSON"
fi

UNRESOLVED=$(echo "$THREADS" | python3 -c "import json,sys; print(json.load(sys.stdin)['unresolved_count'])" 2>/dev/null) || true
if [ "$UNRESOLVED" -ge 1 ]; then
    pass "unresolved count: $UNRESOLVED"
else
    fail "expected unresolved threads"
fi

echo ""
echo "--- Workflow 2: Exit codes reflect state ---"

# Comments with unresolved → exit 1
EXIT=0
"$BINARY" comments -R indrasvat/tbgs --pr 1 --format json > /dev/null 2>&1 || EXIT=$?
if [ "$EXIT" -eq 1 ]; then
    pass "comments exit 1 with unresolved threads"
else
    fail "comments expected exit 1, got $EXIT"
fi

# Comments with all resolved → exit 0
if "$BINARY" comments -R indrasvat/doot --pr 1 --format json > /dev/null 2>&1; then
    pass "comments exit 0 with no unresolved"
else
    fail "comments expected exit 0"
fi

echo ""
echo "--- Workflow 3: Format consistency ---"

# JSON is valid (exit 1 is expected for unresolved threads — not an error)
JSON_OUT=$("$BINARY" comments -R indrasvat/tbgs --pr 1 --format json 2>&1) || true
if echo "$JSON_OUT" | python3 -m json.tool > /dev/null 2>&1; then
    pass "JSON valid"
else
    fail "JSON invalid"
fi

# XML is well-formed
XML_OUT=$("$BINARY" comments -R indrasvat/tbgs --pr 1 --format xml 2>&1) || true
if echo "$XML_OUT" | python3 -c "import xml.etree.ElementTree as ET, sys; ET.fromstring(sys.stdin.read())" 2>/dev/null; then
    pass "XML valid"
else
    fail "XML invalid"
fi

# No ANSI in any format
for fmt in json xml md; do
    OUTPUT=$("$BINARY" comments -R indrasvat/tbgs --pr 1 --format "$fmt" 2>&1) || true
    if echo "$OUTPUT" | grep -qP '\x1b\[' 2>/dev/null; then
        fail "$fmt format has ANSI codes"
    else
        pass "$fmt format clean (no ANSI)"
    fi
done

# NOTE: Checks, resolve, reply, and summary workflow tests will be added
# as those commands are implemented (Tasks 2.2-2.6).

# --- Summary ---
echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="
if [ "$FAIL" -gt 0 ]; then
    exit 1
fi
