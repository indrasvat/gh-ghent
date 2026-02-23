# ghent — Implementation Progress

> **STRICT RULE:** This file MUST be updated at the end of every coding session.
> If a task is BLOCKED, document the issue with full details below.

## Quick Context

| Field | Value |
|-------|-------|
| **Current Phase** | Phase 2: CLI Commands |
| **Current Task** | `docs/tasks/005-checks-command.md` |
| **Blocker** | None |
| **Last Action** | Task 2.1: Comments command — DONE |
| **Last Updated** | 2026-02-22 |

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
- [ ] Task 1.2: Cobra CLI skeleton → `docs/tasks/001-cobra-cli-skeleton.md`
- [x] Task 1.3: Domain types and port interfaces → `docs/tasks/002-domain-types.md`
- [x] Task 1.4: GitHub API client wiring → `docs/tasks/003-github-api-client.md`

### Phase 2: CLI Commands (pipe mode, end-to-end)
- [x] Task 2.1: `gh ghent comments` → `docs/tasks/004-comments-command.md`
- [ ] Task 2.2: `gh ghent checks` → `docs/tasks/005-checks-command.md`
- [ ] Task 2.3: `gh ghent checks --logs` → `docs/tasks/006-checks-logs.md`
- [ ] Task 2.4: `gh ghent resolve` → `docs/tasks/007-resolve-command.md`
- [ ] Task 2.5: `gh ghent reply` → `docs/tasks/008-reply-command.md`
- [ ] Task 2.6: `gh ghent summary` → `docs/tasks/009-summary-command.md`

### Phase 3: CLI Polish
- [ ] Task 3.1: Watch mode (pipe) → `docs/tasks/010-watch-mode-pipe.md`
- [ ] Task 3.2: Error handling hardening → `docs/tasks/011-error-handling.md`
- [ ] Task 3.3: Extension packaging → `docs/tasks/012-extension-packaging.md`
- [ ] Task 3.4: README + --help → `docs/tasks/013-readme-help.md`
- [ ] Task 3.5: Debug logging & tracing → `docs/tasks/028-debug-logging.md`

> **Milestone: CLI complete** — all commands work in pipe mode, fully tested

### Phase 4: TUI Foundation
- [ ] Task 4.1: Tokyo Night theme + Lipgloss styles → `docs/tasks/014-tokyo-night-theme.md`
- [ ] Task 4.2: Shared TUI components → `docs/tasks/015-shared-tui-components.md`
- [ ] Task 4.3: App shell — root model, view switching → `docs/tasks/016-app-shell.md`
- [ ] Task 4.4: Wire TUI to Cobra commands → `docs/tasks/017-wire-tui-cobra.md`

### Phase 5: TUI Views
- [ ] Task 5.1: Comments list view → `docs/tasks/018-comments-list-view.md`
- [ ] Task 5.2: Comments expanded view → `docs/tasks/019-comments-expanded-view.md`
- [ ] Task 5.3: Checks view + log viewer → `docs/tasks/020-checks-view.md`
- [ ] Task 5.4: Resolve view — multi-select → `docs/tasks/021-resolve-view.md`
- [ ] Task 5.5: Summary dashboard → `docs/tasks/022-summary-dashboard.md`
- [ ] Task 5.6: Watch mode TUI → `docs/tasks/023-watch-mode-tui.md`

### Phase 6: Agent Optimization (Future)
- [ ] Task 6.1: --since flag → `docs/tasks/024-since-filter.md`
- [ ] Task 6.2: --group-by flag → `docs/tasks/025-group-by-flag.md`
- [ ] Task 6.3: Summary enhancements → `docs/tasks/026-summary-enhancements.md`
- [ ] Task 6.4: Batch resolve → `docs/tasks/027-batch-resolve.md`

## Blockers

(None currently)

## Session Log

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
- Initialized Go module (github.com/indrasvat/ghent, Go 1.26)
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
