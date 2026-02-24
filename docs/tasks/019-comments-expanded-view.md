# Task 5.2: Comments Expanded View

## Status: DONE

## Depends On
- Task 5.1: Comments list view (needs list to drill into)

## Parallelizable With
- None (depends on 5.1)

## Problem

When a user presses Enter on a thread in the comments list, the expanded view shows the full diff hunk, all comments with nested replies, author coloring, and timestamps. This is the detail view for understanding a review thread.

## PRD Reference

- §6.2 (Comments Command) — expanded view: diff hunk, all comments, n/p navigation
- §5.3 (TUI View Architecture) — ViewCommentsExpand (bubbles/viewport)
- Acceptance criteria: FR-COM-03 (Enter expands showing ALL comments + diff hunk)

## Research References

- `docs/tui-mockups.html` — Comments (expanded) tab
- `docs/testing-strategy.md` §7 — TUI pitfalls (diff hunk rendering)

## Files to Modify

- `internal/tui/comments.go` — Add expanded view state and rendering
- `internal/tui/comments_test.go` — Expanded view tests

## Execution Steps

### Step 1: Read context
1. Read `docs/tui-mockups.html` — comments expanded tab
2. Read PRD §6.2 (expanded view behavior)

### Step 2: Implement expanded view
- Uses `bubbles/viewport` for scrollable content
- Renders: diff hunk (using components/diffhunk.go), all comments in chronological order
- Each comment: author (colored by ownership: PR author, reviewer, commenter), body, timestamp
- Nested reply threading with indentation

### Step 3: Wire key bindings
- Esc: back to comments list
- n/p: next/prev thread (without going back to list)
- r: resolve this thread
- o: open thread in browser
- j/k or scroll: viewport scrolling within expanded content

### Step 4: Handle long content
- Diff hunks can be large — viewport handles scrolling
- Comment bodies with code blocks rendered with monospace
- Truncate extremely long comments with "... (truncated)" link

### Step 5: Unit tests
- Expanded view renders diff hunk + all comments
- n/p cycles through threads
- Esc returns correct message to app shell

## Verification

### L1: Unit Tests
```bash
make test
```

### L4: Visual (iterm2-driver)
Extend `.claude/automations/test_ghent_comments.py` with expanded view tests:
```bash
uv run .claude/automations/test_ghent_comments.py
```
Visual assertions (use `gh ghent comments -R indrasvat/tbgs --pr 1` — 2 threads with diff hunks):
- From comments list, press Enter → expanded view renders
- Verify: diff hunk visible with colored +/- lines (green/red)
- Verify: all comments shown with author names and timestamps
- Verify: n/p cycles to next/prev thread (screen content changes)
- Verify: Esc returns to list view (list re-renders)
- Verify: viewport scrolling works for long content (j/k within expanded)
- Screenshots: `ghent_comments_expanded.png`, `ghent_comments_diffhunk.png`, `ghent_comments_next_thread.png`

## Completion Criteria

1. Expanded view shows diff hunk + all comments
2. Author coloring by ownership role
3. n/p cycles threads, Esc returns to list
4. Viewport scrolling for long content
5. Layout matches `docs/tui-mockups.html`
6. `make ci` passes
7. PROGRESS.md updated

## Commit

```
feat(tui): add comments expanded view with diff hunk and thread display

- bubbles/viewport for scrollable thread content
- Diff hunk with syntax coloring via diffhunk component
- All comments with author coloring and timestamps
- n/p thread cycling, Esc back to list
```

## Visual Test Results

### L4: test_ghent_expanded.py (14/14 — 11 PASS, 3 UNVERIFIED, 0 FAIL)
- Build & Install: PASS
- Launch TUI: PASS
- Enter Expand: PASS
- Thread Header (file:line): PASS
- Diff Hunk (@@ header, +/- lines): PASS
- Comments (@author, body): PASS
- Replies with │ Border: UNVERIFIED (tbgs threads have 1 comment each)
- Time Ago: UNVERIFIED (timestamps may not capture in alt-screen)
- Help Bar (esc, scroll, n/p, resolve): PASS
- Status Bar (Thread X of Y): PASS
- Next Thread (n): UNVERIFIED (tbgs threads have similar content)
- Prev Thread (p): PASS
- Viewport Scroll (j/k): PASS
- Esc Back to List: PASS

### L4: test_ghent_expanded_openclaw.py (11/11 PASS)
- Tested against openclaw/openclaw PR #24660 (5 threads, 2 files)
- Build: PASS
- Launch with threads: PASS
- Enter to expand: PASS
- Diff hunk: PASS
- Author visible: PASS
- Next thread (n) — content changed: PASS
- Navigate to thread 5: PASS
- Prev thread (p) — content changed: PASS
- Status bar (Thread X of Y, unresolved): PASS
- Esc back to list: PASS
- Expand 2nd thread: PASS

### Screenshots
- `ghent_comments_expanded.png` — Expanded thread with diff hunk, file:line, thread ID
- `ghent_comments_diffhunk.png` — Diff context with green/red syntax coloring
- `ghent_comments_next_thread.png` — Different thread after n key press

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
