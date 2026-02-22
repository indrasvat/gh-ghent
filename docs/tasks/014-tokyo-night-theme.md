# Task 4.1: Tokyo Night Theme + Lipgloss Styles

## Status: TODO

## Depends On
- Phase 3 complete (CLI milestone — all pipe mode working)

## Parallelizable With
- None (foundation for all TUI work)

## Problem

ghent's TUI needs a cohesive visual theme based on Tokyo Night colors, implemented as Lipgloss style definitions. This theme package is used by all TUI views and shared components.

## PRD Reference

- §5.4 (Key Design Decisions) — Tokyo Night theme, Lipgloss for styling
- §7.5 (TUI Quality) — No background color bleed, explicit ANSI resets

## Research References

- `docs/testing-strategy.md` §7 — TUI pitfalls (background bleed, lipgloss.Background() avoidance)
- `docs/popular-extensions-research.md` §3 — gh-dash Lipgloss patterns
- `docs/dorikin-patterns-research.md` — Bubble Tea + Lipgloss conventions from our projects
- `docs/tui-mockups.html` — Authoritative visual spec (color bar at top)

## Files to Create

- `internal/tui/styles/theme.go` — Tokyo Night color palette constants
- `internal/tui/styles/styles.go` — Lipgloss style definitions (borders, badges, status indicators)
- `internal/tui/styles/styles_test.go` — Style rendering tests (no panics, correct widths)
- `cmd/theme-demo/main.go` — Temporary test harness for visual verification (renders sample styled elements)
- `.claude/automations/test_ghent_theme.py` — iterm2-driver L4 visual test script (canonical template from `docs/testing-strategy.md` §5)

## Execution Steps

### Step 1: Read context
1. Read `docs/tui-mockups.html` (open in browser — check the color palette in the theme bar)
2. Read `docs/testing-strategy.md` §7 (TUI pitfalls — especially items 1, 2, 6)
3. Read `docs/dorikin-patterns-research.md` (our Lipgloss patterns)

### Step 2: Define Tokyo Night color palette
- Background: dark blue-black (#1a1b26)
- Foreground: light gray (#a9b1d6)
- Accent colors: blue (#7aa2f7), purple (#bb9af7), green (#9ece6a), red (#f7768e), yellow (#e0af68), cyan (#7dcfff)
- Surface colors for cards, borders, dimmed text
- Use `lipgloss.Color()` with hex values

### Step 3: Define Lipgloss styles
- Status bar style (top)
- Help bar style (bottom)
- List item styles (normal, selected, dimmed)
- Badge styles (pass/fail/pending/running)
- Border styles (rounded, with accent colors)
- Diff hunk styles (green additions, red deletions)
- **CRITICAL:** Use `termenv.SetBackgroundColor()` for app background, NOT `lipgloss.Background()`

### Step 4: Add ANSI reset helpers
- Explicit `\033[0m` resets between styled elements
- Helper function: `ResetStyle() string`

### Step 5: Unit tests
- Styles render without panic at various terminal widths
- Color values are correct hex
- No lipgloss.Background() usage (linter rule or test assertion)

## Verification

### L1: Unit Tests
```bash
make test
```

### L4: Visual (iterm2-driver)
Create `.claude/automations/test_ghent_theme.py` following canonical template in `docs/testing-strategy.md` §5:
```bash
uv run .claude/automations/test_ghent_theme.py
```
- Write a small `cmd/theme-demo/main.go` test harness that renders sample styled elements (badges, borders, diff hunks)
- Verify: Tokyo Night colors render correctly, no background bleed, ANSI resets work
- Screenshots: `ghent_theme_badges.png`, `ghent_theme_borders.png`, `ghent_theme_diffhunk.png`
- Assert: no `lipgloss.Background()` usage in codebase (grep assertion in script)

## Completion Criteria

1. Tokyo Night palette defined as constants
2. All Lipgloss styles defined (status bar, help bar, list, badges, borders, diff)
3. NO `lipgloss.Background()` usage — use `termenv.SetBackgroundColor()` instead
4. ANSI reset helpers present
5. `make ci` passes
6. PROGRESS.md updated

## Commit

```
feat(tui): add Tokyo Night theme and Lipgloss style definitions

- Tokyo Night color palette constants
- Lipgloss styles for status bar, help bar, list items, badges, borders, diffs
- termenv.SetBackgroundColor() for app background (no lipgloss.Background())
- ANSI reset helpers between styled elements
```

## Session Protocol

1. Read CLAUDE.md
2. Read this task file
3. **Change this task's status to `IN PROGRESS`**
4. Open `docs/tui-mockups.html` in browser for reference
5. Read `docs/testing-strategy.md` §7
6. Execute steps 1-5
7. Run verification (L1)
8. **Change this task's status to `DONE`**
9. Update `docs/PROGRESS.md`
10. Update CLAUDE.md Learnings if needed
11. Commit
