# Task 4.3: App Shell — Root Model, View Switching, Key Routing

## Status: TODO

## Depends On
- Task 4.2: Shared TUI components (needs status bar, help bar)

## Parallelizable With
- None (sequential dependency)

## Problem

ghent needs a Bubble Tea app shell: a root model that manages view state, routes key events to the active view, handles Tab switching between top-level views, and renders the shared status bar + help bar framing all views.

## PRD Reference

- §5.3 (TUI View Architecture) — root model, view enum, Tab switching, key routing
- §7.5 (TUI Quality) — WindowSizeMsg propagation to ALL sub-models

## Research References

- `docs/tui-mockups.html` — Tab bar, view switching behavior
- `docs/testing-strategy.md` §7 — Pitfall #5 (switch shadowing), #7 (WindowSizeMsg propagation)
- `docs/popular-extensions-research.md` §3 — gh-dash Bubble Tea architecture
- `docs/dorikin-patterns-research.md` — Elm architecture from our projects

## Files to Create

- `internal/tui/app.go` — Root model: Init, Update, View, view enum
- `internal/tui/app_test.go` — View switching, key routing tests
- `internal/tui/keymap.go` — Key bindings (bubbles/key)
- `.claude/automations/test_ghent_shell.py` — iterm2-driver L4 visual test for app shell (pending until 4.4 wired)

## Execution Steps

### Step 1: Read context
1. Read `docs/tui-mockups.html` (tab switching, key bindings per view)
2. Read `docs/testing-strategy.md` §7 (pitfalls 5, 7)
3. Read `docs/dorikin-patterns-research.md` (Elm architecture patterns)

### Step 2: Define view enum and root model
- View enum: ViewCommentsList, ViewCommentsExpand, ViewChecksList, ViewChecksLog, ViewResolve, ViewSummary, ViewWatch
- Root model holds: activeView, width/height, all sub-models, shared data

### Step 3: Implement Update with key routing
- Tab: cycle between comments ↔ checks (top-level views)
- Enter: drill into detail views
- Esc: back to list view
- c/k/r: jump to comments/checks/resolve from summary
- q: quit
- **CRITICAL:** Use `typedMsg := msg.(type)` pattern, avoid switch shadowing (pitfall #5)

### Step 4: Implement View composition
- Render: status bar (top) + active view (middle) + help bar (bottom)
- Help bar content changes per active view

### Step 5: WindowSizeMsg propagation
- **CRITICAL:** On `tea.WindowSizeMsg`, propagate to ALL sub-models (active AND inactive)
- Store width/height in root model
- This prevents garbled layout on view switch (pitfall #7)

### Step 6: Background color setup
- Set `termenv.SetBackgroundColor()` before Bubble Tea starts
- Reset `output.Reset()` after Bubble Tea exits

### Step 7: Unit tests
- Tab switches views
- Esc returns to list from detail
- WindowSizeMsg propagates to all sub-models
- q sends tea.Quit

## Verification

### L1: Unit Tests
```bash
make test
```

### L4: Visual (iterm2-driver)
Create `.claude/automations/test_ghent_shell.py` following canonical template in `docs/testing-strategy.md` §5:
```bash
uv run .claude/automations/test_ghent_shell.py
```
- Note: Full L4 testing requires Task 4.4 (Cobra wiring). Create the script here but mark tests as pending until 4.4 is complete.
- Verify (once wired): App launches in alt screen, Tab cycles views, status bar renders, help bar updates per view
- Verify: WindowSizeMsg propagation (resize terminal during test → layout adjusts)
- Verify: termenv background color applied (no terminal default background showing)
- Screenshots: `ghent_shell_launch.png`, `ghent_shell_tab_comments.png`, `ghent_shell_tab_checks.png`

## Completion Criteria

1. Root model with view enum and all sub-model slots
2. Tab cycles comments ↔ checks
3. Enter/Esc for drill-in/back
4. WindowSizeMsg propagated to ALL sub-models
5. termenv background set/reset around Bubble Tea
6. No switch shadowing in Update
7. `make ci` passes
8. PROGRESS.md updated

## Commit

```
feat(tui): add app shell with view switching and key routing

- Root model with view enum, Tab cycling, Enter/Esc navigation
- WindowSizeMsg propagated to ALL sub-models (active + inactive)
- termenv background set/reset around Bubble Tea lifecycle
- Status bar + help bar framing with context-sensitive hints
```

## Session Protocol

1. Read CLAUDE.md
2. Read this task file
3. **Change this task's status to `IN PROGRESS`**
4. Read `docs/testing-strategy.md` §7 (pitfalls 5, 7)
5. Read `docs/dorikin-patterns-research.md`
6. Execute steps 1-7
7. Run verification (L1)
8. **Change this task's status to `DONE`**
9. Update `docs/PROGRESS.md`
10. Update CLAUDE.md Learnings if needed
11. Commit
