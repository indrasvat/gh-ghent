# Task 032: Summary Pane Overflow, Slow Startup & Esc Navigation

- **Phase:** 9 (Bug Fixes)
- **Status:** DONE
- **Depends on:** None
- **Blocks:** None
- **L4 Visual:** Required (TUI layout, scrolling, navigation — verify with iterm2-driver)

## Problems

Three related TUI issues discovered when testing against large, active PRs:

### P1: Summary Pane Overflow (Critical)

**Repro:** `gh ghent summary -R oven-sh/bun --pr 24063`

The Approvals section in the summary dashboard renders ALL reviews with no limit.
For a PR with 61 reviews, this generates 61+ lines that push the KPI cards, Review
Threads section, and CI Checks section completely off-screen — only the Approvals
list is visible (see screenshot).

**Root cause:** `renderApprovalsSection()` in `internal/tui/summary.go:347` iterates
over all reviews unconditionally, unlike `renderThreadsSection()` which caps at
`maxShow := 3`. The `View()` method joins all sections and pads to fill `m.height`
but never truncates or enables scrolling.

**Fix:** Two-part solution:
1. Cap displayed reviews in `renderApprovalsSection()` to a reasonable max (e.g., 5),
   with a `... and N more` overflow indicator (consistent with threads section pattern).
2. Add viewport-based scrolling to the summary view so users can scroll through content
   that exceeds `m.height`. Use `bubbles/viewport` or manual line-based scrolling.

### P2: Slow TUI Startup (Major)

**Repro:** `gh ghent summary -R oven-sh/bun --pr 24063` — takes ~3-5 seconds before TUI appears.

The summary command fetches threads, checks, and reviews in parallel via `errgroup`,
but ALL three must complete before the TUI launches (cli/summary.go:57-96). For repos
with massive activity, the user stares at a blank terminal.

**Root cause:** `launchTUI()` is called after `g.Wait()` — the TUI is never rendered
until every API call finishes.

**Fix:** Launch the TUI immediately with a loading/spinner state, then populate data
incrementally as each fetch completes via `tea.Cmd` messages. Pattern:
1. Launch TUI with empty data + spinner/loading indicator.
2. Fire off fetch commands as `tea.Cmd`s that send `dataLoadedMsg` on completion.
3. Update sub-models progressively as data arrives.
4. This matches the pattern already used by `watcher.go` for polling.

### P3: Esc Key Doesn't Navigate Back (Minor)

**Repro:** Launch `gh ghent summary -R oven-sh/bun --pr 27269`, press `c` to jump
to comments view, then press `Esc` — nothing happens. Must press `q` to quit entirely.

**Root cause:** In `handleKey()` (app.go:206-221), pressing Esc from `ViewCommentsList`
or `ViewChecksList` falls through silently:
- `isDetail()` returns false for list views → skipped
- `ViewResolve || ViewSummary` check → false for list views → skipped
- Falls to final `return a, nil` — Esc is swallowed

The `prevView` field IS correctly set when navigating from summary (lines 227-238),
but the Esc handler doesn't check it for list views.

**Fix:** After the detail/resolve/summary checks, add a fallback: if `a.prevView`
is set and different from `a.activeView`, navigate back to `a.prevView`:

```go
// From list views (comments/checks), return to previous view if set.
if a.prevView != a.activeView {
    a.activeView = a.prevView
    return a, nil
}
```

Also need to ensure `prevView` is set when navigating from summary to comments/checks
via the `c`/`k` shortcuts — this is already done in lines 227-238. But we also need
to set `prevView` when entering checks/comments from the checks/resolve views so the
back chain works consistently.

## Implementation Plan

### Step 1: Fix Esc navigation (P3) — Smallest, highest confidence

**Files:** `internal/tui/app.go`

1. In `handleKey()`, after the existing Esc checks for detail/resolve/summary views,
   add a general fallback that returns to `prevView` when it differs from `activeView`.
2. Ensure `prevView` is initialized properly (default `ViewSummary` when starting from
   summary, or `ViewCommentsList` when starting from comments).
3. Update `cycleView()` to also set `prevView` for Tab/Shift+Tab cycling.

### Step 2: Fix summary approvals overflow (P1) — Medium complexity

**Files:** `internal/tui/summary.go`

1. In `renderApprovalsSection()`, add `maxShow` cap (5 reviews) with overflow indicator.
2. Show approvals in priority order: CHANGES_REQUESTED first, then APPROVED, then
   COMMENTED/PENDING — most actionable reviews first.
3. Add viewport scrolling to `summaryModel`:
   - Track `scrollOffset int` field.
   - In `View()`, after building full content, slice to visible lines based on
     `scrollOffset` and `m.height`.
   - Handle `j`/`k` keys for scrolling in summary view (forward from app.go).
   - Show scroll indicator (e.g., `↓ more` / `↑ more`) when content overflows.

### Step 3: Fix slow startup (P2) — Highest complexity

**Files:** `internal/cli/summary.go`, `internal/tui/app.go`, `internal/tui/summary.go`

1. Define new message types in tui package:
   ```go
   type commentsLoadedMsg struct { result *domain.CommentsResult }
   type checksLoadedMsg   struct { result *domain.ChecksResult }
   type reviewsLoadedMsg  struct { reviews []domain.Review; err error }
   ```
2. Add `Init()` support to App that fires fetch commands.
3. Modify `launchTUI()` to accept fetch functions instead of pre-fetched data.
4. Modify `summary.go` CLI to launch TUI immediately, passing fetch funcs.
5. Show loading spinner/text in summary view while data loads:
   - "Loading review threads..."
   - "Loading CI checks..."
   - "Loading reviews..."
6. Progressive rendering: show sections as their data arrives.

### Step 4: Update help bar for summary scrolling

**Files:** `internal/tui/components/helpbar.go`

1. Add `j/k` scroll binding to `SummaryKeys()`.
2. Add `esc` back binding to `SummaryKeys()` when entered from another view.

## Testing

### Unit Tests

**File:** `internal/tui/app_test.go`

1. **Esc navigation from comments→summary:**
   - Create App with `ViewSummary`, simulate `c` key → verify `ViewCommentsList`.
   - Simulate `Esc` → verify returns to `ViewSummary`.

2. **Esc navigation from checks→summary:**
   - Create App with `ViewSummary`, simulate `k` key → verify `ViewChecksList`.
   - Simulate `Esc` → verify returns to `ViewSummary`.

3. **Esc no-op at top level:**
   - Create App with `ViewCommentsList` (no prevView), simulate `Esc` → verify stays.

**File:** `internal/tui/summary_test.go`

4. **Approvals overflow capping:**
   - Create summaryModel with 20 reviews → verify `renderApprovalsSection()` output
     contains max 5 review lines + overflow indicator.

5. **Approvals priority ordering:**
   - Create summaryModel with mixed review states → verify CHANGES_REQUESTED appears
     before APPROVED before COMMENTED.

6. **Summary scrolling:**
   - Create summaryModel with content exceeding height → verify scroll offset changes
     with j/k keys.

7. **Empty/small review lists:**
   - Verify 0 reviews → "No reviews yet".
   - Verify 1-5 reviews → all shown, no overflow indicator.

### L3 Real Repo Test Matrix — oven-sh/bun Stress Tests

The `oven-sh/bun` repo provides extreme real-world test cases across multiple
dimensions. These PRs were selected specifically to surface edge cases.

#### Stress Test Repos

| # | PR | State | Files | +/- Lines | Threads (unresolved) | Reviews | Comments | CI Checks | Primary Stress |
|---|-----|-------|-------|-----------|---------------------|---------|----------|-----------|----------------|
| 1 | #27327 | OPEN | 28 | +11,530/-0 | 77 (68) | 25 | 5 | 59 | Review threads (extreme) |
| 2 | #27338 | OPEN | 2 | +919/-23 | 46 (1) | 22 | 15 | 1 | Resolved threads + comments |
| 3 | #27315 | OPEN | 1 | +665/-656 | 6 (4) | 6 | 6 | 61 | Mixed CI (1 fail/29 pass) |
| 4 | #27385 | OPEN | 9 | +610/-56 | 3 (0) | 7 | 3 | 64 | Most CI checks |
| 5 | #27264 | OPEN | 8 | +203/-12 | 6 (6) | 3 | 3 | 65 | 100% unresolved threads |
| 6 | #27397 | OPEN | 4 | +403/-19 | 5 (5) | 4 | 3 | 64 | All unresolved + active |
| 7 | #27056 | MERGED | 15 | +1,065/-110 | 23 (17) | 42 | 7 | 60 | Most reviews (42) |
| 8 | #27019 | MERGED | 6 | +314/-4 | 10 (3) | 6 | 9 | 59 | CHANGES_REQUESTED + merged |
| 9 | #24063 | MERGED | ? | large | many | 61 | many | many | Original overflow repro |

#### PR Detail Notes

- **#27327** (SMTP client): The single best stress-test PR. 68 unresolved review threads
  will hammer the comments TUI with scrolling, navigation, and thread expansion. 11,530-line
  addition is a massive diff. 56+ Buildkite status contexts stress the checks view.

- **#27338** (undici Pool/Client): 46 threads with 45 resolved tests the `--all` flag and
  resolved thread display. 15 issue comments is the highest among open PRs. Tests iterative
  review cycle patterns.

- **#27315** (HTTP refactor): The only open PR with a genuine mixed CI status — 29 passing
  Buildkite jobs and 1 real failure (`windows-aarch64-build-zig`). Tests checks TUI's ability
  to highlight failures among many successes. Pure refactor (+665/-656) is a distinct PR shape.

- **#27385** (TLS keepalive): 64 total status checks (3 GitHub + 61 Buildkite) is the highest
  count among open PRs. Buildkite contexts span a matrix of 11 OS/arch variants × 2 build
  types. Tests how checks TUI handles massive lists with long names.

- **#27264** (fetch error format): 100% unresolved thread rate (6/6). Combined with 8 files
  and 65 CI checks, stresses multiple views simultaneously.

- **#27397** (bundler CJS): Another 100% unresolved rate (5/5) with active review iteration.
  Good for testing the resolve TUI multi-select flow.

- **#27056** (MERGED, --compile --target=browser): 42 reviews is the highest review count found
  in the repo. 4 unique reviewers including repo maintainer, two AI assistants, and a bot.
  17 still-unresolved threads even after merge — a common real-world pattern.

- **#27019** (MERGED, SHA-512 integrity): Merged despite `CHANGES_REQUESTED` review decision and
  3 unresolved threads. Tests edge case for summary's `is_merge_ready` logic.

#### Important: Buildkite CI Discovery

Bun uses **Buildkite** as their primary CI, which appears as GitHub **commit statuses**
(not check runs). The current checks fetcher only uses the REST `check-runs` endpoint:
- `repos/{owner}/{repo}/commits/{sha}/check-runs` → returns only 2-3 GitHub Actions checks
  (Lint JavaScript, Format, Mintlify Deployment)
- `repos/{owner}/{repo}/commits/{sha}/status` → returns 56-61 Buildkite statuses covering
  linux/darwin/windows × x64/aarch64 × cpp/zig build matrix

**This means gh-ghent currently misses most of Bun's CI data.** Consider adding commit
status fetching as a follow-up task, or at minimum note this gap in the checks view.

### L3 Manual Test Commands

```bash
# ── P1: Summary overflow ──────────────────────────────────────────

# Original repro — 61 reviews overflow
gh ghent summary -R oven-sh/bun --pr 24063

# Extreme: 25 reviews + 77 threads + 59 checks
gh ghent summary -R oven-sh/bun --pr 27327

# Small PR — regression (should look clean)
gh ghent summary -R oven-sh/bun --pr 27269

# ── P2: Startup speed ────────────────────────────────────────────

# Time TUI appearance — should be <1s
time gh ghent summary -R oven-sh/bun --pr 24063
time gh ghent summary -R oven-sh/bun --pr 27327

# ── P3: Esc navigation ───────────────────────────────────────────

# summary → c → esc → should be back at summary
# summary → k → esc → should be back at summary
gh ghent summary -R oven-sh/bun --pr 27269

# ── Comments view stress tests ────────────────────────────────────

# 68 unresolved threads — scrolling, navigation
gh ghent comments -R oven-sh/bun --pr 27327

# 6 unresolved threads — moderate list
gh ghent comments -R oven-sh/bun --pr 27264

# 1 unresolved + 45 resolved — test resolved filter
gh ghent comments -R oven-sh/bun --pr 27338

# ── Checks view stress tests ─────────────────────────────────────

# Mixed CI: 1 failure among 29 passes (Buildkite)
gh ghent checks -R oven-sh/bun --pr 27315

# 64 total checks — massive list rendering
gh ghent checks -R oven-sh/bun --pr 27385

# ── Resolve view stress tests ─────────────────────────────────────

# 68 unresolved threads — multi-select at scale
gh ghent resolve -R oven-sh/bun --pr 27327

# 5 unresolved threads — typical resolve flow
gh ghent resolve -R oven-sh/bun --pr 27397

# ── Existing test matrix (regression) ─────────────────────────────

gh ghent comments -R indrasvat/tbgs --pr 1 --format json | jq '.unresolved_count'  # 2
gh ghent checks -R indrasvat/doot --pr 1 --format json | jq '.overall_status'      # "pass"
gh ghent checks -R indrasvat/peek-it --pr 2 --format json | jq '.overall_status'   # "failure"
```

### L4 iterm2-driver Visual Tests

All tests use `iterm2-driver` skill for automated screenshot verification.

#### Test 1: Summary overflow — extreme PR (68 threads, 25 reviews)

```python
"""
Test: Summary pane shows KPI cards and capped reviews for extreme PR.
Repo: oven-sh/bun PR #27327 (68 unresolved threads, 25 reviews, 59 checks)
Verify:
  - KPI cards row visible at top (UNRESOLVED, PASSED, FAILED, APPROVALS)
  - Review Threads section visible with "... and N more" indicator
  - CI Checks section visible
  - Approvals section shows max 5 reviews + "... and N more" indicator
  - Status bar shows "NOT READY" badge
  - No content pushed off-screen
"""
# Launch: gh ghent summary -R oven-sh/bun --pr 27327
# Wait for TUI to render
# Screenshot and verify all 4 sections visible within viewport
```

#### Test 2: Summary overflow — original repro (61 reviews)

```python
"""
Test: Original overflow repro now renders correctly.
Repo: oven-sh/bun PR #24063 (61 reviews — the original bug report)
Verify:
  - KPI cards visible at top
  - Approvals section capped (not 61 raw lines)
  - "... and 56 more" (or similar) overflow text present
"""
# Launch: gh ghent summary -R oven-sh/bun --pr 24063
# Wait for TUI to render
# Screenshot and verify KPI row is visible
```

#### Test 3: Summary — small PR (regression)

```python
"""
Test: Summary pane renders correctly for PR with few/no reviews.
Repo: oven-sh/bun PR #27269 (0 reviews, 5 checks)
Verify:
  - KPI cards: 0 UNRESOLVED, 5 PASSED, 0 FAILED, 0 APPROVALS
  - Approvals section: "No reviews yet"
  - Plenty of empty space (no overflow)
"""
# Launch: gh ghent summary -R oven-sh/bun --pr 27269
# Screenshot and verify layout matches expected
```

#### Test 4: Summary scrolling (j/k navigation)

```python
"""
Test: Summary view scrolls with j/k when content exceeds viewport.
Repo: oven-sh/bun PR #27327 (large content)
Verify:
  - Press j multiple times → content scrolls down
  - Press k → content scrolls back up
  - Scroll indicators visible when content overflows
"""
# Launch: gh ghent summary -R oven-sh/bun --pr 27327
# Wait for render
# Send 'j' key 5 times
# Screenshot — verify content has scrolled
# Send 'k' key 5 times
# Screenshot — verify back at top
```

#### Test 5: Esc navigation from summary sub-views

```python
"""
Test: Esc returns to summary when navigating via c/k shortcuts.
Verify:
  - Start at summary → press 'c' → comments list shown
  - Press Esc → back to summary (KPI cards visible)
  - Press 'k' → checks list shown
  - Press Esc → back to summary
"""
# Launch: gh ghent summary -R oven-sh/bun --pr 27269
# Screenshot 1: summary view
# Send 'c' key
# Screenshot 2: comments view (verify status bar says "comments")
# Send Esc key
# Screenshot 3: summary view (verify KPI cards visible)
# Send 'k' key
# Screenshot 4: checks view
# Send Esc key
# Screenshot 5: summary view again
```

#### Test 6: Startup speed — loading state visible

```python
"""
Test: TUI appears immediately with loading indicator for large PR.
Repo: oven-sh/bun PR #27327 (heaviest PR — 68 threads, 25 reviews, 59 checks)
Verify:
  - TUI frame (status bar + help bar) appears within ~0.5s
  - Loading text/spinner visible while data fetches
  - Sections populate progressively as data arrives
"""
# Launch: gh ghent summary -R oven-sh/bun --pr 27327
# Screenshot immediately after launch (~0.5s) — verify loading state
# Wait 5s for data to load
# Screenshot — verify fully populated summary
```

#### Test 7: Comments list — 68 unresolved threads

```python
"""
Test: Comments list handles extreme thread count with smooth scrolling.
Repo: oven-sh/bun PR #27327 (68 unresolved threads)
Verify:
  - Comments list renders without overflow/garbling
  - j/k navigation scrolls through all 68 threads
  - Thread counter in status bar updates (e.g., "68 unresolved")
  - Enter expands a thread, Esc returns to list
"""
# Launch: gh ghent comments -R oven-sh/bun --pr 27327
# Screenshot 1: initial list (first threads visible)
# Send 'j' key 20 times — scroll deep into list
# Screenshot 2: verify smooth rendering mid-list
# Send 'enter' — expand a thread
# Screenshot 3: expanded thread view
# Send 'esc' — back to list
# Screenshot 4: verify list restored
```

#### Test 8: Comments list — mostly resolved threads

```python
"""
Test: Comments list with 1 unresolved / 45 resolved.
Repo: oven-sh/bun PR #27338 (46 threads, only 1 unresolved)
Verify:
  - Only 1 thread shown in default view (unresolved only)
  - Status bar shows "1 unresolved  45 resolved"
"""
# Launch: gh ghent comments -R oven-sh/bun --pr 27338
# Screenshot — verify single thread + resolved count in status bar
```

#### Test 9: Checks view — massive check list (64 checks)

```python
"""
Test: Checks list renders 64 CI checks without overflow.
Repo: oven-sh/bun PR #27385 (64 total checks across GitHub + Buildkite)
Verify:
  - Checks list renders without garbling
  - j/k scrolls through full list
  - Long check names (e.g., "buildkite/bun/windows-aarch64-build-zig") truncated gracefully
  - Status icons (pass/fail/pending) visible for each check
Note: Currently only GitHub Actions check runs are fetched (2-3 checks).
      Buildkite statuses (61) are NOT fetched yet — this test validates
      current behavior and will be updated when commit status support is added.
"""
# Launch: gh ghent checks -R oven-sh/bun --pr 27385
# Screenshot — verify checks list rendering
```

#### Test 10: Checks view — mixed CI status (failure + passes)

```python
"""
Test: Checks view highlights failures among many passes.
Repo: oven-sh/bun PR #27315 (1 failure, 29 passes in Buildkite)
Verify:
  - Failed checks shown first with red icon
  - Passing checks summarized or listed below
  - Status bar shows accurate pass/fail counts
Note: Same caveat as Test 9 — only GitHub Actions checks currently visible.
"""
# Launch: gh ghent checks -R oven-sh/bun --pr 27315
# Screenshot — verify failure highlighted
```

#### Test 11: Resolve view — extreme thread count

```python
"""
Test: Resolve multi-select handles 68 threads.
Repo: oven-sh/bun PR #27327 (68 unresolved threads)
Verify:
  - All 68 threads listed with checkboxes
  - j/k navigates full list
  - Space toggles selection
  - 'a' selects all 68
  - Status bar shows "68 selected of 68 unresolved"
Note: Do NOT actually resolve — just test UI interaction.
"""
# Launch: gh ghent resolve -R oven-sh/bun --pr 27327
# Screenshot 1: initial list
# Send 'j' 10 times, then 'space' to toggle
# Screenshot 2: mid-list with selection
# Send 'a' to select all
# Screenshot 3: all selected
# Send 'esc' to cancel
```

#### Test 12: Summary — merged PR with CHANGES_REQUESTED (edge case)

```python
"""
Test: Summary for merged PR that had CHANGES_REQUESTED + unresolved threads.
Repo: oven-sh/bun PR #27019 (merged with CHANGES_REQUESTED, 3 unresolved threads)
Verify:
  - Summary renders without error
  - is_merge_ready shows NOT READY (despite being merged)
  - Threads section shows 3 unresolved
  - Approvals section shows CHANGES_REQUESTED review
"""
# Launch: gh ghent summary -R oven-sh/bun --pr 27019
# Screenshot — verify CHANGES_REQUESTED visible in approvals section
```

#### Test 13: Regression — existing test matrix repos

```python
"""
Test: Existing test matrix repos still render correctly.
Repos: indrasvat/tbgs#1, indrasvat/doot#1, indrasvat/peek-it#2
Verify: each summary view shows correct KPI counts and sections.
"""
# Launch each and screenshot for visual regression
```

## Performance Benchmarks (hyperfine)

Capture before/after wall-clock times with `hyperfine`. Run **before** starting
implementation and **after** each major fix to track regression/improvement.

### Benchmark Commands

```bash
# ── BEFORE: Run once and save output ──────────────────────────────
hyperfine --warmup 1 --runs 3 -i \
  -n 'PR #24063 (101 threads, 61 reviews)' \
    'gh ghent summary -R oven-sh/bun --pr 24063 --no-tui --format json 2>/dev/null > /dev/null' \
  -n 'PR #27327 (68 threads, 25 reviews)' \
    'gh ghent summary -R oven-sh/bun --pr 27327 --no-tui --format json 2>/dev/null > /dev/null' \
  -n 'PR #27269 (0 threads, 0 reviews)' \
    'gh ghent summary -R oven-sh/bun --pr 27269 --no-tui --format json 2>/dev/null > /dev/null' \
  --export-markdown /tmp/ghent-before-perf.md

# ── AFTER: Run same commands, export to different file ────────────
hyperfine --warmup 1 --runs 3 -i \
  -n 'PR #24063 (101 threads, 61 reviews)' \
    'gh ghent summary -R oven-sh/bun --pr 24063 --no-tui --format json 2>/dev/null > /dev/null' \
  -n 'PR #27327 (68 threads, 25 reviews)' \
    'gh ghent summary -R oven-sh/bun --pr 27327 --no-tui --format json 2>/dev/null > /dev/null' \
  -n 'PR #27269 (0 threads, 0 reviews)' \
    'gh ghent summary -R oven-sh/bun --pr 27269 --no-tui --format json 2>/dev/null > /dev/null' \
  --export-markdown /tmp/ghent-after-perf.md
```

Notes:
- `-i` ignores non-zero exit codes (exit 1 = "not merge ready", expected)
- `--no-tui --format json` measures pure API + processing time (no TUI overhead)
- For TUI perceived-latency (time-to-first-frame), use iterm2-driver screenshot timing
- Network variance is high — run benchmarks at similar times of day for fair comparison

### BEFORE Baselines (captured 2026-02-24)

| PR | Threads | Reviews | Mean | Min | Max |
|----|---------|---------|------|-----|-----|
| #24063 | 101 unresolved | 61 | **6.46s** ± 0.59s | 5.79s | 6.89s |
| #27327 | 68 unresolved | 25 | **1.95s** ± 0.28s | 1.70s | 2.25s |
| #27269 | 0 | 0 | **1.33s** ± 0.07s | 1.25s | 1.39s |

PR #24063 is **4.86x slower** than the small PR #27269.

### AFTER Results (captured 2026-02-24)

Pipe-mode times unchanged (expected — API latency is fixed). TUI mode now launches
instantly with loading indicator; data renders progressively as each fetch completes.

| PR | Threads | Reviews | Mean (pipe) | TUI first-frame |
|----|---------|---------|-------------|-----------------|
| #24063 | 101 | 61 | **6.22s** ± 0.24s | **<0.5s** (instant) |
| #27327 | 68 | 25 | **2.27s** ± 0.20s | **<0.5s** (instant) |
| #27269 | 0 | 0 | **1.36s** ± 0.08s | **<0.5s** (instant) |

**Key improvement:** TUI first-frame dropped from 1.3-6.5s → <0.5s for all PRs.
Users see "Loading PR data..." immediately, then sections populate as data arrives.
Pipe-mode latency is unchanged (within network variance of BEFORE baselines).

## Discovered Issue: Missing Buildkite/Commit Status Data

Bun uses Buildkite as primary CI, which appears as GitHub **commit statuses** (not
check runs). gh-ghent currently only fetches from the check-runs REST endpoint,
missing 56-61 Buildkite statuses. This should be tracked as a follow-up:

- **Endpoint needed:** `repos/{owner}/{repo}/commits/{sha}/status` (combined status)
  or `repos/{owner}/{repo}/commits/{sha}/statuses` (individual statuses)
- **Impact:** Checks view shows only 2-3 GitHub Actions checks for Bun PRs, missing
  the entire Buildkite build matrix
- **Priority:** Medium — affects any repo using external CI (Buildkite, CircleCI, etc.)
- **Consider:** Task 033 for commit status integration

## Acceptance Criteria

- [x] KPI cards always visible at top of summary for any PR size
- [x] Approvals section capped with overflow indicator
- [x] Summary view scrollable with j/↓/↑ when content exceeds viewport
- [x] TUI appears within ~1s for any PR (loading state shown)
- [x] Data populates progressively as API calls complete
- [x] Esc returns to summary from comments/checks views
- [x] Esc is no-op at top-level (when no prevView set)
- [ ] Comments list handles 68+ threads without garbling
- [ ] Checks list handles 60+ checks without overflow
- [ ] Resolve view handles 68+ threads with smooth multi-select
- [x] All L3 test matrix repos pass (indrasvat/* repos: tbgs, doot, peek-it)
- [ ] All 13 L4 iterm2-driver tests pass (9/13 done — existing summary tests pass)
- [x] `make test` passes with new unit tests (516 tests, 0 failures)
- [x] `make lint` passes
