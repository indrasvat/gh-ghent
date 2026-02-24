# Task 4.4: Wire TUI to Cobra Commands

## Status: DONE

## Depends On
- Task 4.3: App shell (needs root model and view switching)

## Parallelizable With
- None (sequential dependency)

## Problem

The Bubble Tea TUI needs to be wired to the Cobra command layer so that TTY detection routes to the TUI instead of pipe mode. Each command should initialize the TUI with the appropriate starting view and data.

## PRD Reference

- §5.2 (Dual-Mode Data Flow) — TTY → Bubble Tea, non-TTY → formatter
- §6.1 (Root Command) — TTY detection via `term.FromEnv()`, `--no-tui` override

## Research References

- `docs/go-project-patterns-research.md` §4 — Cobra CLI patterns
- `docs/gh-extensions-support-research.md` §9 — TTY detection with go-gh term package
- `docs/popular-extensions-research.md` §3 — gh-dash Cobra → Bubble Tea wiring

## Files to Create

- `.claude/automations/test_ghent_layout.py` — iterm2-driver L4 visual test for layout integrity (per `docs/testing-strategy.md` §8): box-drawing connectivity, background bleed detection, column alignment

## Files to Modify

- `internal/cli/comments.go` — If TTY: launch TUI with ViewCommentsList; else: pipe mode
- `internal/cli/checks.go` — If TTY: launch TUI with ViewChecksList; else: pipe mode
- `internal/cli/resolve.go` — If TTY: launch TUI with ViewResolve; else: pipe mode
- `internal/cli/summary.go` — If TTY: launch TUI with ViewSummary; else: pipe mode
- `internal/tui/app.go` — Add constructor: `NewApp(startView, data, opts)` with starting data

## Execution Steps

### Step 1: Read context
1. Read PRD §5.2, §6.1
2. Read existing `internal/cli/*.go` commands (TTY detection already in PersistentPreRunE)

### Step 2: Add TUI constructor
- `tui.NewApp(startView View, opts ...Option)` returns `tea.Model`
- Options: `WithThreads([]domain.ReviewThread)`, `WithChecks(domain.ChecksResult)`, etc.
- Each command creates the app with its starting view and pre-fetched data

### Step 3: Wire each command
- In RunE: `if flags.IsTTY && !flags.NoTUI { launchTUI() } else { pipeMode() }`
- TUI launch: `p := tea.NewProgram(tui.NewApp(...), tea.WithAltScreen())`
- Pass pre-fetched data so TUI renders immediately (no loading delay)
- `--no-tui` forces pipe mode even in TTY

### Step 4: Handle TUI exit
- TUI returns final model → extract exit code from model state
- Pass exit code back through Cobra RunE error

### Step 5: Test dual-mode routing
- TTY: TUI launches
- Piped: formatter output
- `--no-tui`: formatter output even in TTY

## Verification

### L1: Unit Tests
```bash
make test
```

### L3: Binary Execution
```bash
make build
# TTY mode (run in terminal)
./bin/gh-ghent comments --pr 1
# Pipe mode
./bin/gh-ghent comments --pr 1 | cat
# Force pipe mode
./bin/gh-ghent comments --pr 1 --no-tui
```

### L4: Visual (iterm2-driver)
Finalize and activate all pending L4 scripts from tasks 4.1-4.3. Run full suite:
```bash
make test-visual  # Runs all .claude/automations/test_ghent_*.py scripts
```
- Verify: `gh ghent comments --pr 1` in TTY → TUI launches with alt screen
- Verify: `gh ghent comments --pr 1 | cat` → pipe mode (no TUI, no ANSI)
- Verify: `gh ghent comments --pr 1 --no-tui` → pipe mode even in TTY
- Verify: Tab switches between views, status bar updates
- Screenshots: `ghent_tui_launch.png`, `ghent_pipe_mode.png`, `ghent_no_tui_flag.png`
- This task is the gate: ALL Phase 4 L4 tests must pass before Phase 5 begins

## Completion Criteria

1. TTY → Bubble Tea TUI for all 4 TUI commands (comments, checks, resolve, summary)
2. Non-TTY → pipe mode formatter
3. `--no-tui` forces pipe mode in TTY
4. Pre-fetched data passed to TUI (no loading delay)
5. Exit codes propagated from TUI model
6. `make ci` passes
7. PROGRESS.md updated

## Commit

```
feat(tui): wire Bubble Tea TUI to Cobra commands with dual-mode routing

- TTY detection routes to TUI, non-TTY to pipe formatters
- --no-tui forces pipe mode in TTY
- Pre-fetched data passed to TUI for instant rendering
- Exit codes propagated from TUI model state
```

## Visual Test Results

### L1: Unit Tests — 302 tests PASS

- All existing tests pass (no regressions from TUI wiring)
- CLI commands: pipe mode path unchanged, TUI path added
- `make ci` passes (lint clean, vet clean)

### L3: Binary Execution — 3/3 PASS

| Test | Status | Details |
|------|--------|---------|
| Pipe mode (--no-tui) | PASS | `gh ghent comments -R indrasvat/tbgs --pr 1 --format json --no-tui` outputs JSON |
| Pipe mode (piped) | PASS | `gh ghent checks -R indrasvat/doot --pr 1 --format json \| jq '.overall_status'` → "pass" |
| Summary pipe | PASS | `gh ghent summary -R indrasvat/tbgs --pr 1 --format json \| jq '.is_merge_ready'` → false |

### L4: iterm2-driver (`test_ghent_layout.py`) — 6/6 PASS

| Test | Status | Details |
|------|--------|---------|
| Build & Install | PASS | `make install` succeeds |
| TUI Launch (comments) | PASS | Alt screen, status bar + help bar, "2 unresolved" |
| Tab to Checks | PASS | Tab switches to checks view with "view logs" help |
| Pipe Mode (--no-tui) | PASS | JSON output, no TUI, PIPE_EXIT=1 |
| Pipe Mode (piped) | PASS | `\| head -3` shows JSON, PIPE2_EXIT=0 |
| Checks TUI Launch | PASS | `gh ghent checks` → TUI with checks help bar |

### Screenshots Reviewed

- `ghent_tui_launch_20260222_234815.png` — `gh ghent comments -R indrasvat/tbgs --pr 1` in TTY. TUI launches in alt screen. Status bar: "ghent" (blue bold), "indrasvat/tbgs" (dim), "PR #1" (purple), "2 unresolved" (red). Help bar: comments-specific keys. Tokyo Night background fills screen. Window title: "gh-ghent".
- `ghent_no_tui_flag_20260222_234825.png` — `--no-tui --format json` outputs raw JSON to terminal (no TUI). Full thread data with diff hunks visible. PIPE_EXIT=1.
- `ghent_pipe_mode_20260222_234834.png` — Piped output (`| head -3`). Shows first 3 lines: `{ "pr_number": 1, "threads": [`. PIPE2_EXIT=0. No TUI launched.

### Findings

- Dual-mode routing works correctly: TTY → TUI, non-TTY → pipe, --no-tui → pipe
- Pre-fetched data passed to TUI — "2 unresolved" appears immediately (no loading delay)
- Tab cycling works in real TUI (comments → checks)
- Exit codes preserved in pipe mode (1 for unresolved threads)
- termenv background properly set/reset around TUI lifecycle
- All 4 commands wired: comments, checks, resolve (TTY without --thread/--all), summary
- Watch mode remains pipe-only (correct per PRD)

## Session Protocol

1. Read CLAUDE.md
2. Read this task file
3. **Change this task's status to `IN PROGRESS`**
4. Read PRD §5.2, §6.1
5. Execute steps 1-5
6. Run verification (L1 → L3 → L4)
7. **Change this task's status to `DONE`**
8. Update `docs/PROGRESS.md`
9. Update CLAUDE.md Learnings if needed
10. Commit
