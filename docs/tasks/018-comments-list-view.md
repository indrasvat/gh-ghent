# Task 5.1: Comments List View

## Status: DONE

## Depends On
- Task 4.4: Wire TUI to Cobra (needs TUI shell wired to commands)

## Parallelizable With
- Task 5.3: Checks view (independent view)
- Task 5.4: Resolve view (independent view)

## Problem

The comments list view is the primary TUI for browsing unresolved review threads. It shows a scrollable list with file:line, author, body preview, and cursor navigation.

## PRD Reference

- §6.2 (Comments Command) — TUI mode behavior, key bindings (j/k, Enter, Tab, r, y, o, f, q)
- §5.3 (TUI View Architecture) — ViewCommentsList
- Acceptance criteria: FR-COM-01 (thread list), FR-COM-02 (j/k nav), FR-COM-04 (grouped by file), FR-COM-10 (Tab)

## Research References

- `docs/tui-mockups.html` — Comments view tab (list layout, selected row styling)
- `docs/popular-extensions-research.md` §3 — gh-dash list patterns with Bubble Tea
- `docs/testing-strategy.md` §7 — TUI pitfalls

## Files to Create

- `internal/tui/comments.go` — Comments list model (bubbles/list)
- `internal/tui/comments_test.go` — List rendering, navigation, key binding tests
- `.claude/automations/test_ghent_comments.py` — iterm2-driver L4 visual test (canonical template from `docs/testing-strategy.md` §5)

## Files to Modify

- `internal/tui/app.go` — Register comments list as ViewCommentsList

## Execution Steps

### Step 1: Read context
1. Read `docs/tui-mockups.html` — comments tab (exact layout)
2. Read PRD §6.2 (TUI key bindings)

### Step 2: Implement comments list model
- Uses `bubbles/list` for the scrollable list
- Each item: file:line (bold), author, body preview (truncated), thread ID, reply count
- Current row highlighted with left-border accent color
- j/k or arrow navigation
- Threads grouped by file path (file header rows)

### Step 3: Wire key bindings
- Enter: switch to ViewCommentsExpand (task 5.2)
- Tab: switch to ViewChecksList
- r: resolve current thread (call resolve mutation)
- y: copy thread ID to clipboard
- o: open thread URL in browser (`browser.OpenURL()`)
- f: toggle file filter (shows only threads for selected file)
- q: quit

### Step 4: Connect to data
- Receive `[]domain.ReviewThread` from app shell
- Sort/group by file path
- Update counts in status bar

### Step 5: Unit tests
- List renders correct number of items
- j/k moves cursor
- Grouping by file path works
- Key bindings produce correct messages

## Verification

### L1: Unit Tests
```bash
make test
```

### L4: Visual (iterm2-driver)
Create `.claude/automations/test_ghent_comments.py` following canonical template in `docs/testing-strategy.md` §5:
```bash
uv run .claude/automations/test_ghent_comments.py
```
Visual assertions (per testing-strategy.md §5 `test_comments` example):
- Launch: `gh ghent comments -R indrasvat/tbgs --pr 1` → TUI renders (2 unresolved threads with file:line)
- Verify: "Review Threads" header visible
- Verify: file:line references present (`.go:` pattern)
- Verify: j/k moves highlighted cursor (capture before/after screenshots)
- Verify: threads grouped by file path (file header rows visible)
- Verify: status bar shows unresolved/total count
- Screenshots: `ghent_comments_launch.png`, `ghent_comments_cursor.png`, `ghent_comments_grouped.png`

## Completion Criteria

1. Thread list renders with file:line, author, body preview
2. j/k navigation with cursor highlighting
3. Threads grouped by file path
4. Tab switches to checks view
5. r/y/o key bindings wired
6. Layout matches `docs/tui-mockups.html`
7. `make ci` passes
8. PROGRESS.md updated

## Commit

```
feat(tui): add comments list view with thread browsing

- bubbles/list with file:line, author, body preview
- j/k cursor navigation with left-border accent
- Threads grouped by file path
- r (resolve), y (copy ID), o (open browser), f (filter), Tab (checks)
```

## Visual Test Results

### L4: test_ghent_comments.py (12/12 PASS)
- Build & Install: PASS
- Launch TUI with Real Data (indrasvat/tbgs PR #1): PASS
- File Grouping Headers: PASS
- Thread Content (file:line, author, body): PASS
- Cursor Marker (▶): PASS
- Status Bar (repo, PR, counts): PASS
- Help Bar (key bindings): PASS
- j Key Navigation (cursor down): PASS
- k Key Navigation (cursor up): PASS
- Thread ID Visible: PASS
- Tab to Checks View: PASS
- Tab Back to Comments: PASS

### L4: test_ghent_openclaw.py (11/11 PASS)
- Tested against openclaw/openclaw PR #24660 (5 threads, 2 files)
- File grouping, markdown stripping, time-ago, navigation all verified

### Screenshots
- `ghent_comments_launch.png` — Comments list with status bar, file headers, threads
- `ghent_comments_cursor.png` — Cursor navigation (j key moved to second thread)
- `ghent_comments_grouped.png` — File grouping with cyan headers and ─ separators

## Session Protocol

1. Read CLAUDE.md
2. Read this task file
3. **Change this task's status to `IN PROGRESS`**
4. Open `docs/tui-mockups.html` for visual reference
5. Read PRD §6.2
6. Execute steps 1-5
7. Run verification (L1 → L4)
8. **Change this task's status to `DONE`**
9. Update `docs/PROGRESS.md`
10. Update CLAUDE.md Learnings if needed
11. Commit
