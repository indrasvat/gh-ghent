# Task 4.2: Shared TUI Components

## Status: DONE

## Depends On
- Task 4.1: Tokyo Night theme (needs styles)

## Parallelizable With
- None (foundation for views)

## Problem

ghent's TUI needs reusable components shared across all views: a status bar (top), help bar (bottom), and diff hunk renderer. These must use the theme styles and handle terminal width correctly.

## PRD Reference

- §5.3 (TUI View Architecture) — components/statusbar.go, helpbar.go, diffhunk.go
- §6.2 (Comments) — status bar shows repo, PR, unresolved/resolved counts
- §7.5 (TUI Quality) — modal padding, ANSI bleed prevention

## Research References

- `docs/tui-mockups.html` — Visual spec for status bar and help bar layouts
- `docs/testing-strategy.md` §7 — TUI pitfalls (items 3, 4, 7 — padding, ANSI, WindowSizeMsg)
- `docs/dorikin-patterns-research.md` — Component patterns from our Bubble Tea projects

## Files to Create

- `internal/tui/components/statusbar.go` — Top bar: repo name, PR number, status counts, badges
- `internal/tui/components/helpbar.go` — Bottom bar: context-sensitive key binding hints
- `internal/tui/components/diffhunk.go` — Diff hunk renderer with syntax coloring
- `.claude/automations/test_ghent_components.py` — iterm2-driver L4 visual test for shared components (extends theme-demo harness)
- `internal/tui/components/statusbar_test.go` — Render tests at various widths
- `internal/tui/components/helpbar_test.go` — Key hint rendering
- `internal/tui/components/diffhunk_test.go` — Diff coloring tests

## Execution Steps

### Step 1: Read context
1. Read `docs/tui-mockups.html` (status bar and help bar sections)
2. Read `docs/testing-strategy.md` §7 (pitfalls 3, 4, 7)

### Step 2: Implement status bar
- Takes: repo owner/name, PR number, key counts (unresolved, checks pass/fail)
- Renders: left-aligned repo+PR, right-aligned counts with badges
- Uses `lipgloss.Place()` for layout
- Adapts to terminal width

### Step 3: Implement help bar
- Takes: list of key bindings (key → action label)
- Renders: `j/k navigate · Enter expand · Tab checks · q quit`
- Context-sensitive: different keys per view
- Fixed at bottom of terminal

### Step 4: Implement diff hunk renderer
- Takes: diff hunk string (from GitHub API `diffHunk` field)
- Colors: green for `+` lines, red for `-` lines, dim for context
- Line numbers displayed
- **CRITICAL:** Use `strings.Repeat(" ", width)` for padding, NOT empty strings
- Explicit `\033[0m` resets between colored lines

### Step 5: Handle WindowSizeMsg
- All components accept width parameter
- Test rendering at width 40, 80, 120, 200

### Step 6: Unit tests
- Each component renders without panic
- Status bar truncates gracefully at narrow widths
- Help bar wraps or truncates at narrow widths
- Diff hunk colors correct lines

## Verification

### L1: Unit Tests
```bash
make test
```

### L4: Visual (iterm2-driver)
Create/extend `.claude/automations/test_ghent_components.py` following canonical template in `docs/testing-strategy.md` §5:
```bash
uv run .claude/automations/test_ghent_components.py
```
- Extend the theme-demo harness to render status bar, help bar, diff hunk at various widths
- Verify: status bar shows repo+PR+counts, help bar shows key hints, diff hunk colors +/- lines
- Verify: components handle width 40, 80, 120 without panic or bleed
- Screenshots: `ghent_statusbar.png`, `ghent_helpbar.png`, `ghent_diffhunk_render.png`

## Completion Criteria

1. Status bar renders with repo, PR, counts, badges
2. Help bar renders context-sensitive key hints
3. Diff hunk renderer colors +/- lines correctly
4. All components handle varying terminal widths
5. No `lipgloss.Width()` on inner elements (pitfall #6)
6. `strings.Repeat` for padding (pitfall #3)
7. `make ci` passes
8. PROGRESS.md updated

## Commit

```
feat(tui): add shared components — status bar, help bar, diff hunk

- Status bar with repo, PR number, count badges
- Context-sensitive help bar with key binding hints
- Diff hunk renderer with green/red line coloring
- Width-adaptive layout, explicit ANSI resets
```

## Visual Test Results

### L1: Unit Tests — 35 tests PASS

- StatusBar: 5 cases (basic, counts, badge, narrow, zero width) + 6 width tests
- HelpBar: 3 width cases + empty inputs + 6 predefined key binding sets validated
- DiffHunk: full render, empty, line types, compact mode, compact empty, 6 width tests
- PadLine: 3 cases (padded, exact, shorter)

### L4: iterm2-driver (`test_ghent_components.py`) — 6/6 PASS

| Test | Status | Details |
|------|--------|---------|
| Build | PASS | theme-demo builds with component imports |
| Status Bar | PASS | repo + PR visible in all three variants |
| Status Bar Variants | PASS | comments (unresolved), checks (HEAD sha), summary (NOT READY) |
| Help Bar | PASS | j/k navigate, enter expand, resolve, key hints per view |
| Diff Hunk | PASS | @@ header, FetchThreads context, +/- lines |
| Width Adaptivity | PASS | Narrow (40 char) bars truncate gracefully |

### Screenshots Reviewed

- `ghent_components_statusbar_20260222_233236.png` — Three status bar variants. Comments view shows "5 unresolved · 2 resolved". Checks view shows "HEAD: a1b2c3d" + "4 passed · 1 failed". Summary view shows PR title + "NOT READY" badge in red. All properly left/right aligned.
- `ghent_components_helpbar_20260222_233236.png` — Help bars for comments, checks, resolve views. Blue key highlights, dim action descriptions. Proper spacing between items.
- `ghent_components_diffhunk_20260222_233236.png` — Full diff hunk with green additions, red deletions, purple @@ header, dim context. Compact mode shows 3 lines + "..." truncation indicator.

### Findings

- All components handle width 40, 80, 120, 200 without panic
- `strings.Repeat(" ", width)` used for all padding (no lipgloss.Width on inner elements)
- ANSI resets (`\033[0m`) appended after every styled line in diff hunk renderer
- Status bar truncates repo name gracefully at narrow widths
- Help bar drops items that don't fit instead of wrapping/overflowing
- No background color bleed visible in screenshots

## Session Protocol

1. Read CLAUDE.md
2. Read this task file
3. **Change this task's status to `IN PROGRESS`**
4. Open `docs/tui-mockups.html` for visual reference
5. Read `docs/testing-strategy.md` §7
6. Execute steps 1-6
7. Run verification (L1 → L4)
8. **Change this task's status to `DONE`**
9. Update `docs/PROGRESS.md`
10. Update CLAUDE.md Learnings if needed
11. Commit
