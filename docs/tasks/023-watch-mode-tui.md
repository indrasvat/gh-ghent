# Task 5.6: Watch Mode TUI

## Status: IN PROGRESS

## Depends On
- Task 5.3: Checks view (needs check display as base)

## Parallelizable With
- None (depends on checks view)

## Problem

The `--watch` flag in TUI mode needs a spinner, progress bar, elapsed timer, and event log showing real-time check status updates. Fail-fast behavior surfaces failures immediately.

## PRD Reference

- §6.7 (Watch Mode) — TUI: spinner, progress bar, elapsed time, event log, fail-fast
- §5.3 (TUI View Architecture) — ViewWatch
- Acceptance criteria: FR-WAT-01 through FR-WAT-05

## Research References

- `docs/tui-mockups.html` — Checks --watch tab
- `docs/vivecaka-large-pr-patterns-research.md` §11 — Auto-refresh pattern

## Files to Create

- `internal/tui/watcher.go` — Watch mode TUI model
- `internal/tui/watcher_test.go` — Watcher rendering tests
- `.claude/automations/test_ghent_watch.py` — iterm2-driver L4 visual test (canonical template from `docs/testing-strategy.md` §5)

## Files to Modify

- `internal/tui/app.go` — Register watch view
- `internal/cli/checks.go` — Route `--watch` + TTY to watch TUI view

## Execution Steps

### Step 1: Read context
1. Read `docs/tui-mockups.html` — checks --watch tab
2. Read PRD §6.7

### Step 2: Implement watch model
- `bubbles/spinner` with braille dot animation
- Progress bar: `completed/total` checks (custom Lipgloss rendering)
- Elapsed time counter (updated every second via `tea.Tick`)
- Event log at bottom: `bubbles/viewport` showing timestamped status updates

### Step 3: Implement polling
- `tea.Tick` every 10s triggers API re-fetch
- On each poll: update progress bar, add event to log
- Check status changes logged: "14:32:15 ✓ lint passed (12s)"

### Step 4: Implement fail-fast
- Any check fails → immediately expand to show error details
- Highlight failed check in event log with red
- Auto-transition to ViewChecksList with failed check focused

### Step 5: Handle exit conditions
- All pass → show success banner, exit code 0
- Failure → show failure details, exit code 1
- Ctrl+C → clean exit, exit code 130

### Step 6: Unit tests
- Spinner updates on tick
- Progress bar calculates correctly
- Event log accumulates entries
- Fail-fast triggers on failure

## Verification

### L1: Unit Tests
```bash
make test
```

### L4: Visual (iterm2-driver)
Create `.claude/automations/test_ghent_watch.py` following canonical template in `docs/testing-strategy.md` §5:
```bash
uv run .claude/automations/test_ghent_watch.py
```
Visual assertions (per testing-strategy.md §5 `test_watch_mode` example):
- Launch: `gh ghent checks --watch -R indrasvat/doot --pr 1` → TUI renders (checks already complete, should show pass immediately)
- Also test: `gh ghent checks --watch -R indrasvat/peek-it --pr 2` → shows failure state
- Verify: spinner animation visible (braille dots)
- Verify: progress bar shows completed/total
- Verify: event log accumulates timestamped entries
- Verify: after poll interval (wait 11s), screen content updates
- Verify: q exits cleanly (alt screen cleared)
- Screenshots: `ghent_watch_initial.png`, `ghent_watch_refreshed.png`, `ghent_watch_exit.png`
- Also useful: trigger a real CI run (e.g., push to gh-ghent) and watch with in-progress checks

## Completion Criteria

1. Spinner animation active during polling
2. Progress bar shows completed/total
3. Elapsed time counter ticks
4. Event log shows timestamped status updates
5. Fail-fast: failure → expand details immediately
6. Ctrl+C exits cleanly
7. Layout matches `docs/tui-mockups.html`
8. `make ci` passes
9. PROGRESS.md updated

## Commit

```
feat(tui): add watch mode with spinner, progress, and fail-fast

- bubbles/spinner with braille dots during polling
- Progress bar: completed/total checks
- Event log with timestamped status updates
- Fail-fast: failure immediately shows error details
```

## Visual Test Results

L4 iterm2-driver: **9/9 PASS** (`test_ghent_watch.py`)

### Screenshots Reviewed

1. **ghent_watch_pass.png** (indrasvat/doot PR #1 — all checks passed) — Status bar shows `ghent indrasvat/doot PR #1`. Status line: green `✓ all checks passed 1/1 elapsed: 12s poll: 10s`. Check list: `✓ make ci (python 3.14)` with `36s passed` in green. Event Log header with `last updated 11s ago`, timestamped entries: `12:51:15 ✓ make ci (python 3.14) 36s`, `12:51:15 ✓ All checks passed`. Help bar: `j/k navigate enter view logs ctrl+c stop watching q quit`.

2. **ghent_watch_fail.png** (indrasvat/peek-it PR #2 — failure detected) — Status bar shows `ghent indrasvat/peek-it PR #2`. Status line: red `✗ failure detected 2/2 elapsed: 12s poll: 10s`. Check list: `✗ build-test (1.23.x)` with `2m 11s completed`, `✗ build-test (1.22.x)` with `2m 7s failure` in red. Event Log: timestamped entries for both failed checks, final entry `✗ Check failure detected fail-fast triggered`. Help bar same as pass state.

### Findings

- Both terminal states (pass/fail) render correctly on first poll since checks are already complete.
- Fail-fast triggers immediately when `OverallStatus == StatusFail`, adding event log entry with "fail-fast triggered" detail.
- Elapsed timer updates correctly (12s = time waiting for API response + render).
- `checkStatusIcon()` from checks.go is reused for check list rendering, ensuring icon consistency.
- Event dedup via `seen` map prevents duplicate entries on subsequent polls.
- Help bar correctly shows watch-specific bindings including `ctrl+c stop watching`.

## Session Protocol

1. Read CLAUDE.md
2. Read this task file
3. **Change this task's status to `IN PROGRESS`**
4. Open `docs/tui-mockups.html` for visual reference
5. Read PRD §6.7
6. Execute steps 1-6
7. Run verification (L1 → L4)
8. **Change this task's status to `DONE`**
9. Update `docs/PROGRESS.md`
10. Update CLAUDE.md Learnings if needed
11. Commit
