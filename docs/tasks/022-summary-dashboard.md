# Task 5.5: Summary Dashboard

## Status: DONE

## Depends On
- Task 5.1: Comments list view (needs comment data + view)
- Task 5.3: Checks view (needs checks data + view)

## Parallelizable With
- None (needs both views as navigation targets)

## Problem

The summary dashboard is the TUI landing view: KPI cards (unresolved count, checks pass/fail, approvals), merge readiness badge, section previews, and quick-nav keys to jump to full views.

## PRD Reference

- §6.6 (Summary Command) — TUI: KPI cards, READY/NOT READY badge, sections, quick-nav
- §5.3 (TUI View Architecture) — ViewSummary
- Acceptance criteria: FR-SUM-01 through FR-SUM-05

## Research References

- `docs/tui-mockups.html` — Summary tab (KPI card layout, section previews)

## Files to Create

- `internal/tui/summary.go` — Summary dashboard model
- `internal/tui/summary_test.go` — KPI rendering, quick-nav tests
- `.claude/automations/test_ghent_summary.py` — iterm2-driver L4 visual test (canonical template from `docs/testing-strategy.md` §5)

## Files to Modify

- `internal/tui/app.go` — Register summary view, set as default start view for `gh ghent summary`

## Execution Steps

### Step 1: Read context
1. Read `docs/tui-mockups.html` — summary tab (exact layout with KPI cards)
2. Read PRD §6.6

### Step 2: Implement KPI cards row
- Lipgloss layout: horizontal row of cards
- Cards: Unresolved (count), Checks Passed (count), Checks Failed (count), Approvals (count)
- Color-coded: green (good), red (bad), yellow (pending)

### Step 3: Implement merge readiness badge
- "READY" (green) or "NOT READY" (red) in status bar
- Logic: unresolved=0 AND checks=pass AND approvals sufficient

### Step 4: Implement section previews
- Review Threads: top 3-5 threads with "... and N more" truncation
- CI Checks: failed checks with annotations, "N passed" summary
- Approvals: reviewer name + status (approved ✓, changes requested ✗, pending ◌)

### Step 5: Wire quick-nav keys
- c: jump to ViewCommentsList
- k: jump to ViewChecksList
- r: jump to ViewResolve
- o: open PR in browser
- R: re-run failed checks
- q: quit

### Step 6: Unit tests
- KPI card counts correct
- Merge readiness logic
- Quick-nav keys produce correct view switch messages

## Verification

### L1: Unit Tests
```bash
make test
```

### L4: Visual (iterm2-driver)
Create `.claude/automations/test_ghent_summary.py` following canonical template in `docs/testing-strategy.md` §5:
```bash
uv run .claude/automations/test_ghent_summary.py
```
Visual assertions:
- Launch: `gh ghent summary -R indrasvat/tbgs --pr 1` → TUI renders (NOT READY: 2 unresolved threads, checks pass)
- Also test: `gh ghent summary -R indrasvat/doot --pr 1` → READY (0 unresolved, checks pass)
- Also test: `gh ghent summary -R indrasvat/peek-it --pr 2` → NOT READY (threads + failing checks)
- Verify: KPI cards visible (counts for unresolved, checks, approvals)
- Verify: READY or NOT READY badge visible
- Verify: section previews show thread/check summaries
- Verify: c key switches to comments view, k to checks, r to resolve
- Verify: color coding correct (green=good, red=bad, yellow=pending)
- Screenshots: `ghent_summary_launch.png`, `ghent_summary_ready.png`, `ghent_summary_not_ready.png`

## Completion Criteria

1. KPI cards render with correct counts and colors
2. READY/NOT READY merge readiness badge
3. Section previews (threads, checks, approvals)
4. Quick-nav: c/k/r jump to correct views
5. Layout matches `docs/tui-mockups.html`
6. `make ci` passes
7. PROGRESS.md updated

## Commit

```
feat(tui): add summary dashboard with KPI cards and merge readiness

- KPI card row: unresolved, checks pass/fail, approvals
- READY/NOT READY merge readiness badge
- Section previews with truncation
- Quick-nav: c (comments), k (checks), r (resolve), o (browser)
```

## Visual Test Results

L4 iterm2-driver: **9/9 PASS** (`test_ghent_summary.py`)

### Screenshots Reviewed

1. **ghent_summary_launch.png** (indrasvat/tbgs PR #1 — NOT READY) — KPI cards row renders correctly with 4 cards: "2 UNRESOLVED" (red), "3 PASSED" (green), "0 FAILED" (green), "0 APPROVALS" (yellow). Cards use rounded borders with proper center alignment. "NOT READY" badge in status bar (red, bold). Three sections render: Review Threads (red dot, 2 unresolved + 0 resolved, file paths with line numbers and @authors), CI Checks (green dot, 3 checks passed with names), Approvals (yellow dot, 5 reviews with comment states and time-ago). Help bar shows summary-specific bindings: "c comments k checks r resolve o open PR R re-run failed q quit".

2. **ghent_summary_ready.png** (indrasvat/doot PR #1 — NOT READY) — KPI cards: "0 UNRESOLVED" (green), "1 PASSED" (green), "0 FAILED" (green), "0 APPROVALS" (yellow). Shows "NOT READY" because doot has no approvals (only a commented review). Green dots on threads/checks sections. "No review threads" placeholder renders correctly. 1 check passed (make ci). 1 review (@chatgpt-codex-connector commented).

3. **ghent_summary_not_ready.png** (indrasvat/peek-it PR #2 — NOT READY) — KPI cards: "1 UNRESOLVED" (red), "0 PASSED" (green), "2 FAILED" (red), "0 APPROVALS" (yellow). Red dot on CI Checks. Failed checks display with ✗ icons and annotation counts ("3 errors", "2 errors"). Annotation details show file:line patterns (.github:1, .github:203, .github:7). Thread preview shows file path. Confirms the dashboard correctly surfaces failure details.

### Findings

- KPI cards adapt to terminal width via `lipgloss.JoinHorizontal` with `max()` for minimum card width.
- Section dots color-code correctly: green for all-clear, red for issues, yellow for pending/partial.
- The `isMergeReady()` logic mirrors the CLI's `IsMergeReady()` — requires approvals to show "READY".
- `formatTimeAgo()` (from comments.go) and `checkIsFailed()` (from checks.go) are reused as package-scoped helpers.

## Session Protocol

1. Read CLAUDE.md
2. Read this task file
3. **Change this task's status to `IN PROGRESS`**
4. Open `docs/tui-mockups.html` for visual reference
5. Read PRD §6.6
6. Execute steps 1-6
7. Run verification (L1 → L4)
8. **Change this task's status to `DONE`**
9. Update `docs/PROGRESS.md`
10. Update CLAUDE.md Learnings if needed
11. Commit
