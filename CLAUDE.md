# CLAUDE.md — ghent AI Agent Instructions

> This file is the source of truth for all AI coding agents working on ghent.
> AGENTS.md redirects here.

## Project Overview

ghent (`gh-ghent`) is a GitHub CLI extension for agentic PR monitoring.
Interactive Bubble Tea TUI for humans, structured output (md/json/xml) for AI agents.

Go 1.26 | go-gh v2.13.0 | Cobra v1.10+ | Bubble Tea v1.3+ | Lipgloss v1.1.x
golangci-lint v2.9.0 | lefthook 2.1.1 | gh-extension-precompile v2

- PRD: `docs/PRD.md`
- Progress: `docs/PROGRESS.md`
- Tasks: `docs/tasks/NNN-*.md`
- TUI Mockups: `docs/tui-mockups.html` (authoritative visual spec)
- Research: `docs/*.md` (6 research documents)
- Testing Strategy: `docs/testing-strategy.md`

## Build & Test Commands

```bash
make build              # Build binary
make install            # Build + register as gh extension (symlink)
make test               # Unit tests
make test-race          # Race detector
make lint               # golangci-lint v2
make fmt                # gofumpt
make vet                # go vet
make tidy               # go mod tidy
make ci                 # lint + test + vet (pre-push)
make ci-fast            # fmt + vet + test (quick check)
make clean              # Remove build artifacts
make tools              # Install dev tools
make test-binary        # L3: Run actual binary
make test-visual        # L4: iterm2-driver visual tests
make test-all           # All levels
```

**Local dev:** `make install` symlinks `bin/gh-ghent` into `~/.local/share/gh/extensions/gh-ghent/`,
so `gh ghent` works immediately. The symlink means every `make build` updates what `gh ghent` runs — no reinstall needed.

## Architecture

```
cmd/ghent/main.go            → cli.Execute()
internal/
├── cli/                     # Cobra commands (root, comments, checks, resolve, reply, summary)
├── domain/                  # Types + interfaces (ReviewThread, CheckRun, ports)
├── github/                  # GitHub API adapter (GraphQL + REST via go-gh)
├── tui/                     # Bubble Tea interactive TUI
│   ├── app.go               # Root model, view switching, key routing
│   ├── comments.go          # Comments list + expanded thread
│   ├── checks.go            # Checks list + log viewer
│   ├── resolve.go           # Multi-select resolve
│   ├── summary.go           # Dashboard KPI + sections
│   ├── watcher.go           # Watch mode (spinner, progress, event log)
│   ├── components/          # statusbar, helpbar, diffhunk
│   └── styles/              # Tokyo Night theme, Lipgloss definitions
├── formatter/               # Pipe mode output (markdown, xml, json)
└── version/                 # Build info via ldflags
```

## Dual-Mode Operation

- **TTY detected** → Launch Bubble Tea TUI (interactive, j/k navigation)
- **Non-TTY / piped** → Output via formatter (--format md|json|xml, default: json)
- **`--no-tui`** → Force pipe mode even in TTY (for agents in pseudo-TTY)
- Detection: `term.FromEnv().IsTerminalOutput()`

## Code Conventions

- **Format:** gofumpt (stricter than gofmt)
- **Lint:** golangci-lint v2 with curated linter set
- **Errors:** `fmt.Errorf("context: %w", err)` — always wrap with context
- **No panic:** Return errors, never panic in library code
- **Context:** Pass `context.Context` as first parameter
- **Testing:** Table-driven, stdlib + go-cmp, no testify
- **Imports:** stdlib → external → internal (goimports grouping)
- **Naming:** Unexported by default, export only what's needed
- **Cobra:** `RunE` pattern, shared flags structs, persistent flags on root

## Git Workflow

- Branch: `feat/NNN-short-description` or `fix/NNN-short-description`
- Commits: `type(scope): subject` (feat, fix, refactor, test, docs, chore)
- Hooks: lefthook pre-push → `make ci`
- Never force-push to main

## Task Workflow

- Task status: `TODO` → `IN PROGRESS` → `DONE` (or `BLOCKED`)
- Update task file status at start and end of each task
- Update `docs/PROGRESS.md` at end of each session
- If stuck: set status to `BLOCKED`, document issue in PROGRESS.md with details
- **STRICT: Before starting any task**, query procedural memory (see below) and follow returned rules

## Procedural Memory (cm)

`cm` is the [CASS Memory System](https://github.com/Dicklesworthstone/cass_memory_system) — cross-agent procedural memory that persists patterns across sessions and agents. It has rules from prior Go/TUI projects (vivecaka, yukti, etc.) that are directly relevant to ghent.

```bash
# Install (if `cm` command not found)
curl -fsSL https://raw.githubusercontent.com/Dicklesworthstone/cass_memory_system/main/install.sh \
  | bash -s -- --easy-mode --verify

# REQUIRED before every task — returns scored rules + anti-patterns
cm context "<task description>" --json

# Mark rules during work (in code comments)
// [cass: helpful b-xxx]   — rule was useful
// [cass: harmful b-xxx]   — rule was wrong/misleading

# After task completion
cm outcome success b-xxx,b-yyy   — rules that helped
cm outcome failure b-xxx         — rules that failed
```

Repo-level memory lives in `.cass/` (playbook is committed, diary is gitignored).

## Key Decisions

| Decision | Rationale | Ref |
|----------|-----------|-----|
| Bubble Tea for TUI | Elm architecture, proven at scale (gh-dash) | PRD §4 |
| Dual-mode (TUI + pipe) | Humans need interactive nav, agents need JSON | PRD §5.2 |
| go-gh v2 for API access | Official SDK, inherited auth | PRD §4 |
| GraphQL for review threads | REST doesn't expose `isResolved` | PRD §5.4 |
| REST for check runs/logs | GraphQL doesn't expose job logs | PRD §5.4 |
| Tokyo Night theme | Modern, high contrast, matches mockups | TUI mockups |
| errgroup for parallel fetch | Graceful degradation from vivecaka | PRD §5.4 |

## Testing Requirements

> **CRITICAL:** Every feature must be verified by _running_ `gh ghent` against a real repo.
> Every TUI feature must be verified by iterm2-driver visual tests.
> Unit tests alone are NOT sufficient. See `docs/testing-strategy.md`.

- L1: `make test` (unit)
- L2: `make test` with `-tags=integration` (HTTP mocking)
- L3: `make build` then run `gh ghent` against real repos (see test matrix below)
- L4: `uv run .claude/automations/test_ghent_*.py` (iterm2-driver visual)
- L5: `scripts/test-agent-workflow.sh` (agent workflow)

### L3 Real Repo Test Matrix

Run `make install` first to register with gh, then test against these repos:

| Repo | PR | Threads | Checks | Use For |
|------|-----|---------|--------|---------|
| `indrasvat/tbgs` | #1 | 2 unresolved (`PRRT_kwDOQQ76Ts5iIWqn`, `PRRT_kwDOQQ76Ts5iIWqx`) | pass (3) | comments, resolve, reply |
| `indrasvat/peek-it` | #2 | 1 unresolved | failure (2) | checks with annotations |
| `indrasvat/doot` | #1 | 1 resolved | pass (1) | checks pass, summary merge-ready |
| `indrasvat/visarga` | #1 | 0 | failure (1 fail, 3 skip) | checks failure |
| `indrasvat/querylastic` | #1 | 0 | failure (2) | checks with annotations |
| `indrasvat/context-lens` | #1 | 0 | failure (2 fail, 4 pass) | checks mixed |

```bash
# Quick smoke test after any change:
gh ghent comments -R indrasvat/tbgs --pr 1 --format json | jq '.unresolved_count'  # 2
gh ghent checks -R indrasvat/doot --pr 1 --format json | jq '.overall_status'      # "pass"
gh ghent checks -R indrasvat/peek-it --pr 2 --format json | jq '.overall_status'   # "failure"
```

## TUI Pitfalls (MUST avoid)

These bugs were discovered in yukti/vivecaka. Apply preventively:

1. **Background bleed:** use `termenv.SetBackgroundColor()` before BubbleTea start, `output.Reset()` after exit
2. **AVOID `lipgloss.Background()`** on elements with modal overlays — causes color bleed
3. **Modal padding:** use `strings.Repeat(" ", width)`, NOT empty strings
4. **ANSI bleed:** add explicit `\033[0m` resets between styled elements
5. **Switch shadowing:** `switch typedMsg := msg.(type)` + reassign to outer var
6. **AVOID `lipgloss.Width()`** on inner modal elements — causes padding bleed
7. **WindowSizeMsg:** propagate to ALL sub-models (active AND inactive), not just current view — prevents garbled layout on view switch

> Full list with code examples: `docs/testing-strategy.md` §7

## API Notes

### go-gh v2
- `api.DefaultRESTClient()` / `api.DefaultGraphQLClient()` — auth inherited from gh
- `repository.Current()` — resolves repo from git remote
- `term.FromEnv()` — terminal detection (isTTY, width, colors)
- `gh.Exec()` — run gh subcommands

### GitHub API
- GraphQL `reviewThreads` returns `isResolved`, `path`, `line`, `comments`, `diffHunk`
- `resolveReviewThread` / `unresolveReviewThread` mutations need thread node ID
- REST check runs: `repos/{owner}/{repo}/commits/{ref}/check-runs`
- Job logs: `repos/{owner}/{repo}/actions/jobs/{id}/logs`
- Annotations: `repos/{owner}/{repo}/check-runs/{id}/annotations`
- Rate limits: 5000 REST/hour, 5000 GraphQL points/hour
- Dual field lists for pagination (see `docs/vivecaka-large-pr-patterns-research.md` §1)

## Learnings

> **STRICT RULE:** Update this section at the end of every coding session.
> Format: `- **YYYY-MM-DD (task NNN):** [concrete, actionable insight]`

- **2026-02-22 (task 000):** goimports requires blank line between external (`github.com/spf13/cobra`) and internal (`github.com/indrasvat/gh-ghent/...`) imports — golangci-lint v2 enforces this via the `local-prefixes` setting
- **2026-02-22 (task 000):** go-gh v2.13.0 pins lipgloss to a pre-release commit (`v1.1.1-0.20250319...`), not `@latest` — always let go-gh's version win for lipgloss
- **2026-02-22 (tasks 005/007/008):** `make install` symlinks bin/gh-ghent into gh extensions dir — always use `gh ghent` (not `./bin/gh-ghent`) for L3 testing to match real user experience
- **2026-02-22 (task 007):** `FetchThreads` only returns unresolved threads — any feature needing resolved threads must use `FetchResolvedThreads` (e.g., `--all --unresolve`)
- **2026-02-22 (tasks 005/007/008):** When running parallel agents in worktrees that modify shared files (client.go, formatter.go), agents may step on each other's changes — verify the integrated result builds and passes lint after merging
- **2026-02-23 (task 006):** go-gh REST `DoWithContext` expects JSON responses — use `RequestWithContext` for plain-text endpoints like job logs (`/actions/jobs/{id}/logs`)
- **2026-02-23 (task 006):** Not all check run IDs map to GitHub Actions job IDs — external CI checks (e.g., third-party integrations) return 404 on the logs endpoint. Graceful degradation (skip failed log fetch) is essential.
- **2026-02-23 (task 009):** `gh` extension wrapper may duplicate output to stderr on non-zero exit codes — this is a gh CLI artifact, not a binary bug. Always test with `./bin/gh-ghent` directly to verify
- **2026-02-23 (task 009):** Test repos without real PR approvals will have `is_merge_ready=false` even with clean threads and passing checks — the IsMergeReady logic correctly requires at least 1 APPROVED review
- **2026-02-22 (task 014):** `go get github.com/charmbracelet/bubbletea@latest` can downgrade go-gh from v2.13.0 to v2.11.2 — always re-pin go-gh after adding charmbracelet dependencies: `go get github.com/cli/go-gh/v2@v2.13.0`
- **2026-02-22 (task 016):** golangci-lint `unused` linter catches methods on unexported types — remove unused methods (like `isTopLevel`) rather than keeping them "for later"; re-add when actually needed
- **2026-02-22 (task 017):** L4 iterm2-driver tests may fail to find JSON markers if output is very long and scrolls off screen — check for multiple possible markers including end-of-output fields
- **2026-02-23 (task 023):** When a TUI view has separate `dur` and `status` columns concatenated on the right side, avoid setting the same label in both for a given state — causes duplicate text (e.g., `running... running...`). Only live testing against real in-progress CI caught this; 419 unit tests missed it entirely. Always trigger real CI runs and iterm2-driver test against live data for watch/polling features.
- **2026-02-23 (phase 6):** When running 4 parallel agents in worktrees that all modify `domain/types.go` and `domain/ports.go`, merge one at a time and run `make ci-fast` after each — sequential merge prevents conflicts from compounding. Test count progression (419→443→465→470→489) provides confidence each merge is clean.
- **2026-02-23 (phase 6):** `path.Match` in Go stdlib only matches single path segments — for nested globs like `internal/*/*.go`, it works because `*` doesn't match `/`. For recursive `**` patterns, use `filepath.Match` or a dedicated glob library instead.
- **2026-02-24 (task 031):** Cobra renders `--help`/`--version` templates before `PersistentPreRunE` runs — TTY detection for template funcs must use `term.FromEnv().IsTerminalOutput()` lazily inside the func, not rely on `Flags.IsTTY` which isn't set yet.
- **2026-02-24 (task 031):** Set `cmd.Version` to the raw version string (e.g., `version.Version`), not `version.String()` — custom version templates compose their own layout from `.Version` + helper funcs, so a pre-formatted string causes duplication.
- **2026-02-24 (task 033):** Help bar declarations in `helpbar.go` don't auto-wire handlers — each key advertised in the help bar must have an explicit `key.Matches()` case in the corresponding view's `Update()` method or in `app.handleKey()`. Always verify keybindings with L4 tests after adding them.
- **2026-02-24 (task 033):** For actions that don't visually change the TUI screen (clipboard copy, browser open, async API calls), L4 tests need alternative verification: `pbpaste` for clipboard, `verify_tui_responsive()` for fire-and-forget commands, process spawn checks for browser opens.
- **2026-02-24 (task 034):** When adding `--watch` to a command that already has pipe-mode output, stream watch status to **stderr** and final output to **stdout** — this lets users pipe stdout to `jq` while seeing progress on stderr. Pattern: `WatchChecks(ctx, os.Stderr, f, ...)` then fall through to normal output on `os.Stdout`.
