# PRD & Task Conventions for ghent

> How to structure CLAUDE.md, PRD.md, task files, and PROGRESS.md so that AI agents can work effectively without exhausting context.
> Extracted from patterns across nidhi, vivecaka, shux, yukti + the prd-generator skill template.
> Date: 2026-02-22

---

## Table of Contents

1. [The Context Problem](#1-the-context-problem)
2. [Progressive Disclosure Architecture](#2-progressive-disclosure-architecture)
3. [CLAUDE.md Convention](#3-claudemd-convention)
4. [PRD.md Convention](#4-prdmd-convention)
5. [Task File Convention](#5-task-file-convention)
6. [PROGRESS.md Convention](#6-progressmd-convention)
7. [Cross-Reference System](#7-cross-reference-system)
8. [Anti-Patterns to Avoid](#8-anti-patterns-to-avoid)
9. [ghent-Specific Adaptations](#9-ghent-specific-adaptations)

---

## 1. The Context Problem

Across nidhi, vivecaka, shux, and yukti, a recurring failure pattern emerges:

1. **Agent starts a task** and loads CLAUDE.md + PRD.md + task file
2. **PRD alone is 2000-5000 lines** — consumes most of the context window
3. **Mid-implementation**, agent forgets:
   - To run iterm2-driver visual tests
   - To follow project conventions (commit format, error wrapping, etc.)
   - To match UI mockups precisely
   - To use `make ci` instead of bare `go test`
4. **Session compaction** drops critical instructions
5. **Agent apologizes** and the cycle repeats

**Root cause:** Too much information loaded at once. The agent doesn't need the full competitive analysis section of the PRD when implementing a single formatter function.

**Solution:** Progressive disclosure — each document contains exactly what's needed at its level, with explicit references to deeper context when needed.

---

## 2. Progressive Disclosure Architecture

```
                    ┌─────────────────────┐
                    │     CLAUDE.md       │  Always loaded (≤200 lines)
                    │  Build, conventions, │  Core rules, no feature details
                    │  architecture, refs  │
                    └────────┬────────────┘
                             │ references
                    ┌────────▼────────────┐
                    │   PROGRESS.md       │  Loaded on session start
                    │  Current task,      │  Quick context, recovery
                    │  what's next        │  instructions
                    └────────┬────────────┘
                             │ points to
                    ┌────────▼────────────┐
                    │  docs/tasks/NNN.md  │  Loaded for current task only
                    │  Steps, files,      │  Self-contained execution plan
                    │  verification       │  with PRD section refs
                    └────────┬────────────┘
                             │ references (read on demand)
              ┌──────────────┼──────────────┐
              ▼              ▼              ▼
        ┌──────────┐  ┌──────────┐  ┌──────────────┐
        │ PRD §X.Y │  │ research │  │ tui-mockups  │
        │ (subset) │  │  docs/   │  │   .html      │
        └──────────┘  └──────────┘  └──────────────┘
                     Read ONLY the referenced section
```

**What each layer provides:**

| Layer | Content | When Loaded | Max Size |
|-------|---------|-------------|----------|
| CLAUDE.md | Build commands, conventions, architecture, API gotchas | Always (auto) | ≤200 lines |
| PROGRESS.md | Current phase, current task, recovery steps | Session start | ≤100 lines |
| Task file | Execution steps, files to modify, verification | Per task | ≤150 lines |
| PRD section | Feature spec, acceptance criteria | On demand | Section only |
| Research docs | Deep technical context | On demand | Section only |

**Key rule:** No single document should require >200 lines of agent context. If it does, split it or use references.

---

## 3. CLAUDE.md Convention

CLAUDE.md is always loaded into context. It must be **concise and actionable** — conventions and commands, not feature descriptions.

### Structure (target: ≤200 lines)

```markdown
# CLAUDE.md — ghent AI Agent Instructions

> This file is the source of truth for all AI coding agents working on ghent.
> AGENTS.md redirects here.

## Project Overview

ghent (`gh-ghent`) is a GitHub CLI extension for agentic PR monitoring.
Interactive Bubble Tea TUI for humans, structured output (md/json/xml) for AI agents.

Go 1.26 | go-gh v2.13.0 | Cobra v1.10+ | Bubble Tea v1.3+ | Lipgloss v1.1+
golangci-lint v2.9.0 | lefthook 2.1.1 | GoReleaser v2

- PRD: `docs/PRD.md`
- Progress: `docs/PROGRESS.md`
- Tasks: `docs/tasks/NNN-*.md`
- TUI Mockups: `docs/tui-mockups.html` (authoritative visual spec)
- Research: `docs/*.md` (6 research documents)

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
make test-binary        # L3: Run actual binary (scripts/test-binary.sh)
make test-visual        # L4: iterm2-driver visual tests
make test-all           # All levels
```

## Architecture

```
cmd/ghent/main.go            → cli.Execute()
internal/
├── cli/                     # Cobra commands (root, comments, checks, resolve, summary)
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

## Dual-Mode Operation

- **TTY detected** → Launch Bubble Tea TUI (interactive, j/k navigation)
- **Non-TTY / piped** → Output via formatter (--format md|json|xml, default: json)
- **`--no-tui`** → Force pipe mode even in TTY (for agents in pseudo-TTY)
- Detection: `term.FromEnv().IsTerminalOutput()`

## Task Workflow

- Task status: `TODO` → `IN PROGRESS` → `DONE` (or `BLOCKED`)
- Update task file status at start and end of each task
- Update `docs/PROGRESS.md` at end of each session
- If stuck: set status to `BLOCKED`, document issue in PROGRESS.md with details

## Testing Requirements

> **CRITICAL:** Every feature must be verified by _running_ `gh ghent` against a real repo.
> Unit tests alone are NOT sufficient. See `docs/testing-strategy.md` for full details.

- L1: `make test` (unit)
- L2: `make test` with `-tags=integration` (HTTP mocking)
- L3: `make test-binary` (run actual binary)
- L4: `uv run .claude/automations/test_ghent_*.py` (visual verification)
- L5: `scripts/test-agent-workflow.sh` (agent workflow)

## Visual Testing (L4)

iterm2-driver scripts in `.claude/automations/`. Screenshots to `.claude/screenshots/` (gitignored).

```bash
uv run .claude/automations/test_ghent_comments.py
```

Always verify: box integrity, no color bleed, layout matches `docs/tui-mockups.html`.

## Important API Notes

### go-gh v2
- `api.DefaultRESTClient()` / `api.DefaultGraphQLClient()` — auth inherited from gh
- `repository.Current()` — resolves repo from git remote
- `tableprinter.New(w, isTTY, width)` — table in TTY, TSV in pipes
- `term.FromEnv()` — terminal detection
- `gh.Exec()` — run gh subcommands

### GitHub API
- GraphQL `reviewThreads` returns `isResolved`, `path`, `line`, `comments`, `diffHunk`
- `resolveReviewThread` / `unresolveReviewThread` mutations need thread node ID
- REST check runs: `repos/{owner}/{repo}/commits/{ref}/check-runs`
- Job logs: `repos/{owner}/{repo}/actions/jobs/{id}/logs`
- Annotations: `repos/{owner}/{repo}/check-runs/{id}/annotations`
- Rate limits: 5000 REST/hour, 5000 GraphQL points/hour
- Dual field lists for pagination (see `docs/vivecaka-large-pr-patterns-research.md` §1)

### TUI Pitfalls (from yukti/vivecaka)
- Background bleed: use `termenv.SetBackgroundColor()`, NOT `lipgloss.Background()`
- Modal padding: use `strings.Repeat(" ", width)`, NOT empty strings
- ANSI bleed: add explicit `\033[0m` resets between styled elements
- Switch shadowing: `switch typedMsg := msg.(type)` + reassign to outer var
- See `docs/testing-strategy.md` §7 for full list (10 known pitfalls)

## Learnings

> **STRICT RULE:** Update this section at the end of every coding session.
> Format: `- **YYYY-MM-DD (task NNN):** [concrete, actionable insight]`
> Remove entries that become obsolete.

(No entries yet — project is in pre-implementation phase)
```

### What CLAUDE.md does NOT contain
- Feature descriptions (those live in PRD.md)
- Detailed API response schemas (those live in research docs)
- Step-by-step task instructions (those live in task files)
- Progress tracking (that lives in PROGRESS.md)

---

## 4. PRD.md Convention

The PRD should be structured so agents **never need to read the whole thing**. Each section is self-contained, referenced by `§X.Y` from task files.

### Structure

```markdown
# ghent — Product Requirements Document

| Field | Value |
|-------|-------|
| **Version** | 1.0 |
| **Author** | indrasvat |
| **Date** | 2026-02-22 |
| **Status** | Draft |

---

## Table of Contents

1. Vision & Philosophy
2. Problem Statement
3. Target Audience
4. Technology Stack
5. Architecture
6. Functional Requirements
7. Non-Functional Requirements
8. Implementation Phases
9. Testing Strategy
10. Risk Assessment
11. Open Questions
12. Change Log

---

## 1. Vision & Philosophy

### 1.1 One-line Vision
[Single sentence]

### 1.2 Design Principles
| Principle | Meaning | Implementation |
...

### 1.3 What ghent Is NOT
- Not a full TUI (CLI output, not BubbleTea)
- Not a PR review tool (shows comments, doesn't create them)
- ...

---

## 2. Problem Statement
...

## 6. Functional Requirements

### 6.1 `gh ghent comments`

**Purpose:** Show unresolved review threads for a PR.

**Flags:**
- `--pr <number>` — PR number (default: current branch's PR)
- `--format <md|xml|json>` — Output format (default: md)
- `-R <owner/repo>` — Explicit repo

**Behavior:**
1. Resolve repo from git remote (or `-R` flag)
2. Fetch unresolved review threads via GraphQL
3. Group by file path
4. Render in requested format with file:line, author, body, diffHunk context

**Exit codes:**
- 0: No unresolved threads
- 1: Has unresolved threads
- 2: Error (API failure, auth, etc.)

**Acceptance criteria:**
- [ ] FR-001: Shows file path and line number for each thread
- [ ] FR-002: Shows all comments in thread (not just first)
- [ ] FR-003: Includes diffHunk context around the comment
- [ ] FR-004: Groups threads by file
- [ ] FR-005: JSON output parseable by jq
- [ ] FR-006: No ANSI codes in piped output

### 6.2 `gh ghent comments`
...

### 6.3 `gh ghent checks`
...

### 6.4 `gh ghent resolve`
...

### 6.5 `gh ghent summary`
...

### 6.6 Watch Mode
...

### 6.7 Output Formats
...

## 8. Implementation Phases

> **Strategy: CLI-first, TUI second.** Full CLI built and hardened before any TUI code.
> Each task includes Depends On / Parallel With for scheduling.

### Phase 1: Walking Skeleton
| Task | Depends On | Parallel With |
|------|-----------|---------------|
| 1.1: Repository scaffold | — | — |
| 1.2: Cobra CLI skeleton | 1.1 | 1.3 |
| 1.3: Domain types and port interfaces | 1.1 | 1.2 |
| 1.4: GitHub API client wiring | 1.1, 1.3 | — |

### Phase 2: CLI Commands (pipe mode, end-to-end)
| Task | Depends On | Parallel With |
|------|-----------|---------------|
| 2.1: `gh ghent comments` | Phase 1 | 2.2 |
| 2.2: `gh ghent checks` | Phase 1 | 2.1 |
| 2.3: `gh ghent checks --logs` | 2.2 | — |
| 2.4: `gh ghent resolve` | 2.1 | 2.3 |
| 2.5: `gh ghent summary` | 2.1, 2.2 | — |

### Phase 3: CLI Polish
| Task | Depends On | Parallel With |
|------|-----------|---------------|
| 3.1: Watch mode (pipe) | Phase 2 | 3.2 |
| 3.2: Error handling hardening | Phase 2 | 3.1 |
| 3.3: Extension packaging | 3.1, 3.2 | 3.4 |
| 3.4: README + --help | Phase 2 | 3.3 |

> **Milestone: CLI complete**

### Phase 4: TUI Foundation
| Task | Depends On | Parallel With |
|------|-----------|---------------|
| 4.1: Tokyo Night theme + Lipgloss | Phase 3 | — |
| 4.2: Shared TUI components | 4.1 | — |
| 4.3: App shell (root model, views, keys) | 4.2 | — |
| 4.4: Wire TUI to Cobra | 4.3 | — |

### Phase 5: TUI Views
| Task | Depends On | Parallel With |
|------|-----------|---------------|
| 5.1: Comments list view | 4.4 | 5.3, 5.4 |
| 5.2: Comments expanded view | 5.1 | — |
| 5.3: Checks view + log viewer | 4.4 | 5.1, 5.4 |
| 5.4: Resolve view (multi-select) | 4.4 | 5.1, 5.3 |
| 5.5: Summary dashboard | 5.1, 5.3 | — |
| 5.6: Watch mode TUI | 5.3 | — |

### Phase 6: Agent Optimization (Future)
- [ ] Task 6.1: --since flag
- [ ] Task 6.2: --group-by flag
- [ ] Task 6.3: Summary enhancements
- [ ] Task 6.4: Batch resolve
```

### Key PRD Rules

1. **Section numbering is mandatory** — task files reference `§6.1` not "the comments section"
2. **Acceptance criteria use checkboxes** — `- [ ] FR-001: ...` for tracking
3. **Task IDs use X.Y format** — Phase.Task (1.1, 2.3, etc.)
4. **Each functional requirement section is self-contained** — an agent reading §6.1 has everything needed for that command
5. **Keep PRD under 1000 lines** — move competitive analysis, detailed API schemas, and research findings to `docs/` research files and reference them

### What PRD references (doesn't inline)
- Detailed API response schemas → `docs/github-api-research.md`
- Popular extension patterns → `docs/popular-extensions-research.md`
- go-gh SDK details → `docs/gh-extensions-support-research.md`
- Large PR handling patterns → `docs/vivecaka-large-pr-patterns-research.md`
- TUI mockups → `docs/tui-mockups.html`
- Testing strategy → `docs/testing-strategy.md`

---

## 5. Task File Convention

Task files are the primary unit of work. Each is self-contained — an agent should be able to complete a task by reading ONLY:
1. CLAUDE.md (always loaded)
2. The task file itself
3. The specific PRD section(s) referenced

### Structure (target: ≤150 lines)

```markdown
# Task X.Y: [Descriptive Title]

## Status: TODO | IN PROGRESS | DONE | BLOCKED

## Depends On
- Task X.Z (or "None")

## Parallelizable With
- Task X.W (or "None")

## Problem

[2-4 sentences: what's missing and why it matters]

## PRD Reference

- §6.1 (`gh ghent comments`) — flags, behavior, acceptance criteria FR-001 through FR-006
- §5 (Architecture) — domain types and adapter layer

## Research References

- `docs/github-api-research.md` §3 (GraphQL reviewThreads query) — exact query shape
- `docs/gh-extensions-support-research.md` §9 (go-gh API client) — DefaultGraphQLClient usage

## Files to Create

- `internal/github/threads.go` — GraphQL query for unresolved review threads
- `internal/formatter/markdown.go` — Markdown formatter for review threads

## Files to Modify

- `internal/cli/comments.go` — Wire GraphQL client to formatter
- `internal/domain/types.go` — Add ReviewThread, Comment types (if not already present)

## Execution Steps

### Step 1: Read context
1. Read CLAUDE.md (conventions, architecture)
2. Read PRD §6.1 (comments command spec)
3. Read `docs/github-api-research.md` §3 (GraphQL query shape)

### Step 2: Implement domain types
[Specific instructions with code hints, NOT full code blocks]

### Step 3: Implement GraphQL client
[Specific instructions]

### Step 4: Implement markdown formatter
[Specific instructions]

### Step 5: Wire to Cobra command
[Specific instructions]

### Step 6: Unit tests
[What to test, example test case structure]

## Verification

### L1: Unit Tests
```bash
make test
```

### L3: Binary Execution
```bash
make build
gh extension install .
gh ghent comments -R indrasvat/test-repo --pr 1
gh ghent comments -R indrasvat/test-repo --pr 1 --format json | jq .
```

### L4: Visual (iterm2-driver)
Create `.claude/automations/test_ghent_comments.py` following the template in
`docs/testing-strategy.md` §8. Must verify:
- File paths and line numbers visible
- Comment bodies readable
- Box-drawing characters connected
- No background color bleed

## Completion Criteria

1. `make ci` passes (lint + test + vet)
2. `gh ghent comments --pr N` shows unresolved threads
3. `gh ghent comments --format json | jq .` produces valid JSON
4. No ANSI codes in piped output
5. Exit code 0 when no threads, 1 when threads exist
6. iterm2-driver test passes with screenshots verified
7. PROGRESS.md updated

## Commit

```
feat(comments): add review thread display with markdown formatter

- GraphQL query for unresolved review threads
- Markdown output grouped by file path
- JSON/XML format support via --format flag
- Exit codes: 0=clean, 1=unresolved, 2=error
```

## Session Protocol

1. Read CLAUDE.md
2. Read this task file
3. **Change this task's status to `IN PROGRESS`**
4. Read PRD §6.1
5. Read referenced research docs (§ sections only, not full files)
6. Execute steps 1-6
7. Run verification (L1 → L3 → L4)
8. **Change this task's status to `DONE`** (or `BLOCKED` with details)
9. Update `docs/PROGRESS.md` — mark task done + session log entry
10. Update CLAUDE.md Learnings section if new insights discovered
11. Commit with message above
```

### Key Task File Rules

1. **≤150 lines** — if longer, the task is too big; split it
2. **Self-contained** — all context either inline or explicitly referenced by section
3. **PRD references use §X.Y** — never "read the PRD" (too vague, agent reads everything)
4. **Research references use §N** — point to specific section, not whole file
5. **Verification is mandatory** — must include L1 + L3 minimum; L4 for any visual output
6. **Session Protocol at the end** — step-by-step for the agent to follow
7. **Execution steps are imperative** — "Implement X" not "You might want to consider X"

### Naming Convention

```
docs/tasks/
├── 000-repository-scaffold.md        # Phase 1
├── 001-cobra-cli-skeleton.md         # Phase 1
├── 002-domain-types.md               # Phase 1
├── 003-github-api-client.md          # Phase 1
├── 004-comments-command.md           # Phase 2
├── 005-checks-command.md             # Phase 2
├── 006-checks-logs.md                # Phase 2
├── 007-resolve-command.md            # Phase 2
├── 008-summary-command.md            # Phase 2
├── 009-watch-pipe.md                 # Phase 3
├── 010-error-handling.md             # Phase 3
├── 011-extension-packaging.md        # Phase 3
├── 012-readme-help.md                # Phase 3
├── 013-tui-theme-styles.md           # Phase 4
├── 014-tui-components.md             # Phase 4
├── 015-tui-app-shell.md              # Phase 4
├── 016-tui-cobra-wiring.md           # Phase 4
├── 017-tui-comments-view.md          # Phase 5
├── 018-tui-comments-expanded.md      # Phase 5
├── 019-tui-checks-view.md            # Phase 5
├── 020-tui-resolve-view.md           # Phase 5
├── 021-tui-summary-view.md           # Phase 5
└── 022-tui-watch-view.md             # Phase 5
```

Format: `NNN-kebab-case-title.md` with 3-digit zero-padded numbers.

---

## 6. PROGRESS.md Convention

PROGRESS.md serves two purposes:
1. **Quick context** — what's the current state? (for session start)
2. **Recovery** — how to resume after a crash or new session

### Structure (target: ≤100 lines active, session log grows)

```markdown
# ghent — Implementation Progress

> **STRICT RULE:** This file MUST be updated at the end of every coding session.

## Quick Context

| Field | Value |
|-------|-------|
| **Current Phase** | Phase 1: Walking Skeleton |
| **Current Task** | `docs/tasks/004-comments-command.md` |
| **Blocker** | None |
| **Last Action** | Completed domain types (Task 1.4) |
| **Last Updated** | 2026-02-23 |

## How to Resume

1. Read this file — find current task above
2. Read the task file at the path above
3. Read CLAUDE.md (auto-loaded)
4. Read PRD sections referenced in the task file
5. Execute, verify, update this file, commit

## Phase Progress

### Phase 1: Walking Skeleton
- [x] Task 1.1: Repository scaffold — `abc123`
- [x] Task 1.2: Cobra CLI skeleton — `def456`
- [x] Task 1.3: GitHub API client — `789abc`
- [x] Task 1.4: Domain types — `bcd012`

### Phase 2: Core MVP
- [ ] Task 2.1: Comments command
- [ ] Task 2.2: Comments XML/JSON formatters
- [ ] Task 2.3: Checks command
- [ ] Task 2.4: Checks --logs
- [ ] Task 2.5: Resolve command
- [ ] Task 2.6: --watch mode

### Phase 3: Polish & Harden
- [ ] Task 3.1: Error handling
- [ ] Task 3.2: Extension packaging
- [ ] Task 3.3: README, --help
- [ ] Task 3.4: Agent-oriented features

## Session Log

### 2026-02-23
- Task 1.4: Implemented domain types (ReviewThread, CheckRun, ports)
  - Created: `internal/domain/types.go`, `internal/domain/ports.go`
  - Tests: 12 passing, 0 lint issues
  - Learning: go-gh's graphql client requires struct tags matching GraphQL field names exactly

### 2026-02-22
- Task 1.3: Wired go-gh API client
  - Created: `internal/github/client.go`
  - Verified: `gh ghent --version` works
```

### Key PROGRESS.md Rules

1. **Quick Context table is the first thing** — agent reads 5 lines and knows where they are
2. **Completed tasks include commit hash** — for quick verification
3. **Session log is reverse chronological** — newest first
4. **Session entries include:** files created/modified, test metrics, learnings
5. **Keep under 100 lines** of active content (session log grows but old entries can be trimmed)
6. **"How to Resume" is always present** — critical for new sessions

---

## 7. Cross-Reference System

All references use a consistent format:

### From task files to PRD
```
## PRD Reference
- §6.1 (`gh ghent comments`) — flags, behavior, acceptance criteria
```

### From task files to research docs
```
## Research References
- `docs/github-api-research.md` §3 — GraphQL reviewThreads query
```

### From CLAUDE.md to research docs
```
- Dual field lists for pagination (see `docs/vivecaka-large-pr-patterns-research.md` §1)
```

### From PROGRESS.md to task files
```
| **Current Task** | `docs/tasks/004-comments-command.md` |
```

### Section numbering conventions

| Document | Section Format | Example |
|----------|---------------|---------|
| PRD.md | §X.Y | §6.1, §8 |
| Research docs | §N (header number) | §3, §15 |
| Task files | Task X.Y | Task 2.1 |
| CLAUDE.md | ## Header name | "Build & Test Commands" |

---

## 8. Anti-Patterns to Avoid

### 8.1 The Mega-PRD

**Bad:** 3000-line PRD that agents must read in full.
**Good:** 800-line PRD with self-contained sections, detailed specs in research docs.

### 8.2 The Kitchen-Sink CLAUDE.md

**Bad:** 600-line CLAUDE.md with feature descriptions, API schemas, and usage examples.
**Good:** 200-line CLAUDE.md with conventions and references.

**yukti's CLAUDE.md grew to 784 lines** because API learnings, bug fixes, and framework gotchas accumulated. For ghent, framework-specific details should stay in research docs, with only the most critical gotchas (top 5-10) in CLAUDE.md.

### 8.3 The Vague Reference

**Bad:** "Read the PRD for details."
**Good:** "Read PRD §6.1 for the comments command flags and acceptance criteria FR-001 through FR-006."

### 8.4 The Monolith Task

**Bad:** 500-line task file that covers an entire feature end-to-end.
**Good:** Multiple 80-150 line tasks: types → client → formatter → command wiring.

### 8.5 The Missing Verification

**Bad:** Task has execution steps but no verification section.
**Good:** Every task has L1 + L3 minimum. L4 for visual output. L5 for agent-facing features.

### 8.6 The Forgotten Session Protocol

**Bad:** Task ends at completion criteria. Agent doesn't update PROGRESS.md or CLAUDE.md Learnings.
**Good:** Explicit Session Protocol section at the end of every task file.

### 8.7 Duplicating Content Across Files

**Bad:** Repeating the same API gotcha in CLAUDE.md, PRD.md, and 3 task files.
**Good:** Define it once in CLAUDE.md (if critical) or research doc (if detailed), reference from task files.

---

## 9. ghent-Specific Adaptations

### Research docs as the knowledge base

ghent already has 6 research documents totaling ~7000 lines. These serve as the deep reference layer:

| Research Doc | Sections to Reference From Tasks |
|-------------|----------------------------------|
| `gh-extensions-support-research.md` | Extension naming (§1), go-gh SDK (§5-6), Auth (§7), JSON output (§9), testing (§13) |
| `github-api-research.md` | GraphQL review threads (§3), resolveReviewThread (§4), check runs (§5), job logs (§6), annotations (§7) |
| `popular-extensions-research.md` | gh-dash patterns (§1), Cobra patterns (§3), testing (§8) |
| `go-project-patterns-research.md` | Directory structure (§1), Makefile (§3), CLAUDE.md format (§5), Cobra (§7) |
| `dorikin-patterns-research.md` | BubbleTea patterns (§3), error handling (§6) |
| `vivecaka-large-pr-patterns-research.md` | Dual field lists (§1), errgroup (§3), caching (§5), CI aggregation (§13) |

### Testing strategy integration

The testing strategy doc (`docs/testing-strategy.md`) is its own deep reference. Task files should reference specific sections:
- "See `docs/testing-strategy.md` §5 for iterm2-driver script template"
- "See `docs/testing-strategy.md` §7 for TUI pitfalls checklist"

### Dual-mode: Bubble Tea TUI + Pipe Output

ghent operates in two modes based on TTY detection:
- **TTY → Bubble Tea TUI** with Lipgloss styling, j/k navigation, view switching (Tab), interactive resolve
- **Non-TTY → Pipe mode** with `--format md|json|xml` output to stdout
- `--no-tui` flag forces pipe mode even in TTY (for agents in pseudo-TTY)
- Detection: `go-gh/v2/pkg/term.FromEnv().IsTerminalOutput()`

The TUI mockups (`docs/tui-mockups.html`) are the authoritative visual spec — 7 views covering comments, expanded thread, checks, watch, resolve, summary, and pipe mode.

All yukti/vivecaka TUI pitfalls apply directly. See `docs/testing-strategy.md` §7.

### Phase-gated PRD sections

Each implementation phase should only require reading specific PRD sections:

| Phase | PRD Sections Needed |
|-------|-------------------|
| Phase 1 (Walking Skeleton) | §4 (Tech Stack), §5 (Architecture) |
| Phase 2 (CLI Commands) | §6.2-6.5 (commands), §6.7 (formats) |
| Phase 3 (CLI Polish) | §6.6 (Watch Mode), §7 (NFRs) |
| Phase 4 (TUI Foundation) | §5.3 (TUI view architecture), §6.1 (TTY detection), TUI mockups |
| Phase 5 (TUI Views) | §6.2-6.6 (command TUI behavior), TUI mockups |
| Phase 6 (Agent Optimization) | §6.2-6.5 (command extensions) |

This means an agent working on Phase 1 never needs to read §6 (Functional Requirements). The task files enforce this by only referencing the relevant sections.
