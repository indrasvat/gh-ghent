# Task 5.4: Resolve View — Multi-Select

## Status: TODO

## Depends On
- Task 4.4: Wire TUI to Cobra

## Parallelizable With
- Task 5.1: Comments list view (independent)
- Task 5.3: Checks view (independent)

## Problem

The resolve view provides a multi-select interface for resolving review threads interactively. Users toggle checkboxes, confirm, and see a spinner while mutations execute.

## PRD Reference

- §6.4 (Resolve Command) — TUI mode: multi-select, confirmation bar, spinner
- §5.3 (TUI View Architecture) — ViewResolve
- Acceptance criteria: FR-RES-01 (multi-select), FR-RES-02 (confirmation), FR-RES-06 (spinner)

## Research References

- `docs/tui-mockups.html` — Resolve tab
- `docs/testing-strategy.md` §7 — TUI pitfalls

## Files to Create

- `internal/tui/resolve.go` — Resolve view model (list + checkboxes + confirmation)
- `internal/tui/resolve_test.go` — Multi-select and confirmation tests
- `.claude/automations/test_ghent_resolve.py` — iterm2-driver L4 visual test (canonical template from `docs/testing-strategy.md` §5)

## Files to Modify

- `internal/tui/app.go` — Register resolve view

## Execution Steps

### Step 1: Read context
1. Read `docs/tui-mockups.html` — resolve tab
2. Read PRD §6.4 (TUI behavior)

### Step 2: Implement multi-select list
- Uses `bubbles/list` with custom delegate adding checkboxes
- Each item: checkbox [ ] / [✓], file:line, author, body preview
- Space: toggle selection
- a: select all
- j/k: navigate

### Step 3: Implement confirmation bar
- Enter with selections: show confirmation bar at bottom
- "Resolve N threads? [y/N]"
- y: proceed with mutations
- n/Esc: cancel, deselect all

### Step 4: Implement resolution spinner
- During mutations: show `bubbles/spinner` with "Resolving N threads..."
- Update list items as each resolves (change icon to ✓ resolved)
- When complete: show summary "N/M resolved successfully"

### Step 5: Handle permissions
- Disable checkbox for threads where `viewerCanResolve` is false
- Show "(no permission)" label on disabled items

### Step 6: Unit tests
- Space toggles selection
- a selects all eligible threads
- Confirmation bar shows/hides correctly
- Disabled items can't be selected

## Verification

### L1: Unit Tests
```bash
make test
```

### L4: Visual (iterm2-driver)
Create `.claude/automations/test_ghent_resolve.py` following canonical template in `docs/testing-strategy.md` §5:
```bash
uv run .claude/automations/test_ghent_resolve.py
```
Visual assertions:
- Launch: `gh ghent resolve -R indrasvat/tbgs --pr 1` → TUI renders (2 unresolved threads with viewerCanResolve=true)
- **IMPORTANT:** Unresolve threads after testing to restore state
- Verify: checkboxes visible ([ ] or [x] patterns)
- Verify: Space toggles checkbox on current item (before/after screenshot)
- Verify: Enter with selections → confirmation bar visible ("Resolve N threads? [y/N]")
- Verify: disabled items show "(no permission)" label
- Verify: spinner appears during resolution (if live repo with permissions)
- Screenshots: `ghent_resolve_launch.png`, `ghent_resolve_selected.png`, `ghent_resolve_confirm.png`

## Completion Criteria

1. Multi-select with checkboxes
2. Space toggle, a select all
3. Confirmation bar before mutations
4. Spinner during resolution
5. Permission-disabled items handled
6. Layout matches `docs/tui-mockups.html`
7. `make ci` passes
8. PROGRESS.md updated

## Commit

```
feat(tui): add resolve view with multi-select and confirmation

- Multi-select checkboxes with Space toggle and a (select all)
- Confirmation bar before executing mutations
- Spinner during resolution with live status updates
- Permission-disabled threads handled gracefully
```

## Session Protocol

1. Read CLAUDE.md
2. Read this task file
3. **Change this task's status to `IN PROGRESS`**
4. Open `docs/tui-mockups.html` for visual reference
5. Read PRD §6.4
6. Execute steps 1-6
7. Run verification (L1 → L4)
8. **Change this task's status to `DONE`**
9. Update `docs/PROGRESS.md`
10. Update CLAUDE.md Learnings if needed
11. Commit
