# ghent Testing Strategy

> Comprehensive verification plan ensuring ghent works in reality, not just in isolation.
> Date: 2026-02-22

**The cardinal rule:** Every feature MUST be verified by _actually running_ `gh ghent` against a real GitHub repository. Unit tests and smoke tests are necessary but NOT sufficient. If the binary hasn't been executed and its output visually inspected, the feature is NOT verified.

---

## Table of Contents

1. [Testing Pyramid](#1-testing-pyramid)
2. [L1: Unit Tests](#2-l1-unit-tests)
3. [L2: Integration Tests](#3-l2-integration-tests)
4. [L3: Binary Execution Tests (CRITICAL)](#4-l3-binary-execution-tests-critical)
5. [L4: Visual Verification via iterm2-driver](#5-l4-visual-verification-via-iterm2-driver)
6. [L5: Agent Workflow Tests](#6-l5-agent-workflow-tests)
7. [Known TUI Pitfalls (from yukti/vivecaka)](#7-known-tui-pitfalls-from-yuktivivecaka)
8. [iterm2-driver Script Catalog](#8-iterm2-driver-script-catalog)
9. [Test Infrastructure](#9-test-infrastructure)
10. [Verification Checklist Per Feature](#10-verification-checklist-per-feature)
11. [cm Integration](#11-cm-integration)

---

## 1. Testing Pyramid

```
                    ┌───────────┐
                    │    L5     │  Agent workflow tests
                    │  (3-5)    │  End-to-end with AI agents
                   ┌┴───────────┴┐
                   │     L4      │  Visual verification (iterm2-driver)
                   │   (10-15)   │  Screenshots + layout checks
                  ┌┴─────────────┴┐
                  │      L3       │  Binary execution (gh ghent ...)
                  │   (20-30)     │  Run real binary, verify output
                 ┌┴───────────────┴┐
                 │       L2        │  Integration tests
                 │    (15-25)      │  HTTP mocking, API contracts
                ┌┴─────────────────┴┐
                │        L1         │  Unit tests
                │     (50-100)      │  Pure functions, parsers, formatters
                └───────────────────┘
```

**Every layer is mandatory. No feature ships without L3 verification.**

---

## 2. L1: Unit Tests

Standard Go table-driven tests for pure functions. These run fast and catch logic regressions.

### What to test at L1

| Component | Tests |
|-----------|-------|
| GraphQL response parsing | Parse review threads, check runs, annotations |
| Output formatters | Markdown, XML, JSON rendering of review threads |
| CI status aggregation | fail > pending > pass priority logic |
| Comment threading | `in_reply_to_id` grouping, insertion order |
| Diff hunk extraction | Parsing `diffHunk` into file:line context |
| Flag validation | `--format` values, `--interval` bounds |
| Exit code logic | 0/1/2 determination from check states |

### Example

```go
func TestAggregateCI(t *testing.T) {
    tests := []struct {
        name   string
        checks []CheckRun
        want   CIStatus
    }{
        {"all pass", []CheckRun{{Status: "COMPLETED", Conclusion: "SUCCESS"}}, CIPass},
        {"one fail", []CheckRun{
            {Status: "COMPLETED", Conclusion: "SUCCESS"},
            {Status: "COMPLETED", Conclusion: "FAILURE"},
        }, CIFail},
        {"pending beats pass", []CheckRun{
            {Status: "COMPLETED", Conclusion: "SUCCESS"},
            {Status: "IN_PROGRESS"},
        }, CIPending},
        {"fail beats pending", []CheckRun{
            {Status: "IN_PROGRESS"},
            {Status: "COMPLETED", Conclusion: "FAILURE"},
        }, CIFail},
        {"empty", nil, CINone},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := aggregateCI(tt.checks)
            if got != tt.want {
                t.Errorf("aggregateCI() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Running

```bash
make test          # go test ./...
make test-race     # go test -race ./...
make coverage      # go test -coverprofile=coverage/cover.out ./...
```

---

## 3. L2: Integration Tests

Test API interaction layers with HTTP mocking. Verify GraphQL queries and REST calls produce correct domain objects.

### What to test at L2

| Component | Tests |
|-----------|-------|
| GraphQL review thread query | Response shape → domain.ReviewThread |
| GraphQL resolveReviewThread mutation | Success + error cases |
| REST check runs endpoint | Response shape → domain.CheckRun |
| REST job logs endpoint | Log parsing, step filtering |
| REST annotations endpoint | Lint error extraction |
| `go-gh` API client wiring | DefaultRESTClient, DefaultGraphQLClient |
| Rate limit handling | 403/429 responses → graceful error |

### HTTP Mocking Pattern

```go
func TestGetUnresolvedThreads(t *testing.T) {
    // Use httptest to mock GitHub API
    mux := http.NewServeMux()
    mux.HandleFunc("/graphql", func(w http.ResponseWriter, r *http.Request) {
        body, _ := io.ReadAll(r.Body)
        if !strings.Contains(string(body), "reviewThreads") {
            t.Fatal("expected reviewThreads query")
        }
        w.Header().Set("Content-Type", "application/json")
        w.Write(loadTestdata(t, "review_threads_response.json"))
    })
    srv := httptest.NewServer(mux)
    defer srv.Close()

    client := NewGitHubClient(WithBaseURL(srv.URL))
    threads, err := client.GetUnresolvedThreads(context.Background(), "owner/repo", 42)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if len(threads) != 3 {
        t.Errorf("got %d threads, want 3", len(threads))
    }
}
```

### Test fixtures

Store in `testdata/` directory:

```
testdata/
├── review_threads_response.json     # GraphQL reviewThreads response
├── check_runs_response.json         # REST check runs response
├── job_log_lint_errors.txt          # Sample job log with lint annotations
├── job_log_test_failures.txt        # Sample job log with test failures
├── annotations_response.json        # REST annotations response
├── large_pr_100_threads.json        # Stress test: 100 review threads
└── rate_limit_response.json         # 403 rate limit response
```

---

## 4. L3: Binary Execution Tests (CRITICAL)

**This is where most bugs are caught.** L3 tests build the actual binary and run it against real GitHub repos (or a test repo with known state).

### Setup: Install ghent as a local gh extension

```bash
# Build the binary
make build

# Install as a local extension (symlink)
gh extension install .

# Verify installation
gh extension list | grep ghent
gh ghent --help
```

### What to verify at L3

Every command must be run and its output inspected:

#### `gh ghent comments`

```bash
# Basic: fetch unresolved review threads
gh ghent comments

# With PR number
gh ghent comments --pr 42

# Different output formats
gh ghent comments --format markdown
gh ghent comments --format xml
gh ghent comments --format json

# Pipe to file (verify non-TTY output)
gh ghent comments --format json | jq '.threads | length'

# Explicit repo
gh ghent comments -R owner/repo --pr 42

# Verify: file paths, line numbers, comment bodies, thread grouping
```

#### `gh ghent checks`

```bash
# Basic: show check run status
gh ghent checks

# With PR number
gh ghent checks --pr 42

# Show failing logs
gh ghent checks --logs

# Watch mode
gh ghent checks --watch --interval 10

# JSON output for agents
gh ghent checks --format json

# Verify: check names, status icons, durations, log excerpts
```

#### `gh ghent resolve`

```bash
# Resolve a specific thread
gh ghent resolve --thread <thread-id>

# Resolve all unresolved threads
gh ghent resolve --all

# Unresolve a thread
gh ghent resolve --unresolve --thread <thread-id>

# Verify: thread actually resolves on GitHub (check in browser!)
```

### Test Runner Script

Create a shell script that exercises all commands:

```bash
#!/usr/bin/env bash
# scripts/test-binary.sh - Run binary execution tests
set -euo pipefail

REPO="indrasvat/ghent"  # or a test repo with known PRs
PR_NUM="${1:-}"          # optional: specific PR number

echo "=== ghent Binary Execution Tests ==="
echo "Repo: $REPO"
echo ""

# Build
echo "--- Building ---"
make build
gh extension install . 2>/dev/null || true

# Help
echo "--- Help output ---"
gh ghent --help
gh ghent comments --help
gh ghent checks --help

# Comments (markdown)
echo "--- Comments (markdown) ---"
if [ -n "$PR_NUM" ]; then
    gh ghent comments -R "$REPO" --pr "$PR_NUM" --format markdown
fi

# Comments (json)
echo "--- Comments (json) ---"
if [ -n "$PR_NUM" ]; then
    gh ghent comments -R "$REPO" --pr "$PR_NUM" --format json | head -50
fi

# Checks
echo "--- Checks ---"
if [ -n "$PR_NUM" ]; then
    gh ghent checks -R "$REPO" --pr "$PR_NUM"
fi

# Checks (json)
echo "--- Checks (json) ---"
if [ -n "$PR_NUM" ]; then
    gh ghent checks -R "$REPO" --pr "$PR_NUM" --format json | jq '.checks | length'
fi

# Version
echo "--- Version ---"
gh ghent --version

echo ""
echo "=== All binary tests passed ==="
```

### What to look for

| Issue | How to detect |
|-------|---------------|
| Panic/crash | Non-zero exit code, stack trace in stderr |
| Garbled output | Pipe to file, inspect for ANSI codes in non-TTY |
| Missing data | Compare against `gh pr view --json` output |
| Wrong exit code | `echo $?` after each command |
| Slow response | `time gh ghent checks` should be <2s |
| Auth failure | Run without GH_TOKEN to verify error message |
| Rate limiting | Run 50 rapid calls, verify graceful degradation |

---

## 5. L4: Visual Verification via iterm2-driver

iterm2-driver automates iTerm2 to launch ghent, capture screenshots, and verify visual output. This catches issues that text comparison misses: layout misalignment, color bleed, box-drawing character corruption.

### Script Structure

All visual test scripts follow this canonical pattern:

```python
# /// script
# requires-python = ">=3.14"
# dependencies = [
#   "iterm2",
#   "pyobjc",
#   "pyobjc-framework-Quartz",
# ]
# ///

"""
ghent <Command> Visual Test: Automated verification of gh ghent <command>

Tests:
    1. Launch: Verify gh ghent <command> renders expected output
    2. Content: Verify file paths, line numbers, status indicators
    3. Format: Verify output format matches expected structure
    4. Pipe: Verify non-TTY output (no ANSI escapes)

Verification Strategy:
    - Poll screen content with 5-second timeout per state
    - Verify header, content, and footer sections
    - Check box-drawing characters form connected borders
    - Capture screenshots at each stage for manual review

Screenshots:
    - ghent_<cmd>_launch.png: Initial output
    - ghent_<cmd>_content.png: Full output rendered
    - ghent_<cmd>_pipe.png: Non-TTY output

Screenshot Inspection Checklist:
    - Colors: Status indicators (green pass, red fail, yellow pending)
    - Boundaries: Table borders, section dividers
    - Visible Elements: File paths, line numbers, comment bodies
    - Alignment: Columns align, no orphaned box corners

Usage:
    uv run .claude/automations/test_ghent_<cmd>.py
"""

import iterm2
import asyncio
import subprocess
import os
import time
from datetime import datetime

PROJECT_ROOT = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
SCREENSHOT_DIR = os.path.join(PROJECT_ROOT, ".claude", "screenshots")
TIMEOUT_SECONDS = 5.0

# ... (result tracking, Quartz window targeting, verification helpers)
# ... (see canonical template in iterm2-driver skill)
```

### Visual Tests for Each Command

#### Test: `gh ghent comments`

```python
async def test_comments(session):
    """Verify gh ghent comments output."""

    # Build and install
    await session.async_send_text(f"cd {PROJECT_ROOT} && make build && gh extension install .\n")
    await asyncio.sleep(3.0)

    # Run comments command
    await session.async_send_text("gh ghent comments -R indrasvat/tbgs --pr 1\n")
    await asyncio.sleep(2.0)

    # Verify header
    if await verify_screen_contains(session, "Review Threads", "header"):
        log_result("Header visible", "PASS")
    else:
        log_result("Header visible", "FAIL", "No header found")
        await dump_screen(session, "comments_no_header")

    # Verify file paths appear
    if await verify_screen_contains(session, ".go:", "file path with line"):
        log_result("File paths shown", "PASS")
    else:
        log_result("File paths shown", "UNVERIFIED", "No .go file paths")

    # Verify comment bodies
    screen = await session.async_get_screen_contents()
    line_count = 0
    for i in range(screen.number_of_lines):
        if screen.line(i).string.strip():
            line_count += 1
    if line_count > 5:
        log_result("Content rendered", "PASS", f"{line_count} non-empty lines")
    else:
        log_result("Content rendered", "FAIL", f"Only {line_count} lines")

    capture_screenshot("ghent_comments")
```

#### Test: `gh ghent checks`

```python
async def test_checks(session):
    """Verify gh ghent checks output."""

    await session.async_send_text("gh ghent checks -R indrasvat/tbgs --pr 1\n")
    await asyncio.sleep(2.0)

    # Verify status indicators are present
    screen_text = await get_all_screen_text(session)

    # Should show check names
    if "CI" in screen_text or "test" in screen_text.lower() or "build" in screen_text.lower():
        log_result("Check names visible", "PASS")
    else:
        log_result("Check names visible", "FAIL", "No check names found")

    # Should show status (pass/fail/pending indicators)
    has_status = any(s in screen_text for s in ["pass", "fail", "pending", "PASS", "FAIL", "PENDING", "✓", "✗", "●"])
    if has_status:
        log_result("Status indicators", "PASS")
    else:
        log_result("Status indicators", "UNVERIFIED", "No status indicators found")

    capture_screenshot("ghent_checks")
```

#### Test: `gh ghent checks --watch`

```python
async def test_watch_mode(session):
    """Verify --watch mode with auto-refresh."""

    await session.async_send_text("gh ghent checks --watch --interval 5 -R indrasvat/tbgs --pr 1\n")
    await asyncio.sleep(3.0)

    # Capture initial state
    capture_screenshot("ghent_watch_initial")

    # Verify refresh indicator
    if await verify_screen_contains(session, "refresh", "refresh countdown"):
        log_result("Refresh indicator", "PASS")
    else:
        log_result("Refresh indicator", "UNVERIFIED", "No refresh text")

    # Wait for a refresh cycle
    await asyncio.sleep(6.0)
    capture_screenshot("ghent_watch_refreshed")

    # Verify content updated (screen should still have check data)
    screen_text = await get_all_screen_text(session)
    if len(screen_text.strip()) > 20:
        log_result("Watch mode content", "PASS")
    else:
        log_result("Watch mode content", "FAIL", "Screen empty after refresh")

    # Exit watch mode
    await session.async_send_text("q")
    await asyncio.sleep(0.5)
```

#### Test: Output Format Verification

```python
async def test_json_output(session):
    """Verify JSON output is valid (no ANSI codes, parseable)."""

    await session.async_send_text(
        "gh ghent comments -R indrasvat/tbgs --pr 1 --format json > /tmp/ghent_test.json 2>&1; "
        "echo EXIT_CODE=$?\n"
    )
    await asyncio.sleep(3.0)

    # Check exit code
    if await verify_screen_contains(session, "EXIT_CODE=0", "zero exit"):
        log_result("JSON exit code", "PASS")
    else:
        log_result("JSON exit code", "FAIL", "Non-zero exit code")

    # Validate JSON
    await session.async_send_text("python3 -m json.tool /tmp/ghent_test.json > /dev/null 2>&1 && echo JSON_VALID || echo JSON_INVALID\n")
    await asyncio.sleep(1.0)

    if await verify_screen_contains(session, "JSON_VALID", "json validation"):
        log_result("JSON valid", "PASS")
    else:
        log_result("JSON valid", "FAIL", "Invalid JSON output")
        await session.async_send_text("cat /tmp/ghent_test.json | head -5\n")
        await asyncio.sleep(0.5)
        await dump_screen(session, "invalid_json")

    capture_screenshot("ghent_json_output")
```

#### Test: Layout Integrity

```python
async def test_layout_integrity(session):
    """Verify box-drawing characters and alignment."""

    await session.async_send_text("gh ghent comments -R indrasvat/tbgs --pr 1\n")
    await asyncio.sleep(2.0)

    # Check box-drawing character connectivity
    box_result = await verify_box_integrity(session, "comments output")
    if box_result['valid']:
        log_result("Box integrity", "PASS")
    else:
        log_result("Box integrity", "FAIL", box_result['issues'][0])
        await dump_layout_debug(session, "box_failure")

    # Check for background color bleed (consecutive spaces > 10 in middle of content)
    screen = await session.async_get_screen_contents()
    bleed_issues = []
    for i in range(screen.number_of_lines):
        line = screen.line(i).string
        if not line.strip():
            continue
        # Check for suspicious gaps
        in_content = False
        space_run = 0
        for ch in line:
            if ch != ' ':
                if in_content and space_run > 15:
                    bleed_issues.append(f"Line {i}: {space_run}-space gap (possible bleed)")
                in_content = True
                space_run = 0
            else:
                space_run += 1

    if not bleed_issues:
        log_result("No color bleed", "PASS")
    else:
        log_result("No color bleed", "FAIL", bleed_issues[0])

    capture_screenshot("ghent_layout")
```

### Running Visual Tests

```bash
# Run a specific visual test
uv run .claude/automations/test_ghent_comments.py

# Run all visual tests
for f in .claude/automations/test_ghent_*.py; do
    echo "=== Running $f ==="
    uv run "$f"
done

# Screenshots saved to .claude/screenshots/ (gitignored)
```

---

## 6. L5: Agent Workflow Tests

Verify ghent works as expected when called by AI coding agents (Claude Code, Codex, etc.).

### What agents expect

| Expectation | Test |
|-------------|------|
| Valid JSON from `--format json` | Parse output with `jq`, verify schema |
| Meaningful exit codes | 0 = all clear, 1 = issues found, 2 = error |
| No ANSI in piped output | `gh ghent checks --format json \| cat` has no escape codes |
| Concise error messages | Stderr is actionable, not a stack trace |
| Fast response | <2s for any non-watch command |
| Handles missing PR gracefully | `gh ghent comments --pr 99999` returns clean error |

### Agent Test Script

```bash
#!/usr/bin/env bash
# scripts/test-agent-workflow.sh
set -euo pipefail

REPO="indrasvat/ghent"
PR="${1:-1}"

echo "=== Agent Workflow Tests ==="

# 1. JSON output is valid
echo "--- JSON validity ---"
OUTPUT=$(gh ghent comments -R "$REPO" --pr "$PR" --format json 2>/dev/null)
echo "$OUTPUT" | python3 -m json.tool > /dev/null
echo "PASS: JSON is valid"

# 2. No ANSI codes in piped output
echo "--- No ANSI in pipe ---"
if echo "$OUTPUT" | grep -P '\x1b\[' > /dev/null 2>&1; then
    echo "FAIL: ANSI codes found in JSON output"
    exit 1
fi
echo "PASS: No ANSI codes"

# 3. Exit codes
echo "--- Exit codes ---"
gh ghent checks -R "$REPO" --pr "$PR" --format json > /dev/null 2>&1
CHECKS_EXIT=$?
echo "checks exit code: $CHECKS_EXIT (0=clear, 1=issues, 2=error)"

# 4. Error handling
echo "--- Error handling ---"
if gh ghent comments -R "nonexistent/repo" --pr 1 2>/dev/null; then
    echo "FAIL: Should have errored on nonexistent repo"
    exit 1
fi
echo "PASS: Clean error on nonexistent repo"

# 5. Performance
echo "--- Performance ---"
START=$(date +%s%N)
gh ghent checks -R "$REPO" --pr "$PR" --format json > /dev/null 2>&1
END=$(date +%s%N)
DURATION=$(( (END - START) / 1000000 ))
echo "Duration: ${DURATION}ms"
if [ "$DURATION" -gt 5000 ]; then
    echo "WARN: Response >5s"
fi

echo ""
echo "=== All agent workflow tests passed ==="
```

---

## 7. Known TUI Pitfalls (from yukti/vivecaka)

These are bugs discovered across our Go TUI projects. Every one MUST be checked in ghent.

### 7.1 Terminal Background Color Bleed

**Problem:** Empty terminal cells show the terminal's default background instead of the app's color. Creates a "two-tone" appearance.

**Root cause:** `lipgloss.Background()` only applies to explicitly rendered characters. Empty cells use the terminal's default background.

**Solution:** Use `termenv.SetBackgroundColor()` before starting BubbleTea, reset with `output.Reset()` on exit.

```go
// BEFORE starting BubbleTea
output := termenv.NewOutput(os.Stdout)
output.SetBackgroundColor(output.Color(string(styles.Background)))

p := tea.NewProgram(app, tea.WithAltScreen())
_, err := p.Run()

// AFTER TUI exits (before any os.Exit)
output.Reset()
```

**Source:** yukti commit `64d8eb7`, file `internal/cli/tui.go:101-112`

**Visual test:** Screenshot the app, look for mismatched background colors in empty areas.

### 7.2 Go Switch Type Assertion Variable Shadowing

**Problem:** `switch msg := msg.(type)` creates a LOCAL variable. Modifications to it don't propagate to the outer `msg`.

**Broken:**
```go
switch msg := msg.(type) {
case tea.WindowSizeMsg:
    msg.Height = max(1, msg.Height-6)  // Only modifies LOCAL copy!
}
// Views receive ORIGINAL msg with full height → header pushed off-screen
```

**Fixed:**
```go
switch typedMsg := msg.(type) {
case tea.WindowSizeMsg:
    typedMsg.Height = max(1, typedMsg.Height-6)
    msg = typedMsg  // CRITICAL: reassign to outer variable
}
```

**Source:** yukti commit `f6336ce`, file `internal/tui/app.go:99-112`

### 7.3 Empty String Padding Breaks Modal Overlay

**Problem:** `ensureExactHeight()` padding with `""` causes `ansi.Cut("", 0, leftOffset)` to return nothing, so modals start at column 0.

**Fix:** Pad with full-width space strings: `strings.Repeat(" ", width)`

**Source:** yukti commit `f6336ce`, file `internal/tui/views/projects.go:681-701`

### 7.4 lipgloss.Background() on Elements Causes Bleed

**Problem:** `Background()` wrappers on styled elements cause color bleed when composited with modals.

**Fix:** Remove `Background()` entirely. Use `termenv.SetBackgroundColor()` globally, and rely on border styling for visual separation.

**Source:** yukti commit `b52c860`

### 7.5 lipgloss.Width() on Inner Elements Causes Padding Bleed

**Problem:** Using `lipgloss.Width()` on inner elements introduces extra padding that leaks when composited.

**Fix:** Use manual padding with `strings.Repeat(" ", delta)` instead.

**Source:** yukti commit `93e71af`, file `internal/tui/components/help.go:141-196`

### 7.6 ANSI Escape Code Bleed Between Elements

**Problem:** ANSI codes from styled borders leak into adjacent padding/content.

**Fix:** Add explicit `\033[0m` (ANSI reset) codes between styled elements and padding.

**Source:** yukti commit `fcf91d0`

### 7.7 Modal Key Interception

**Problem:** Global key handlers (Esc/Back) fire before modals handle them, navigating away instead of closing the modal.

**Fix:** Implement `ModalHandler` interface, check `HasModal()` before global key handling.

**Source:** yukti commit `15178f7`, file `internal/tui/app.go:136-153`

### 7.8 Modal Overlay Destroys Background Content

**Problem:** Plain spaces in modal overlay erase background ANSI codes.

**Fix:** Use `ansi.Cut()` from `github.com/charmbracelet/x/ansi` to extract and preserve background content.

**Source:** yukti commit `aa9c8f5`, file `internal/tui/views/workspace.go:799-852`

### 7.9 API Timeout on Large PR Lists

**Problem:** `statusCheckRollup` field causes API timeouts when fetching 50+ PRs.

**Fix:** Use dual field lists — full for first page, light (without statusCheckRollup) for pagination.

**Source:** vivecaka `internal/adapter/ghcli/reader.go:12-17`

### 7.10 Large File Syntax Highlighting Stalls UI

**Problem:** Chroma tokenization stalls on files with 5000+ lines.

**Fix:** Set `maxHighlightLines = 5000` threshold. Skip highlighting above it.

**Source:** vivecaka `internal/tui/views/diffview.go:100-102`

---

## 8. iterm2-driver Script Catalog

Scripts live in `.claude/automations/` and are run with `uv run`.

| Script | Purpose | Screenshots |
|--------|---------|-------------|
| `test_ghent_help.py` | Help output, version, subcommand listing | help, version |
| `test_ghent_comments.py` | Comments command all formats | comments_md, comments_xml, comments_json |
| `test_ghent_checks.py` | Checks command, status indicators | checks, checks_fail, checks_pass |
| `test_ghent_watch.py` | Watch mode with refresh cycle | watch_initial, watch_refreshed |
| `test_ghent_resolve.py` | Resolve command (dry-run) | resolve_dryrun |
| `test_ghent_pipe.py` | Non-TTY output (piped) | pipe_json, pipe_md |
| `test_ghent_layout.py` | Box integrity, alignment, bleed | layout_ok, layout_debug |
| `test_ghent_errors.py` | Error handling (bad repo, bad PR, rate limit) | error_notfound |
| `test_ghent_agent.py` | Agent workflow (JSON, exit codes, perf) | N/A (text-only) |
| `test_ghent_install.py` | Extension install, list, upgrade | install, list |

### Script Template for ghent

```python
# /// script
# requires-python = ">=3.14"
# dependencies = [
#   "iterm2",
#   "pyobjc",
#   "pyobjc-framework-Quartz",
# ]
# ///

"""
ghent Comments Visual Test: Automated verification of `gh ghent comments`

Tests:
    1. Build: Verify `make build` succeeds
    2. Install: Verify `gh extension install .` succeeds
    3. Help: Verify `gh ghent comments --help` shows usage
    4. Markdown: Verify markdown output has file paths and comments
    5. JSON: Verify JSON output is valid and parseable
    6. XML: Verify XML output has expected structure
    7. Layout: Verify box-drawing characters are connected
    8. No Bleed: Verify no background color bleed in output

Verification Strategy:
    - Build binary, install as extension, run against test repo
    - Poll screen for expected content with 5s timeout
    - Capture screenshot at each stage
    - Run layout verification (box integrity, bleed detection)
    - Validate JSON by piping through python3 json.tool

Screenshots:
    - ghent_comments_help.png
    - ghent_comments_markdown.png
    - ghent_comments_json.png
    - ghent_comments_layout.png

Screenshot Inspection Checklist:
    - Colors: File paths highlighted, comment text readable
    - Boundaries: Section dividers intact, no orphaned corners
    - Visible Elements: File paths, line numbers, author names, comment text
    - Alignment: Columns align for multi-thread display

Usage:
    uv run .claude/automations/test_ghent_comments.py
"""

import iterm2
import asyncio
import subprocess
import os
import time
from datetime import datetime

# ============================================================
# CONFIGURATION
# ============================================================

PROJECT_ROOT = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
SCREENSHOT_DIR = os.path.join(PROJECT_ROOT, ".claude", "screenshots")
TIMEOUT_SECONDS = 5.0
TEST_REPO = "indrasvat/tbgs"  # Repo with 2 unresolved review threads on PR #1
TEST_PR = "1"  # PR with review threads + passing checks

# ============================================================
# RESULT TRACKING
# ============================================================

results = {
    "passed": 0,
    "failed": 0,
    "unverified": 0,
    "tests": [],
    "screenshots": [],
    "start_time": None,
    "end_time": None,
}


def log_result(test_name: str, status: str, details: str = "", screenshot: str = None):
    results["tests"].append({"name": test_name, "status": status, "details": details, "screenshot": screenshot})
    if screenshot:
        results["screenshots"].append(screenshot)
    if status == "PASS":
        results["passed"] += 1
        print(f"  [+] PASS: {test_name}")
    elif status == "FAIL":
        results["failed"] += 1
        print(f"  [x] FAIL: {test_name} - {details}")
    else:
        results["unverified"] += 1
        print(f"  [?] UNVERIFIED: {test_name} - {details}")
    if screenshot:
        print(f"      Screenshot: {screenshot}")


def print_summary() -> int:
    results["end_time"] = datetime.now()
    total = results["passed"] + results["failed"] + results["unverified"]
    duration = (results["end_time"] - results["start_time"]).total_seconds()
    print("\n" + "=" * 60)
    print("TEST SUMMARY")
    print("=" * 60)
    print(f"Duration:   {duration:.1f}s")
    print(f"Total:      {total}")
    print(f"Passed:     {results['passed']}")
    print(f"Failed:     {results['failed']}")
    print(f"Unverified: {results['unverified']}")
    if results["screenshots"]:
        print(f"Screenshots: {len(results['screenshots'])}")
    print("=" * 60)
    if results["failed"] > 0:
        print("\nFailed tests:")
        for test in results["tests"]:
            if test["status"] == "FAIL":
                print(f"  - {test['name']}: {test['details']}")
    print("\n" + "-" * 60)
    if results["failed"] > 0:
        print("OVERALL: FAILED")
        return 1
    elif results["unverified"] > 0:
        print("OVERALL: PASSED (with unverified tests)")
        return 0
    else:
        print("OVERALL: PASSED")
        return 0


def print_test_header(test_name: str, test_num: int):
    print("\n" + "=" * 60)
    print(f"TEST {test_num}: {test_name}")
    print("=" * 60)


# ============================================================
# QUARTZ WINDOW TARGETING
# ============================================================

try:
    import Quartz
    def get_iterm2_window_id():
        window_list = Quartz.CGWindowListCopyWindowInfo(
            Quartz.kCGWindowListOptionOnScreenOnly | Quartz.kCGWindowListExcludeDesktopElements,
            Quartz.kCGNullWindowID
        )
        for window in window_list:
            if 'iTerm' in window.get('kCGWindowOwnerName', ''):
                return window.get('kCGWindowNumber')
        return None
except ImportError:
    def get_iterm2_window_id():
        return None


def capture_screenshot(name: str) -> str:
    os.makedirs(SCREENSHOT_DIR, exist_ok=True)
    timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
    filepath = os.path.join(SCREENSHOT_DIR, f"{name}_{timestamp}.png")
    window_id = get_iterm2_window_id()
    if window_id:
        subprocess.run(["screencapture", "-x", "-l", str(window_id), filepath], check=True)
    else:
        subprocess.run(["screencapture", "-x", filepath], check=True)
    print(f"  SCREENSHOT: {filepath}")
    return filepath


# ============================================================
# VERIFICATION HELPERS
# ============================================================

async def verify_screen_contains(session, expected: str, description: str) -> bool:
    start = time.monotonic()
    while (time.monotonic() - start) < TIMEOUT_SECONDS:
        screen = await session.async_get_screen_contents()
        for i in range(screen.number_of_lines):
            if expected in screen.line(i).string:
                return True
        await asyncio.sleep(0.2)
    return False


async def get_all_screen_text(session) -> str:
    screen = await session.async_get_screen_contents()
    lines = []
    for i in range(screen.number_of_lines):
        line = screen.line(i).string
        if line.strip():
            lines.append(line)
    return "\n".join(lines)


async def dump_screen(session, label: str):
    screen = await session.async_get_screen_contents()
    print(f"\n{'='*60}")
    print(f"SCREEN DUMP: {label}")
    print(f"{'='*60}")
    for i in range(screen.number_of_lines):
        line = screen.line(i).string
        if line.strip():
            print(f"{i:03d}: {line}")
    print(f"{'='*60}\n")


# Box-drawing character sets
BOX_CHARS = {
    'corners': '┌┐└┘╭╮╰╯╔╗╚╝',
    'horizontal': '─═━',
    'vertical': '│║┃',
    'junctions': '├┤┬┴┼╠╣╦╩╬',
}


async def verify_box_integrity(session) -> dict:
    screen = await session.async_get_screen_contents()
    issues = []
    all_box = BOX_CHARS['corners'] + BOX_CHARS['horizontal'] + BOX_CHARS['vertical']
    for i in range(screen.number_of_lines):
        line = screen.line(i).string
        for j, char in enumerate(line):
            if char in '┌╭╔':
                if j + 1 < len(line) and line[j + 1] not in BOX_CHARS['horizontal'] + BOX_CHARS['junctions']:
                    issues.append(f"Line {i}: '{char}' at col {j} not connected right")
            elif char in '┐╮╗':
                if j > 0 and line[j - 1] not in BOX_CHARS['horizontal'] + BOX_CHARS['junctions']:
                    issues.append(f"Line {i}: '{char}' at col {j} not connected left")
    return {'valid': len(issues) == 0, 'issues': issues[:5]}


async def cleanup_session(session):
    try:
        await session.async_send_text("\x03")
        await asyncio.sleep(0.2)
        await session.async_send_text("exit\n")
        await asyncio.sleep(0.2)
        await session.async_close()
    except Exception as e:
        print(f"  Cleanup warning: {e}")


# ============================================================
# MAIN TEST FUNCTION
# ============================================================

async def main(connection):
    results["start_time"] = datetime.now()

    print("\n" + "#" * 60)
    print("# ghent Comments Visual Test")
    print("#" * 60)

    app = await iterm2.async_get_app(connection)
    window = app.current_terminal_window
    if not window:
        log_result("Setup", "FAIL", "No active iTerm2 window")
        return print_summary()

    tab = await window.async_create_tab()
    session = tab.current_session

    try:
        # TEST 1: Build
        print_test_header("Build", 1)
        await session.async_send_text(f"cd {PROJECT_ROOT} && make build 2>&1; echo BUILD_EXIT=$?\n")
        await asyncio.sleep(5.0)
        if await verify_screen_contains(session, "BUILD_EXIT=0", "build success"):
            log_result("Build", "PASS")
        else:
            log_result("Build", "FAIL", "Build failed")
            await dump_screen(session, "build_failure")
            return print_summary()

        # TEST 2: Install extension
        print_test_header("Install Extension", 2)
        await session.async_send_text("gh extension install . 2>&1; echo INSTALL_EXIT=$?\n")
        await asyncio.sleep(2.0)
        if await verify_screen_contains(session, "INSTALL_EXIT=0", "install success"):
            log_result("Install", "PASS")
        else:
            # May already be installed
            log_result("Install", "UNVERIFIED", "Install returned non-zero (may already exist)")

        # TEST 3: Help output
        print_test_header("Help Output", 3)
        await session.async_send_text("gh ghent comments --help\n")
        await asyncio.sleep(1.0)
        if await verify_screen_contains(session, "Usage", "help usage"):
            screenshot = capture_screenshot("ghent_comments_help")
            log_result("Help Output", "PASS", screenshot=screenshot)
        else:
            log_result("Help Output", "FAIL", "No Usage text in help")
            await dump_screen(session, "help_missing")

        # TEST 4: Markdown output
        print_test_header("Markdown Output", 4)
        await session.async_send_text(f"gh ghent comments -R {TEST_REPO} --pr {TEST_PR}\n")
        await asyncio.sleep(3.0)
        screenshot = capture_screenshot("ghent_comments_markdown")
        screen_text = await get_all_screen_text(session)
        if len(screen_text.strip()) > 20:
            log_result("Markdown Output", "PASS", f"{len(screen_text)} chars", screenshot=screenshot)
        else:
            log_result("Markdown Output", "FAIL", "No content rendered", screenshot=screenshot)

        # TEST 5: JSON output
        print_test_header("JSON Output", 5)
        await session.async_send_text(
            f"gh ghent comments -R {TEST_REPO} --pr {TEST_PR} --format json > /tmp/ghent_test.json 2>&1; "
            "python3 -m json.tool /tmp/ghent_test.json > /dev/null 2>&1 && echo JSON_VALID || echo JSON_INVALID\n"
        )
        await asyncio.sleep(3.0)
        if await verify_screen_contains(session, "JSON_VALID", "json check"):
            log_result("JSON Output", "PASS")
        else:
            log_result("JSON Output", "FAIL", "Invalid JSON")
            await session.async_send_text("head -5 /tmp/ghent_test.json\n")
            await asyncio.sleep(0.5)
            await dump_screen(session, "invalid_json")

        # TEST 6: Layout integrity
        print_test_header("Layout Integrity", 6)
        await session.async_send_text(f"gh ghent comments -R {TEST_REPO} --pr {TEST_PR}\n")
        await asyncio.sleep(3.0)
        box_result = await verify_box_integrity(session)
        screenshot = capture_screenshot("ghent_comments_layout")
        if box_result['valid']:
            log_result("Layout Integrity", "PASS", screenshot=screenshot)
        else:
            log_result("Layout Integrity", "FAIL", box_result['issues'][0], screenshot=screenshot)

    except Exception as e:
        log_result("Execution", "FAIL", str(e))
        await dump_screen(session, "error_state")

    finally:
        await cleanup_session(session)

    return print_summary()


if __name__ == "__main__":
    exit_code = iterm2.run_until_complete(main)
    exit(exit_code if exit_code else 0)
```

---

## 9. Test Infrastructure

### Directory Layout

```
ghent/
├── .claude/
│   ├── automations/                  # iterm2-driver visual test scripts
│   │   ├── test_ghent_comments.py
│   │   ├── test_ghent_checks.py
│   │   ├── test_ghent_watch.py
│   │   ├── test_ghent_resolve.py
│   │   ├── test_ghent_pipe.py
│   │   ├── test_ghent_layout.py
│   │   ├── test_ghent_errors.py
│   │   ├── test_ghent_agent.py
│   │   └── test_ghent_install.py
│   └── screenshots/                  # Visual test screenshots (gitignored)
├── scripts/                          # Created in Phase 2-3 (not in Phase 1)
│   ├── test-binary.sh               # L3 binary execution tests
│   └── test-agent-workflow.sh        # L5 agent workflow tests
├── testdata/                         # Test fixtures for L1/L2
│   ├── review_threads_response.json
│   ├── check_runs_response.json
│   └── ...
└── internal/
    └── *_test.go                     # L1/L2 Go test files
```

### Makefile Targets

```makefile
# L1: Unit tests
test:
	go test ./...

test-race:
	go test -race ./...

coverage:
	go test -coverprofile=coverage/cover.out ./...

# L2: Integration tests (with build tags)
test-integration:
	go test -tags=integration ./...

# L3: Binary execution
test-binary: build
	bash scripts/test-binary.sh

# L4: Visual tests (requires iTerm2)
test-visual:
	@for f in .claude/automations/test_ghent_*.py; do \
		echo "=== Running $$f ==="; \
		uv run "$$f"; \
	done

# L5: Agent workflow
test-agent: build
	bash scripts/test-agent-workflow.sh

# All tests
test-all: test test-integration test-binary test-visual test-agent

# CI (no visual tests - headless)
ci: lint test test-race test-integration
```

### .gitignore Additions

```
.claude/screenshots/
/tmp/ghent_test.*
```

---

## 10. Verification Checklist Per Feature

Every feature implementation MUST complete ALL checks before being considered done.

### Template

```
## Feature: <name>

### L1: Unit Tests
- [ ] Pure function tests pass
- [ ] Table-driven test with edge cases
- [ ] `make test` green

### L2: Integration Tests
- [ ] HTTP mocked API tests pass
- [ ] Response parsing verified against testdata fixtures

### L3: Binary Execution
- [ ] `make build` succeeds
- [ ] `gh ghent <command>` runs without error
- [ ] Output matches expected format (markdown/xml/json)
- [ ] Exit code is correct (0/1/2)
- [ ] Piped output has no ANSI codes
- [ ] `-R owner/repo` flag works
- [ ] `--help` shows usage

### L4: Visual Verification
- [ ] iterm2-driver script written and passing
- [ ] Screenshot captured and visually inspected
- [ ] Box-drawing characters connected (if applicable)
- [ ] No background color bleed
- [ ] Layout matches TUI mockup in docs/tui-mockups.html

### L5: Agent Workflow
- [ ] JSON output valid and parseable
- [ ] Exit codes meaningful for automation
- [ ] Error messages actionable
- [ ] Response time <2s
```

### Feature: `gh ghent comments`

```
### L1: Unit Tests
- [ ] GraphQL response → ReviewThread parsing
- [ ] Markdown formatter: file paths, line numbers, bodies
- [ ] XML formatter: thread structure, escaping
- [ ] JSON formatter: schema matches spec
- [ ] Thread grouping by file
- [ ] Empty threads handled (no PR comments)

### L2: Integration Tests
- [ ] GraphQL reviewThreads query mocked
- [ ] Unresolved filter works
- [ ] Pagination for >100 threads
- [ ] Rate limit response handled

### L3: Binary Execution
- [ ] `gh ghent comments --pr <num>` shows threads
- [ ] `gh ghent comments --format json` valid JSON
- [ ] `gh ghent comments --format xml` valid XML
- [ ] `gh ghent comments --format markdown` readable output
- [ ] Piped output: `gh ghent comments --format json | jq .`
- [ ] `-R owner/repo` works
- [ ] Error on nonexistent PR

### L4: Visual Verification
- [ ] test_ghent_comments.py passes
- [ ] Screenshots: help, markdown, json, layout
- [ ] Box integrity check passes
- [ ] File paths and line numbers visible
- [ ] Comment bodies readable

### L5: Agent Workflow
- [ ] JSON has .threads[].file, .line, .comments[]
- [ ] Exit code 0 when no unresolved, 1 when unresolved
- [ ] <2s response time
```

### Feature: `gh ghent checks`

```
### L1: Unit Tests
- [ ] REST response → CheckRun parsing
- [ ] CI status aggregation (fail > pending > pass)
- [ ] Check status mapping (all GitHub states)
- [ ] Duration calculation
- [ ] Log filtering (failing steps only)

### L2: Integration Tests
- [ ] REST check runs endpoint mocked
- [ ] Job logs endpoint mocked
- [ ] Annotations endpoint mocked
- [ ] Error responses handled

### L3: Binary Execution
- [ ] `gh ghent checks --pr <num>` shows statuses
- [ ] `gh ghent checks --logs` shows failing logs
- [ ] `gh ghent checks --format json` valid JSON
- [ ] `gh ghent checks --watch` refreshes
- [ ] Exit code: 0=all pass, 1=any fail, 2=error

### L4: Visual Verification
- [ ] test_ghent_checks.py passes
- [ ] Status icons visible (pass/fail/pending)
- [ ] Duration shown
- [ ] Log output readable
- [ ] Watch mode countdown visible

### L5: Agent Workflow
- [ ] JSON has .checks[].name, .status, .duration
- [ ] Exit codes match: 0/1/2
- [ ] Watch mode exits on all-pass or failure
```

### Feature: `gh ghent resolve`

```
### L1: Unit Tests
- [ ] Thread ID extraction from GraphQL response
- [ ] Dry-run mode produces correct output
- [ ] File filter matches threads

### L2: Integration Tests
- [ ] GraphQL resolveReviewThread mutation mocked
- [ ] Success and error responses handled
- [ ] Multiple thread resolution

### L3: Binary Execution
- [ ] `gh ghent resolve --thread <id>` resolves on GitHub
- [ ] VERIFY IN BROWSER: thread is actually resolved
- [ ] `gh ghent resolve --all` resolves all unresolved threads
- [ ] `gh ghent resolve --unresolve --thread <id>` unresolves a thread
- [ ] Error on invalid thread ID

### L4: Visual Verification
- [ ] test_ghent_resolve.py passes (dry-run mode)
- [ ] Confirmation prompt visible
- [ ] Success/failure feedback visible

### L5: Agent Workflow
- [ ] JSON output confirms resolution
- [ ] Exit code 0 on success
- [ ] Idempotent: resolving already-resolved thread doesn't error
```

---

## 11. cm Integration

Capture testing learnings in CASS Memory for cross-session persistence.

### Capturing TUI Bug Patterns

After fixing a TUI bug, add to the playbook:

```bash
# After fixing a background bleed issue
cm playbook add "Always use termenv.SetBackgroundColor() for TUI apps - lipgloss.Background() causes bleed in empty cells" --category debugging

# After fixing modal alignment
cm playbook add "AVOID: lipgloss.Width() on inner modal elements causes padding bleed. Use strings.Repeat() manually" --category debugging

# After fixing switch variable shadowing
cm playbook add "Go switch type assertion creates LOCAL variable - always reassign to outer var after modification" --category debugging
```

### Inline Feedback During Development

```go
// [cass: helpful b-xyz] - termenv.SetBackgroundColor fixed the bleed issue

// [cass: harmful b-abc] - lipgloss.Background() made it worse, not better
```

### Session Outcome Tracking

```bash
# After a successful TUI feature implementation
cm outcome success b-xyz,b-abc --summary "ghent checks command TUI implemented with proper background handling"

# After a failed approach
cm outcome failure b-def --summary "lipgloss.Width() approach caused modal bleed, switched to manual padding"
```

### Querying Before Starting TUI Work

```bash
# Before implementing any new TUI feature
cm context "bubbletea lipgloss TUI modal rendering background" --json

# Before implementing watch mode
cm context "bubbletea tick refresh auto-update timer" --json
```

---

## Appendix: Quick Reference

### Running Tests by Level

```bash
make test               # L1 only (fast, <10s)
make test-integration   # L1 + L2 (<30s)
make test-binary        # L3 (<60s, requires gh)
make test-visual        # L4 (<120s, requires iTerm2)
make test-agent         # L5 (<30s, requires gh + test repo)
make test-all           # Everything
make ci                 # L1 + L2 + lint (headless CI)
```

### Key Paths

| Path | Purpose |
|------|---------|
| `.claude/automations/test_ghent_*.py` | Visual test scripts |
| `.claude/screenshots/` | Screenshot output (gitignored) |
| `scripts/test-binary.sh` | Binary execution tests |
| `scripts/test-agent-workflow.sh` | Agent workflow tests |
| `testdata/` | Test fixtures |
| `docs/tui-mockups.html` | Visual reference for L4 tests |

### iterm2-driver Key Methods

| Task | Method |
|------|--------|
| Get screen content | `await session.async_get_screen_contents()` |
| Send commands | `await session.async_send_text("command\n")` |
| Read line | `screen.line(i).string` |
| Take screenshot | `screencapture -x -l <window_id> path.png` |
| Create tab | `await window.async_create_tab()` |
| Close session | `await session.async_close()` |
| Special keys | Enter: `\r`, Esc: `\x1b`, Ctrl+C: `\x03`, Up: `\x1b[A` |
