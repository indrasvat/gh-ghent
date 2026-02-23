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
- L3: `make test-binary` (run actual binary)
- L4: `uv run .claude/automations/test_ghent_*.py` (iterm2-driver visual)
- L5: `scripts/test-agent-workflow.sh` (agent workflow)

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

- **2026-02-22 (task 000):** goimports requires blank line between external (`github.com/spf13/cobra`) and internal (`github.com/indrasvat/ghent/...`) imports — golangci-lint v2 enforces this via the `local-prefixes` setting
- **2026-02-22 (task 000):** go-gh v2.13.0 pins lipgloss to a pre-release commit (`v1.1.1-0.20250319...`), not `@latest` — always let go-gh's version win for lipgloss
