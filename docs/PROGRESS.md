# ghent — Implementation Progress

> **STRICT RULE:** This file MUST be updated at the end of every coding session.
> If a task is BLOCKED, document the issue with full details below.

## Quick Context

| Field | Value |
|-------|-------|
| **Current Phase** | Phase 9: Bug Fixes |
| **Current Task** | Task 033 DONE. Fix 10 dead TUI keybindings. |
| **Blocker** | None |
| **Last Action** | All 10 keybindings implemented, 60 new tests, L4 11/11 PASS. |
| **Last Updated** | 2026-02-24 |

## How to Resume

1. Read this file — find current task above
2. Read the task file at the path above
3. Read CLAUDE.md (auto-loaded)
4. Read PRD sections referenced in the task file
5. Change task status to `IN PROGRESS`
6. Execute, verify, update this file, mark task `DONE`, commit

## Phase Progress

### Phase 1: Walking Skeleton
- [x] Task 1.1: Repository scaffold → `docs/tasks/000-repository-scaffold.md`
- [x] Task 1.2: Cobra CLI skeleton → `docs/tasks/001-cobra-cli-skeleton.md`
- [x] Task 1.3: Domain types and port interfaces → `docs/tasks/002-domain-types.md`
- [x] Task 1.4: GitHub API client wiring → `docs/tasks/003-github-api-client.md`

### Phase 2: CLI Commands (pipe mode, end-to-end)
- [x] Task 2.1: `gh ghent comments` → `docs/tasks/004-comments-command.md`
- [x] Task 2.2: `gh ghent checks` → `docs/tasks/005-checks-command.md`
- [x] Task 2.3: `gh ghent checks --logs` → `docs/tasks/006-checks-logs.md`
- [x] Task 2.4: `gh ghent resolve` → `docs/tasks/007-resolve-command.md`
- [x] Task 2.5: `gh ghent reply` → `docs/tasks/008-reply-command.md`
- [x] Task 2.6: `gh ghent summary` → `docs/tasks/009-summary-command.md`

### Phase 3: CLI Polish
- [x] Task 3.1: Watch mode (pipe) → `docs/tasks/010-watch-mode-pipe.md`
- [x] Task 3.2: Error handling hardening → `docs/tasks/011-error-handling.md`
- [x] Task 3.3: Extension packaging → `docs/tasks/012-extension-packaging.md`
- [x] Task 3.4: README + --help → `docs/tasks/013-readme-help.md`
- [x] Task 3.5: Debug logging & tracing → `docs/tasks/028-debug-logging.md`

> **Milestone: CLI complete** — all commands work in pipe mode, fully tested

### Phase 4: TUI Foundation
- [x] Task 4.1: Tokyo Night theme + Lipgloss styles → `docs/tasks/014-tokyo-night-theme.md`
- [x] Task 4.2: Shared TUI components → `docs/tasks/015-shared-tui-components.md`
- [x] Task 4.3: App shell — root model, view switching → `docs/tasks/016-app-shell.md`
- [x] Task 4.4: Wire TUI to Cobra commands → `docs/tasks/017-wire-tui-cobra.md`

> **Milestone: TUI Foundation complete** — all views have placeholder content, dual-mode routing works

### Phase 5: TUI Views
- [x] Task 5.1: Comments list view → `docs/tasks/018-comments-list-view.md`
- [x] Task 5.2: Comments expanded view → `docs/tasks/019-comments-expanded-view.md`
- [x] Task 5.3: Checks view + log viewer → `docs/tasks/020-checks-view.md`
- [x] Task 5.4: Resolve view — multi-select → `docs/tasks/021-resolve-view.md`
- [x] Task 5.5: Summary dashboard → `docs/tasks/022-summary-dashboard.md`
- [x] Task 5.6: Watch mode TUI → `docs/tasks/023-watch-mode-tui.md`

> **Milestone: TUI Views complete** — all interactive views implemented, verified with live CI

### Phase 6: Agent Optimization
- [x] Task 6.1: --since flag → `docs/tasks/024-since-filter.md`
- [x] Task 6.2: --group-by flag → `docs/tasks/025-group-by-flag.md`
- [x] Task 6.3: Summary enhancements → `docs/tasks/026-summary-enhancements.md`
- [x] Task 6.4: Batch resolve → `docs/tasks/027-batch-resolve.md`

> **Milestone: Agent Optimization complete** — all agent-facing features implemented, 489 tests passing

### Phase 7: Distribution & Ecosystem
- [x] Task 7.1: Agent Skill → `docs/tasks/030-agent-skill.md`

> **Milestone: Distribution complete** — Agent Skill installable via `npx skills add indrasvat/gh-ghent`

### Phase 8: Polish & DX
- [x] Task 8.1: Styled help & version output → `docs/tasks/031-styled-help-version.md`

### Phase 9: Bug Fixes
- [x] Task 9.1: Summary overflow, async startup, Esc navigation → `docs/tasks/032-summary-overflow-esc-nav.md`
- [x] Task 9.2: Dead keybindings → `docs/tasks/033-dead-keybindings.md`

## Blockers

(None currently)

## Session Log

### 2026-02-24 (Phase 9: Bug Fixes — Task 033 Dead Keybindings)
- **Task 033 (Dead keybindings):** Fixed all 10 advertised-but-unimplemented TUI keybindings across 4 views.
  - **Comments list:** `f` (filter by file cycling), `y` (copy thread ID to clipboard), `o` (open comment in browser), `r` (switch to resolve view)
  - **Comments expanded:** `r` (resolve current thread via API), `y` (copy thread ID), `o` (open in browser)
  - **Checks list:** `R` (re-run failed checks via `gh run rerun`)
  - **Summary:** `o` (open PR in browser), `R` (re-run failed checks)
- **New files:** `internal/tui/clipboard.go` — clipboard helper (`pbcopy`/`xclip`/`xsel`), following `openInBrowser()` pattern
- **Modified files:** `comments.go` (y/o/f handlers + filter state + `cycleFilter()` + `computeUniquePaths()`), `app.go` (r/o/R app-level handlers + `clipboardCopyMsg`/`rerunResultMsg` msg cases + filter indicator in status bar), `checks.go` (`extractRunID()` + `rerunFailedChecks()` + `rerunResultMsg`), `keymap.go` (OpenPR + Rerun bindings)
- **Test count:** 489 → 549 (60 new tests). `make ci-fast` clean with race detector.
- **L4:** `.claude/automations/test_ghent_keybindings.py` rewritten with exhaustive content-specific assertions: 11/11 PASS (10 keybinding tests + 1 regression). Clipboard verified via `pbpaste` (PRRT_ prefix), filter cycling verified with badge + item count + clear, resolve view switch verified with multi-indicator (checkboxes, help bar), roundtrip Esc confirmed.

### 2026-02-24 (Phase 9: Bug Fixes — Task 032 Summary Overflow, Async Startup, Esc Nav)
- **P3 (Esc navigation):** Generalized Esc handler in `handleKey()` — returns to `prevView` from any list view, not just resolve/summary. Initialized `prevView = initialView` in `NewApp()`. 3 new tests.
- **P1 (Summary overflow):** Capped approvals at `maxReviewsShow = 5` with priority sort (CHANGES_REQUESTED > APPROVED > COMMENTED) and "... and N more" overflow indicator. Added viewport scrolling with `scrollOffset` and j/↓/↑ keys. 7 new tests.
- **P2 (Async loading):** Added `FetchCommentsFunc`/`FetchChecksFunc`/`FetchReviewsFunc` types, `commentsLoadedMsg`/`checksLoadedMsg`/`reviewsLoadedMsg` messages, `SetAsyncFetch()` method. `Init()` fires parallel `tea.Cmd`s, `Update()` handles progressive rendering. Loading view shows "Loading PR data..." until first data arrives. Modified `cli/summary.go` TTY path to launch TUI immediately with async fetch closures. Pipe mode retains blocking errgroup. 6 new tests.
- **Stress-tested** against oven-sh/bun extreme PRs: #24063 (61 reviews, 101 threads), #27327 (68 threads, 25 reviews, 59 checks), #27338 (46 threads), #27264 (6/6 unresolved), #27056 (42 reviews). All render correctly with overflow indicators.
- **Performance:** TUI first-frame dropped from 1.3–6.5s → <0.5s (instant). Pipe-mode latency unchanged (API-bound).
- **L4:** 9/9 existing summary tests PASS + 5/5 task 032 stress tests (4 PASS, 1 UNVERIFIED).
- **Test count:** 501 → 516 (15 new tests). `make lint` clean. `make test` all pass with race detector.


- **Task 8.1 (Styled Help & Version):** Custom Cobra templates with Tokyo Night lipgloss styling for `--version`, `--help` (root + all 5 subcommands). TTY-aware: full color in terminal, clean plain text when piped. Created `internal/cli/help.go` with template functions, added `ShortCommit()`/`ShortDate()` to version package.
- Added Unicode block-character ASCII banner with Tokyo Night blue→purple→cyan gradient for `--version` TTY output (skipped when piped).
- **Test count:** 489 → 501 (12 new version helper tests)
- **L4:** 8/8 iterm2-driver tests pass. Screenshots in `.claude/screenshots/ghent_help_*.png`.
- Also fixed false positive in `pre-task-done-gate.sh` hook (matched prose "DONE" in session protocols).

### 2026-02-24 (Phase 7: Distribution — Task 7.1 Agent Skill)
- **Task 7.1 (Agent Skill):** Created `skill/` directory with 6 files (1,088 total lines):
  - `SKILL.md` (161 lines) — main skill file with YAML frontmatter, triggering conditions, quick start, agent mode, core workflow, exit codes summary, --since/--compact/--group-by coverage, links to supporting files
  - `references/command-reference.md` (347 lines) — all 5 commands with flags, types, defaults, JSON output schemas, key fields for agents
  - `references/agent-workflows.md` (192 lines) — 5 opinionated workflows: fix comments, monitor CI, full review cycle, incremental delta, selective resolve
  - `references/exit-codes.md` (87 lines) — per-command exit code table, bash conditional patterns
  - `examples/review-cycle.md` (152 lines) — annotated walkthrough against indrasvat/tbgs PR #1 with real JSON output
  - `examples/ci-monitor.md` (149 lines) — CI monitoring walkthrough against indrasvat/peek-it PR #2 and indrasvat/visarga PR #1
- **Verification:** All commands, flags, exit codes, and JSON schemas verified by running `gh ghent` against 7 real repos (tbgs, peek-it, doot, visarga, querylastic, context-lens, openclaw/openclaw). Tested --group-by, --since, --compact, --logs, --dry-run, --file, --author, all 3 output formats, error cases (nonexistent repo/PR). Every field name and exit code in the skill matches actual implementation.
- Updated README.md with "For AI Agents" section and skill installation instructions.
- **Phase 7 complete.** Installable via `npx skills add indrasvat/gh-ghent`.

### 2026-02-23 (Phase 6: Agent Optimization — all 4 tasks parallel)
- **Tasks 3.2, 3.4 (error handling, README):** Already code-complete from prior sessions but task files still showed TODO. Created L4 visual tests (`test_ghent_errors.py` 6/6 PASS, `test_ghent_help.py` 8/8 PASS, `test_ghent_agent.py` 8/8 PASS) to satisfy pre-task-done-gate hook, added Visual Test Results sections, marked DONE.
- **Task 6.1 (--since filter):** Added `internal/cli/since.go` — `ParseSince()` for ISO 8601 + relative durations (30d, 2w, 24h), `FilterThreadsBySince()`, `FilterChecksBySince()`. `--since` persistent flag on root. 15 tests.
- **Task 6.2 (--group-by flag):** Added `--group-by` flag to comments (file/author/status). `groupThreads()` + `threadKeyFunc()` in comments.go. `FormatGroupedComments` on all 3 formatters — JSON groups array, XML groups/group, Markdown `##` headers. 8 grouping tests + 6 formatter tests.
- **Task 6.3 (--compact summary):** Added `--compact` flag to summary. `FormatCompactSummary` with `pr_age`, `last_update`, `review_cycles`, `is_merge_ready`. `computePRAge`, `computeLastUpdate`, `computeReviewCycles`, `formatRelativeTime` helpers. All 3 formatters support compact mode.
- **Task 6.4 (batch resolve):** Added `--file`, `--author`, `--dry-run` flags to resolve. `matchesFilters()` with `path.Match` glob, `resolveBatch()` with per-thread results. `SkippedCount`/`DryRun` on domain.ResolveResults. 15 tests.
- All 4 tasks ran in parallel via agent teams in git worktrees, merged sequentially with `make ci-fast` verification after each.
- **L4 Phase 6 visual test:** `test_ghent_phase6.py` — 8/8 PASS against indrasvat/tbgs PR #1 (--since, --group-by file/author, --compact, batch resolve --dry-run/--file, relative 30d)
- **TUI visual regression:** All 4 core TUI tests re-run — comments 12/12, checks 15/15, resolve 12/12, summary 9/9 PASS.
- Test count: 419 → 489 (70 new tests)
- **Phase 6 complete. All 30 tasks DONE. Project feature-complete.**

### 2026-02-23 (Task 5.6 — TUI Views: Watch Mode)
- **Task 5.6 (Watch mode TUI):** Created `internal/tui/watcher.go` — `watcherModel` with `bubbles/spinner` (dot animation), `tea.Tick` polling (10s interval), three states (polling/done/failed). Check list with `checkStatusIcon()` reuse, color-coded status. Event log with timestamps, auto-scroll, dedup via `seen` map. `formatDuration()` helper for elapsed/check times.
- Wired to `app.go`: `watcher` sub-model, `ViewWatch` rendering, WindowSizeMsg propagation, `SetWatchFetch()` method, `Init()` returns watcher commands for ViewWatch start.
- Updated `cli/checks.go`: `--watch` + TTY routes to watch TUI via `launchTUI(ViewWatch, withWatchFetch(...))`. Non-TTY still uses pipe-mode `WatchChecks`.
- 14 unit tests in watcher_test.go (empty view, initial, poll result, all pass, fail-fast, poll error, seen dedup, tick ignored when done, nil fetchFn, event log scroll, formatDuration, makeEvent, app integration, status bar).
- **Bug found via live CI testing**: Duplicate `running...  running...` on in-progress check lines. Both `dur` and `status` were set to "running..." for in_progress state. Fixed by only setting `status`, leaving `dur` empty.
- L4: 10/10 PASS (test_ghent_watch.py) + live CI test against indrasvat/gh-ghent PR #1 with in-progress checks.
- Verification: 419 tests pass, lint clean, vet clean (`make ci` ✓)
- **Phase 5 complete** — all TUI views implemented and verified.

### 2026-02-23 (Task 5.5 — TUI Views: Summary Dashboard)
- **Task 5.5 (Summary dashboard):** Created `internal/tui/summary.go` — `summaryModel` with KPI cards row (4 cards: Unresolved, Passed, Failed, Approvals using `lipgloss.JoinHorizontal` and rounded borders), three section previews (Review Threads with top-3 truncation, CI Checks with failed check annotations and pass count, Approvals with reviewer icons and states), merge readiness badge (READY/NOT READY in status bar). Color-coded dots per section (green=clear, red=issues, yellow=pending). Reuses package-scoped helpers: `padWithRight`, `formatTimeAgo`, `checkIsFailed`, `dimStyle`/`greenStyle`/`redStyle`.
- Wired to `app.go`: summary sub-model, WindowSizeMsg propagation, `SetComments`/`SetChecks`/`SetReviews` set data, status bar shows merge readiness badge, `ViewSummary` renders summary view. Quick-nav c/k/r already handled in app.go.
- 16 unit tests in summary_test.go (empty view, KPI cards, merge readiness 6 cases, badge, threads section, truncation, checks section, approvals section, review icons, card colors, check names, app integration, ready integration, quick-nav, zero width).
- L4: 9/9 PASS (test_ghent_summary.py against tbgs NOT READY, doot READY-ish, peek-it NOT READY)
- Verification: 405 tests pass, lint clean, vet clean (`make ci-fast` ✓)
- Next: Task 5.6 Watch Mode TUI

### 2026-02-23 (Task 5.4 — TUI Views: Resolve)
- **Task 5.4 (Resolve view — multi-select):** Created `internal/tui/resolve.go` — `resolveModel` with four states (browsing, confirming, resolving, done). Multi-select checkboxes: `[ ]` unselected, `[✓]` selected, `[-]` no permission, `[✗]` failed. Key bindings: j/k navigate, Space toggle, a select all/deselect, Enter confirm, Esc cancel, y/n confirm shortcuts, o open in browser. Confirmation bar: "Resolve N threads?" with enter/esc hints. Resolving status: "⟳ Resolving... N/M". Done status: "✓ N resolved" / "✗ N failed". `resolveRequestMsg` emitted to App for API calls via `resolveFunc` callback. `resolveThreadMsg` per-thread results, `resolveAllDoneMsg` when complete. Permission filtering: threads without `viewerCanResolve` get `[-]` and "(no permission)" label. Thread rendering: cursor highlight, file:line, author, body preview, truncated thread ID right-aligned.
- Wired to `app.go`: resolve sub-model, `resolveFunc` callback, `SetResolver`, `resolveRequestMsg` handler with `tea.Batch`, status bar (resolve mode + selection count + unresolved count), help bar `ResolveKeys()`. Fixed Esc routing: App's global Esc handler now checks `resolve.state == resolveStateConfirming` and forwards to resolve model instead of switching views.
- Updated `cli/tui.go` with `withResolver` option, `cli/resolve.go` creates resolver callback.
- 16 unit tests in resolve_test.go + 1 new app_test.go test for Esc routing.
- L4: 12/12 PASS (test_ghent_resolve.py against indrasvat/tbgs PR #1)
- Verification: 384 tests pass, lint clean, vet clean (`make ci-fast` ✓)
- Next: Task 5.5 Summary Dashboard

### 2026-02-23 (Task 5.3 — TUI Views: Checks)
- **Task 5.3 (Checks view + log viewer):** Created `internal/tui/checks.go` — `checksListModel` with custom scrollable list, status icons (✓/✗/⟳/◌), auto-expanded annotations for failed checks with error count header and file:line detail. `checksLogModel` line-based viewport with check header, annotations, log excerpt display. Helper functions: `checkIsFailed`, `checkStatusIcon`, `renderCheckStatusText`, `formatCheckDuration`, `openInBrowser`. Wired to `app.go` with `selectCheckMsg` pattern and WindowSizeMsg propagation. Pre-fetch logs for failed checks in `cli/checks.go` before TUI launch. Added `ChecksLogKeys()` to helpbar. 16 unit tests. L4: 15/15 PASS (peek-it fail, doot pass, context-lens mixed).
- Verification: 367 tests pass, lint clean, vet clean (`make ci` ✓)
- Next: Task 5.4 Resolve View — Multi-Select

### 2026-02-23 (Tasks 5.1, 5.2 — TUI Views: Comments)
- **Task 5.1 (Comments list view):** Created `internal/tui/comments.go` — custom scrollable list with file-path grouping, cursor navigation (j/k) skipping file headers, multi-line item viewport (threads=3 lines, headers=1 line). `formatTimeAgo()` for relative timestamps, `stripMarkdown()` regex-based cleanup of images/links/HTML/bold/backticks. 18 unit tests. L4: 12/12 PASS (tbgs), 11/11 PASS (openclaw).
- **Task 5.2 (Comments expanded view):** Added `commentsExpandedModel` to comments.go — line-based viewport for scrollable thread content. Renders: thread header (file:line + ID), diff hunk via `components.RenderDiffHunk`, all comments with author coloring and time-ago, reply indentation with `│` left borders. n/p thread cycling, j/k viewport scrolling, Esc back to list. Status bar shows "Thread X of Y". 15 unit tests. L4: 11/14 PASS (tbgs, 3 unverified), 11/11 PASS (openclaw).
- Wired both sub-models to `app.go` with WindowSizeMsg propagation to ALL sub-models (active + inactive)
- Verification: 352 tests pass, lint clean, vet clean (`make ci` ✓)
- Next: Task 5.3 Checks View + Log Viewer

### 2026-02-22 (Tasks 4.1, 4.2, 4.3 — TUI Foundation)
- **Task 4.1 (Tokyo Night theme):** Created `internal/tui/styles/theme.go` (17 color constants), `styles.go` (all Lipgloss style definitions), `styles_test.go`. `cmd/theme-demo/main.go` visual harness. L4 test: 8/8 PASS.
- **Task 4.2 (Shared components):** Created `internal/tui/components/` — statusbar.go, helpbar.go, diffhunk.go with tests. 6 predefined key binding sets per view. Extended theme-demo. L4 test: 6/6 PASS.
- **Task 4.3 (App shell):** Created `internal/tui/app.go` — root Bubble Tea model with View enum (7 views), key routing (Tab cycle, Enter drill-in, Esc back, summary shortcuts c/k/r), WindowSizeMsg propagation to all sub-models. `keymap.go` — bubbles/key bindings. `cmd/shell-demo/main.go` — interactive demo. 23 unit tests, L4 test: 6/6 PASS. No switch shadowing (pitfall #5), termenv background set/reset.
- **Task 4.4 (Wire TUI to Cobra):** Created `internal/cli/tui.go` — `launchTUI()` helper with functional options (withRepo, withPR, withComments, withChecks, withReviews). Modified `comments.go`, `checks.go`, `summary.go`, `resolve.go` — if `Flags.IsTTY` → launch TUI with pre-fetched data, else pipe mode. Resolve: TTY without --thread/--all → interactive TUI, else pipe mode. Watch mode stays pipe-only. L3: all pipe mode tests pass. L4 test: 6/6 PASS (TUI launch, Tab switching, --no-tui, piped output, checks TUI).
- Added charmbracelet/bubbletea v1.3.10, bubbles v1.0.0 dependencies
- **Phase 4 complete** — TUI foundation wired to Cobra, dual-mode routing works
- Verification: 302 tests pass, lint clean, vet clean (`make ci` ✓)
- Next: Phase 5 (TUI Views) — Task 5.1 Comments List View

### 2026-02-23 (Tasks 2.3, 2.6 — parallel execution)
- Ran two agents in parallel via worktree isolation
- **Task 2.3 (checks --logs):** `internal/github/logs.go` — REST job log fetcher via `RequestWithContext` (raw text, not JSON). `ExtractErrorLines` with ANSI stripping, timestamp removal, error keyword/prefix matching, file:line pattern detection, context lines, gap markers, 50-line truncation. Wired `--logs` flag in cli/checks.go with graceful degradation. LogExcerpt field added to domain.CheckRun, XML/MD formatters updated.
- **Task 2.6 (summary):** `internal/github/reviews.go` — GraphQL PR reviews query. `internal/cli/summary.go` — errgroup parallel fetch (threads + checks + reviews), graceful degradation (reviews optional). `IsMergeReady` logic: no unresolved threads AND checks pass AND at least 1 approval with no CHANGES_REQUESTED. Exit code 0=ready, 1=not ready. XML proper types (xmlSummary, xmlSummaryComments, xmlSummaryChecks, xmlReview). Enhanced markdown FormatSummary with [READY]/[NOT READY] badge and reviewer table.
- Merged both worktrees: resolved formatter overlaps (LogExcerpt in checks + summary XML types), removed FetchReviews stub from client.go, added golang.org/x/sync dependency
- Added `.claude/worktrees/` to .gitignore
- **Phase 2 complete** — all 6 CLI commands working in pipe mode
- Verification: 171 tests pass, 0 lint issues, L3 smoke tests pass

### 2026-02-22 (Tasks 2.2, 2.4, 2.5 — parallel execution)
- Ran three agents in parallel via worktree isolation
- **Task 2.2 (checks):** `internal/github/checks.go` — REST check runs + annotations fetch, HEAD SHA resolution, status aggregation (fail > pending > pass). Formatters extended with FormatChecks. Exit codes: 0/1/3. Tests with fixtures.
- **Task 2.4 (resolve):** `internal/github/resolve.go` — GraphQL `resolveReviewThread`/`unresolveReviewThread` mutations. `cli/resolve.go` wired with --thread/--all/--unresolve flags, viewerCanResolve permission check. New domain types: ResolveResult, ResolveError, ResolveResults.
- **Task 2.5 (reply):** `internal/github/reply.go` — REST reply via POST .../comments/{id}/replies. Thread validation via GraphQL, viewerCanReply check, databaseId targeting. cli/reply.go with --body/--body-file (stdin via -) mutual exclusion.
- Fixed gofmt lint issue in reply_test.go (struct alignment)
- Verification: `go test -race -shuffle=on ./...` ✓, `go build` ✓, `make lint` ✓

### 2026-02-22 (Task 2.1: Comments command)
- Created `internal/github/threads.go` — GraphQL review thread fetcher with pagination (pageInfo/endCursor loop)
- Client-side isResolved filtering: only unresolved threads returned in result
- Response struct mapping with json tags for go-gh `DoWithContext` raw string queries
- Created `internal/formatter/` package — JSON, XML, Markdown formatters implementing `domain.Formatter`
- JSON: `json.NewEncoder` with indent, XML: custom wrapper types with `xml.Header`, MD: structured headers/tables
- Created `internal/cli/repo.go` — `resolveRepo` helper (flag OWNER/REPO or `repository.Current()`)
- Wired `internal/cli/comments.go` RunE: fetch → format → stdout, exit(1) on unresolved threads
- Removed FetchThreads stub from client.go, updated client_test.go to remove stale test case
- Test fixtures: `testdata/graphql/review_threads.json` (3 threads, mix), `review_threads_page2.json` (pagination)
- Tests: 5 thread tests (mapping, filtering, pagination, multi-comment, empty), 3×3 formatter tests (valid, fields/structure, no ANSI)
- L4: `test_ghent_pipe.py` iterm2-driver test — 7/7 pass, screenshots captured
- Fixed verify-visual-tests.sh bash bug: `((content_lines++))` fails with `set -e` when value is 0
- Verification: `go test -race -shuffle=on ./...` ✓, `go build` ✓, `make ci` ✓

### 2026-02-22 (Task 1.4: GitHub API client wiring)
- Created `internal/github/client.go` — Client struct wrapping `*api.GraphQLClient` and `*api.RESTClient`
- Functional options: `WithGraphQLClient`, `WithRESTClient` for test injection
- Compile-time interface checks: ThreadFetcher, CheckFetcher, ThreadResolver, ThreadReplier, ReviewFetcher
- All 6 stub methods return "not implemented" errors
- Wired to Cobra via `PersistentPreRunE` — only creates client for subcommands (not root help/version)
- Exported `GitHubClient()` accessor for subcommands
- Created `internal/github/client_test.go` — 3 test functions: options injection, auth-required path, all stub errors
- go-gh's `api.NewGraphQLClient`/`api.NewRESTClient` with `ClientOptions{Host, AuthToken}` used for test mocks
- Verification: `go test -race -shuffle=on ./...` ✓ (all pass), `go build` ✓, `--help`/`--version` ✓

### 2026-02-22 (Task 1.3: Domain types and port interfaces)
- Created `internal/domain/types.go` — all domain types: ReviewThread, Comment, CommentsResult, CheckRun, Annotation, ChecksResult, OverallStatus, Review, ReviewState, ReplyResult, SummaryResult
- Created `internal/domain/ports.go` — port interfaces: ThreadFetcher, CheckFetcher, ThreadResolver, ThreadReplier, ReviewFetcher, Formatter
- AggregateStatus: fail > pending > pass precedence with short-circuit on fail
- Comment.DatabaseID (int64) for REST reply endpoint
- Used `omitzero` instead of `omitempty` for CheckRun.CompletedAt (time.Time struct — omitempty has no effect)
- Added go-cmp dependency for tests
- Created `internal/domain/types_test.go` — 9 AggregateStatus cases, JSON round-trip, zero-value, DatabaseID key verification
- Verification: `make ci-fast` ✓ (51 tests, lint clean, vet clean)

### 2026-02-22 (Task 1.1: Repository scaffold)
- Initialized Go module (github.com/indrasvat/gh-ghent, Go 1.26)
- Created minimal main.go → cli.Execute() → Cobra root command
- Created internal/version package with ldflags injection
- Created Makefile with full target set (build, test, lint, ci, ci-fast, etc.)
- Created .golangci.yml (golangci-lint v2, 12 linters + goimports)
- Created lefthook.yml (pre-push → make ci)
- Created GitHub Actions: ci.yml (push/PR) + release.yml (gh-extension-precompile)
- Created script stubs: test-binary.sh (L3), test-agent-workflow.sh (L5)
- Updated .gitignore with bin/, coverage/, debug artifacts
- Verification: `make build` ✓, `--help` ✓, `--version` ✓, `make ci` ✓
- Note: go-gh, bubbletea, lipgloss, bubbles, go-cmp not yet in go.mod (unused); will appear in Tasks 1.2-1.4

### 2026-02-22 (Pre-implementation, session 3)
- Created all 24 remaining task files (004-027) for Phases 2-6
- Updated PROGRESS.md with file paths for all 28 tasks
- Total: 28 task files covering the full ghent application

### 2026-02-22 (Pre-implementation, session 2)
- Added `gh ghent reply` command to PRD v1.2 (§6.5) — pipe-only agent command for replying to review threads
- Added Phase 2 task 2.5 (reply), renumbered summary to 2.6
- Updated all cross-references: architecture (§5.1, §5.3), gap analysis (§2.3), domain ports, Phase 1 tasks
- Uses REST API `POST .../comments/{comment_id}/replies` (simpler than GraphQL for single replies)
- Flags: `--thread`, `--body`, `--body-file` (with stdin via `-`)

### 2026-02-22 (Pre-implementation, session 1)
- Completed all research documentation (8 docs, ~7000 lines)
- Created TUI mockups (7 views: comments, expanded, checks, watch, resolve, summary, pipe)
- Created PRD v1.1 with full TUI architecture (Bubble Tea + Lipgloss)
- Created CLAUDE.md with dual-mode operation, TUI pitfalls
- Created PROGRESS.md (this file)
- Created Phase 1 task files (000-003)
- Fixed major error: v1.0 PRD incorrectly omitted TUI — corrected in v1.1
- No code written yet — implementation starts with Task 1.1
