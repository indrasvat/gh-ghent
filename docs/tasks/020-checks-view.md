# Task 5.3: Checks View + Log Viewer

## Status: DONE

## Depends On
- Task 4.4: Wire TUI to Cobra

## Parallelizable With
- Task 5.1: Comments list view (independent)
- Task 5.4: Resolve view (independent)

## Problem

The checks view shows CI check runs with pass/fail/pending icons, auto-expanded annotations for failing checks, and a log viewer accessible via Enter. This covers both ViewChecksList and ViewChecksLog.

## PRD Reference

- §6.3 (Checks Command) — TUI mode: check list, annotations, log viewer, key bindings
- §5.3 (TUI View Architecture) — ViewChecksList, ViewChecksLog
- Acceptance criteria: FR-CHK-01 (check list), FR-CHK-02 (auto-expand annotations), FR-CHK-03 (log viewer)

## Research References

- `docs/tui-mockups.html` — Checks tab
- `docs/testing-strategy.md` §7 — TUI pitfalls

## Files to Create

- `internal/tui/checks.go` — Checks list + log viewer models
- `internal/tui/checks_test.go` — Rendering and navigation tests
- `.claude/automations/test_ghent_checks.py` — iterm2-driver L4 visual test (canonical template from `docs/testing-strategy.md` §5)

## Files to Modify

- `internal/tui/app.go` — Register checks views

## Execution Steps

### Step 1: Read context
1. Read `docs/tui-mockups.html` — checks tab
2. Read PRD §6.3 (TUI behavior)

### Step 2: Implement checks list
- Uses `bubbles/list`
- Each item: status icon (pass ✓ / fail ✗ / pending ◌ / running ⟳), name, duration
- Failed checks auto-expand to show annotations inline (file:line + message)
- j/k navigation

### Step 3: Implement log viewer
- Enter on a check: switch to ViewChecksLog
- Uses `bubbles/viewport` for scrollable log content
- Fetches log via existing `internal/github/logs.go`
- Esc: back to checks list

### Step 4: Wire key bindings
- Tab: switch to comments view
- l: view log for selected check
- o: open check in browser
- R: re-run failed checks (via `gh.Exec("run", "rerun", ...)`)
- q: quit

### Step 5: Unit tests
- Check list renders with correct icons
- Failed checks show annotations inline
- Log viewer renders and scrolls
- Key bindings produce correct messages

## Verification

### L1: Unit Tests
```bash
make test
```

### L4: Visual (iterm2-driver)
Create `.claude/automations/test_ghent_checks.py` following canonical template in `docs/testing-strategy.md` §5:
```bash
uv run .claude/automations/test_ghent_checks.py
```
Visual assertions (per testing-strategy.md §5 `test_checks` example):
- Launch: `gh ghent checks -R indrasvat/peek-it --pr 2` → TUI renders (Lint fail, Test fail, cancelled, Build pass)
- Also test: `gh ghent checks -R indrasvat/doot --pr 1` → all passing (1 check)
- Verify: check names visible (CI, test, build patterns)
- Verify: status icons present (pass/fail/pending indicators)
- Verify: failed checks auto-expand with annotations inline
- Verify: Enter opens log viewer (viewport content changes)
- Verify: Tab switches to comments view
- Screenshots: `ghent_checks_launch.png`, `ghent_checks_annotations.png`, `ghent_checks_logviewer.png`

## Completion Criteria

1. Check list with pass/fail/pending icons
2. Failed checks auto-expand annotations
3. Enter opens log viewer (viewport)
4. Tab switches to comments
5. R re-runs failed checks
6. Layout matches `docs/tui-mockups.html`
7. `make ci` passes
8. PROGRESS.md updated

## Commit

```
feat(tui): add checks view with annotations and log viewer

- Check list with status icons and auto-expanded annotations
- Log viewer via bubbles/viewport for failing checks
- R to re-run, Tab to comments, o to open in browser
```

## Visual Test Results

**L4 iterm2-driver: 15/15 PASS** (`test_ghent_checks.py`)

Repos tested:
- `indrasvat/peek-it` PR #2 — 2 failing checks with annotations
- `indrasvat/doot` PR #1 — 1 passing check
- `indrasvat/context-lens` PR #1 — 4 passed, 2 failed (mixed)

Screenshots reviewed:
- `ghent_checks_launch.png` — Status bar shows repo/PR/SHA, "2 failed" in red, check names with ✗ icons, auto-expanded annotations with file:line and messages, duration/status on right, help bar with all key bindings
- `ghent_checks_annotations.png` — Failed checks show "N errors" header + bullet list of .github:line annotations
- `ghent_checks_logviewer.png` — Check name header with ✗ icon, duration/status, 3 annotations fully expanded, "No log excerpt available" for external CI (graceful degradation), help bar shows esc/back/scroll/browser/quit
- `ghent_checks_launch_pass.png` — Single check with green ✓, "1 passed" in green, 36s duration, clean layout
- `ghent_checks_launch_mixed.png` — "4 passed 2 failed" in status bar, both ✓ and ✗ icons, annotations only on failed checks
- `ghent_checks_tab_comments.png` — Tab switches to comments view correctly, shows "No review threads found."
- `ghent_checks_tab_back.png` — Tab returns to checks view with full check list restored

Findings:
- All status icons render correctly (✓/✗/⟳/◌)
- Auto-expanded annotations show file:line and message text
- j/k navigation moves cursor through checks
- Enter opens log viewer, Esc returns to list
- Tab cycles between checks and comments views
- Status bar shows accurate pass/fail counts
- No background bleed, ANSI issues, or layout corruption observed

## Session Protocol

1. Read CLAUDE.md
2. Read this task file
3. **Change this task's status to `IN PROGRESS`**
4. Open `docs/tui-mockups.html` for visual reference
5. Read PRD §6.3
6. Execute steps 1-5
7. Run verification (L1 → L4)
8. **Change this task's status to `DONE`**
9. Update `docs/PROGRESS.md`
10. Update CLAUDE.md Learnings if needed
11. Commit
