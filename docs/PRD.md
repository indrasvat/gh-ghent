# ghent — Product Requirements Document

| Field | Value |
|-------|-------|
| **Version** | 1.2 |
| **Author** | indrasvat |
| **Date** | 2026-02-22 |
| **Status** | Draft |
| **Feature Guide** | `~/.agent/diagrams/ghent-research-report.html` — authoritative source for all features and agent capabilities |

---

## Table of Contents

1. [Vision & Philosophy](#1-vision--philosophy)
2. [Problem Statement](#2-problem-statement)
3. [Target Audience](#3-target-audience)
4. [Technology Stack](#4-technology-stack)
5. [Architecture](#5-architecture)
6. [Functional Requirements](#6-functional-requirements)
7. [Non-Functional Requirements](#7-non-functional-requirements)
8. [Implementation Phases](#8-implementation-phases)
9. [Testing Strategy](#9-testing-strategy)
10. [Risk Assessment](#10-risk-assessment)
11. [Open Questions](#11-open-questions)
12. [Change Log](#12-change-log)

---

## 1. Vision & Philosophy

### 1.1 One-line Vision

ghent gives developers and AI coding agents instant, structured access to PR review feedback and CI status — through an interactive TUI for humans and structured output for agents.

### 1.2 Design Principles

| Principle | Meaning | Implementation |
|-----------|---------|----------------|
| **Dual-mode** | Interactive TUI in terminal, structured output in pipes | TTY → Bubble Tea TUI; non-TTY → `--format md\|xml\|json` |
| **Agent-first, human-friendly** | Structured for AI parsing, beautiful for humans | JSON/XML for agents, Tokyo Night TUI for developers |
| **Fail-fast, not fail-silent** | Surface problems immediately | `--watch` shows failures as they appear, doesn't wait |
| **Zero config** | Works out of the box if `gh` is authenticated | Auth inherited from gh CLI; repo detected from git remote |
| **Unix philosophy** | Do one thing per command, compose with pipes | Separate commands; meaningful exit codes; pipe-friendly |

### 1.3 What ghent Is NOT

- **Not a PR review tool** — shows existing comments, doesn't create new ones
- **Not a CI dashboard** — shows check status for a single PR, not repo-wide
- **Not a notification system** — pull-based (you run it), not push-based
- **Not a replacement for `gh pr`** — complements it with review-thread and CI-focused views

---

## 2. Problem Statement

### 2.1 For Developers

When working on a PR, developers need to check two things repeatedly:
1. **What review comments are outstanding?** — GitHub's web UI buries unresolved threads across files
2. **What CI checks are failing?** — Navigating to the Actions tab, finding the failing job, scrolling through logs

This context-switching from terminal to browser breaks flow. `gh pr view` shows PR metadata but NOT unresolved review threads or structured CI failure information.

### 2.2 For AI Coding Agents

AI agents (Claude Code, Codex, Cursor, etc.) working on PRs need:
1. **Structured review feedback** — parseable list of what reviewers want changed, with file:line context
2. **CI failure details** — which checks failed, what the error messages are, which files are involved
3. **Resolution capability** — mark threads as resolved after addressing feedback

Current workflow requires agents to scrape `gh pr view` output or make raw API calls. There's no purpose-built tool.

### 2.3 Gap Analysis

| Need | `gh pr view` | `gh pr checks` | **ghent** |
|------|:---:|:---:|:---:|
| Interactive TUI for threads | No | No | **Yes** |
| Unresolved review threads | No | No | **Yes** |
| Thread file:line location | No | No | **Yes** |
| diffHunk context | No | No | **Yes** |
| CI check annotations | No | No | **Yes** |
| Failing job logs | No | No | **Yes** |
| Resolve threads from CLI | No | No | **Yes** |
| Reply to review threads | No | No | **Yes** |
| Summary dashboard | No | No | **Yes** |
| Machine-readable output | `--json` | `--json` | **`--format json\|xml\|md`** |
| Watch mode with fail-fast | No | `--watch` (basic) | **Yes** |

---

## 3. Target Audience

### 3.1 Primary: Developers in Terminal

- Developers who live in the terminal and dislike browser context-switching
- Want an interactive TUI: navigate threads with j/k, expand details, resolve, switch views
- Tokyo Night-themed interface matching modern terminal aesthetics

### 3.2 Primary: AI Coding Agents

- Claude Code, Codex, Gemini CLI, Cursor, Aider
- Need structured output (JSON/XML) with file paths, line numbers, and comment bodies
- Operate in non-TTY environments (pipes, subprocess)
- Use exit codes for control flow (0 = clean, 1 = issues found, 2 = error)

### 3.3 Tertiary: CI/CD Pipelines

- PR quality gates: "fail if there are unresolved review threads"
- Automated status reporting
- Integration with other tools via JSON output

---

## 4. Technology Stack

> **Naming convention:** The GitHub repository is `indrasvat/gh-ghent`, following the `gh-` prefix
> convention required for `gh` extensions. The compiled binary is `gh-ghent`. Users install via
> `gh extension install indrasvat/gh-ghent` and invoke via `gh ghent <subcommand>`.

| Component | Choice | Version | Rationale |
|-----------|--------|---------|-----------|
| Language | Go | 1.26 | gh CLI is Go; go-gh SDK is Go-native |
| GitHub SDK | go-gh | v2.13.0 | Official SDK, inherited auth, API clients |
| CLI framework | Cobra | v1.10+ | Standard for Go CLIs; matches our conventions |
| **TUI framework** | **Bubble Tea** | **v1.3+** | **Elm architecture, proven by gh-dash** |
| **TUI styling** | **Lipgloss** | **v1.1.x** | **Declarative terminal styling** |
| **TUI components** | **Bubbles** | **latest** | **list, viewport, spinner, key bindings** |
| Linter | golangci-lint | v2.9.0 | Curated linter set; v2 for latest rules |
| Formatter | gofumpt | latest | Stricter than gofmt; consistent style |
| Git hooks | lefthook | 2.1.1 | Pre-push → `make ci` |
| Release | gh-extension-precompile | v2 | GitHub Action; handles `gh-ghent-<os>-<arch>` naming + checksums |
| Testing | stdlib + go-cmp | latest | No testify; table-driven tests |

> **Full SDK details:** `docs/gh-extensions-support-research.md` §4 (go-gh library), §5 (API access), §6 (auth)
> **gh-dash TUI patterns:** `docs/popular-extensions-research.md` §3 (gh-dash)
> **TUI pitfalls (yukti/vivecaka):** `docs/testing-strategy.md` §7
> **Convention source:** `docs/go-project-patterns-research.md` §1 (directory), §4 (Cobra), §5 (Makefile)

---

## 5. Architecture

### 5.1 Directory Structure

```
ghent/
├── cmd/ghent/main.go              # Entry point → cli.Execute()
├── internal/
│   ├── cli/                       # Cobra commands
│   │   ├── root.go                # Root command, global flags, version
│   │   ├── comments.go            # gh ghent comments → TUI or pipe
│   │   ├── checks.go              # gh ghent checks → TUI or pipe
│   │   ├── resolve.go             # gh ghent resolve → TUI or pipe
│   │   ├── reply.go               # gh ghent reply → pipe only (agent command)
│   │   └── summary.go             # gh ghent summary → TUI or pipe
│   ├── domain/                    # Types + interfaces (no dependencies)
│   │   ├── types.go               # ReviewThread, CheckRun, Annotation, etc.
│   │   └── ports.go               # ThreadFetcher, CheckFetcher, ThreadResolver, ThreadReplier, Formatter
│   ├── github/                    # GitHub API adapter (implements ports)
│   │   ├── client.go              # go-gh client wiring
│   │   ├── threads.go             # GraphQL: fetch review threads
│   │   ├── checks.go              # REST: fetch check runs + annotations
│   │   ├── logs.go                # REST: fetch job logs
│   │   ├── resolve.go             # GraphQL: resolve/unresolve mutations
│   │   └── reply.go               # REST: reply to review comments
│   ├── tui/                       # Bubble Tea interactive TUI
│   │   ├── app.go                 # Root model, view switching, key routing
│   │   ├── comments.go            # Comments list + expanded thread view
│   │   ├── checks.go              # Checks list + annotation display
│   │   ├── resolve.go             # Multi-select resolve interface
│   │   ├── summary.go             # Dashboard KPI cards + section summaries
│   │   ├── watcher.go             # Watch mode TUI (spinner, progress, event log)
│   │   ├── components/            # Reusable TUI components
│   │   │   ├── statusbar.go       # Top status bar with Lipgloss badges
│   │   │   ├── helpbar.go         # Bottom help bar with key hints
│   │   │   └── diffhunk.go        # Diff hunk renderer (green/red coloring)
│   │   └── styles/                # Lipgloss style definitions
│   │       └── theme.go           # Tokyo Night colors, border styles, badges
│   ├── formatter/                 # Output formatters (pipe/non-TTY mode)
│   │   ├── formatter.go           # Formatter interface + factory
│   │   ├── markdown.go            # Human-readable markdown
│   │   ├── xml.go                 # Structured XML for agents
│   │   └── json.go                # Machine-readable JSON
│   └── version/                   # Build info
│       └── version.go             # Version, commit, date via ldflags
├── scripts/
│   ├── test-binary.sh             # L3: Binary execution tests
│   └── test-agent-workflow.sh     # L5: Agent workflow tests
├── testdata/                      # Test fixtures (GraphQL responses, etc.)
├── Makefile
├── .golangci.yml
├── .goreleaser.yml
├── lefthook.yml
├── go.mod / go.sum
├── CLAUDE.md
├── README.md
└── docs/
```

### 5.2 Dual-Mode Data Flow

```
User runs: gh ghent comments --pr 42

                    ┌───────────────────┐
                    │   Cobra CLI       │
                    │   (comments.go)   │
                    └────────┬──────────┘
                             │
                    ┌────────▼──────────┐
                    │ Is stdout a TTY?  │
                    │ term.FromEnv()    │
                    └───┬──────────┬────┘
                        │          │
                   YES (TTY)    NO (pipe)
                        │          │
               ┌────────▼───┐  ┌───▼──────────┐
               │ Bubble Tea │  │ formatter/   │
               │ tui/app.go │  │ md|json|xml  │
               └────────┬───┘  └───┬──────────┘
                        │          │
               ┌────────▼──────────▼───┐
               │   github/ adapter     │
               │ (GraphQL + REST)      │
               └────────┬──────────────┘
                        │
               ┌────────▼──────────────┐
               │   GitHub API          │
               └───────────────────────┘
```

### 5.3 TUI View Architecture

```
tui/app.go (root model)
├── ViewCommentsList    → tui/comments.go   (bubbles/list, thread list with file:line)
├── ViewCommentsExpand  → tui/comments.go   (bubbles/viewport, full thread + diff hunk + replies)
├── ViewChecksList      → tui/checks.go     (bubbles/list, check runs with auto-expanded annotations)
├── ViewChecksLog       → tui/checks.go     (bubbles/viewport, full job log for selected check)
├── ViewResolve         → tui/resolve.go    (bubbles/list + checkboxes, multi-select resolve)
├── ViewSummary         → tui/summary.go    (lipgloss layout, KPI cards + section summaries)
└── ViewWatch           → tui/watcher.go    (spinner + progress bar + event log)

Shared components:
├── components/statusbar.go   (top bar: repo, PR, counts)
├── components/helpbar.go     (bottom bar: key bindings per view)
└── components/diffhunk.go    (syntax-colored diff hunks)

Tab switching: Tab cycles between comments ↔ checks (top-level views)
View transitions: Enter (expand/drill into), Esc (back), c/k/r (from summary)
Pipe mode: Non-TTY outputs via formatter/ (not a view, handled in cli/ layer)
Reply: No TUI view — pipe-only agent command (cli/reply.go → github/reply.go → formatter/)
```

### 5.4 Key Design Decisions

| Decision | Rationale | Reference |
|----------|-----------|-----------|
| Bubble Tea for TUI | Elm architecture, proven at scale (gh-dash has 10.2k stars) | `docs/popular-extensions-research.md` §3 |
| Dual-mode (TUI + pipe) | Agents need JSON, humans need interactive navigation | `docs/tui-mockups.html` (pipe view) |
| Interface-based ports | Enables testing TUI with mocked data | `docs/popular-extensions-research.md` §14 |
| GraphQL for review threads | REST API doesn't expose `isResolved` on threads | `docs/github-api-research.md` §1 |
| REST for check runs/logs | GraphQL doesn't expose job logs endpoint | `docs/github-api-research.md` §6-7 |
| errgroup for parallel fetch | Graceful degradation — partial results on failure | `docs/vivecaka-large-pr-patterns-research.md` §3 |
| Tokyo Night theme | Modern, widely liked, high contrast | `docs/tui-mockups.html` (theme bar) |
| termenv for background | Prevents color bleed (yukti lesson) | `docs/testing-strategy.md` §7 |

> **Full TUI mockups:** `docs/tui-mockups.html`
> **TUI pitfalls to avoid:** `docs/testing-strategy.md` §7

---

## 6. Functional Requirements

### 6.1 Root Command (`gh ghent`)

**Purpose:** Entry point; shows help and version.

**Flags (persistent, inherited by all subcommands):**
- `-R, --repo <owner/repo>` — Target repository (default: current repo from git remote)
- `--format <md|xml|json>` — Output format for non-TTY mode (default: `json` when piped)
- `--no-tui` — Force pipe mode even in TTY
- `--verbose` — Show additional context (diffHunks in pipe, debug info)
- `--version` — Show version, commit, build date

**Behavior:**
1. With no subcommand: show help text
2. Repo resolution: `-R` flag → `repository.Current()` → error if neither
3. Mode detection: `term.FromEnv().IsTerminalOutput()` → TUI if TTY, pipe if not
4. `--no-tui` overrides TTY detection (for agents running in pseudo-TTY)

**Acceptance criteria:**
- [ ] FR-ROOT-01: `gh ghent --version` prints `ghent vX.Y.Z (commit) (date)`
- [ ] FR-ROOT-02: `gh ghent` with no args shows help with subcommand list
- [ ] FR-ROOT-03: `-R owner/repo` overrides git-remote detection
- [ ] FR-ROOT-04: TTY → launches TUI; piped → outputs formatted text
- [ ] FR-ROOT-05: `--no-tui` forces pipe mode even in TTY

### 6.2 Comments Command (`gh ghent comments`)

**Purpose:** Show unresolved review threads for a PR.

**Flags:**
- `--pr <number>` — PR number (required; future: auto-detect from branch)
- All persistent flags from root

**TUI mode (TTY):** See `docs/tui-mockups.html` — comments view + expanded view
- Status bar: repo, PR number, unresolved/resolved counts
- Thread list: file:line, author, body preview, thread ID, reply count
- Cursor navigation: j/k or arrows, current row highlighted with left-border accent
- Enter: expand thread → full diff hunk, all comments with nested replies, ownership coloring
- Esc: collapse back to list
- n/p: next/prev thread while expanded
- r: resolve current thread
- y: copy thread ID
- o: open thread in browser
- f: filter by file path
- Tab: switch to checks view
- q: quit

**Pipe mode (non-TTY):** Direct output in `--format`
```
pr_number: 42
total_unresolved: 5
threads:
  - file: "internal/api/graphql.go"
    line: 47
    thread_id: "PRRT_abc123"
    is_outdated: false
    comments:
      - author: "reviewer1"
        body: "This should handle the nil case"
        created_at: "2026-02-20T10:00:00Z"
        diff_hunk: "@@ -40,6 +40,8 @@..."
```

**Exit codes:**
- `0` — No unresolved threads
- `1` — Has unresolved threads (user quit TUI with threads remaining)
- `2` — Error (API failure, auth, repo not found)

**Acceptance criteria:**
- [ ] FR-COM-01: TUI shows thread list with file:line, author, body preview
- [ ] FR-COM-02: j/k navigation with cursor row highlighting
- [ ] FR-COM-03: Enter expands thread showing ALL comments + diff hunk
- [ ] FR-COM-04: Threads grouped by file path
- [ ] FR-COM-05: Pipe mode JSON valid and parseable by `jq`
- [ ] FR-COM-06: Pipe mode XML well-formed
- [ ] FR-COM-07: No ANSI escape codes in piped output
- [ ] FR-COM-08: Exit code 0 when no unresolved threads
- [ ] FR-COM-09: Handles PRs with 100+ threads (pagination via `pageInfo.hasNextPage`/`endCursor`)
- [ ] FR-COM-10: Tab switches to checks view
- [ ] FR-COM-11: Client-side filtering of `isResolved` (cannot filter server-side)

**Pagination requirement:** The GraphQL client MUST paginate using `pageInfo.hasNextPage` / `endCursor` with `after` parameter. `reviewThreads(first: 100)` only returns the first page. Loop until `hasNextPage` is `false`. Filter `isResolved` client-side (the API returns all threads regardless of resolution status).

> **GraphQL query:** `docs/github-api-research.md` §1
> **TUI mockup:** `docs/tui-mockups.html` — comments + comments (expanded) tabs
> **Pagination pattern:** `docs/vivecaka-large-pr-patterns-research.md` §2

### 6.3 Checks Command (`gh ghent checks`)

**Purpose:** Show CI check status, failing job details, and annotations for a PR.

**Flags:**
- `--pr <number>` — PR number (required)
- `--logs` — Include failing job log excerpts (pipe mode; TUI always has log access via Enter)
- `--watch` — Poll until all checks complete; see §6.7
- All persistent flags from root

**TUI mode (TTY):** See `docs/tui-mockups.html` — checks view
- Status bar: PR number, HEAD SHA, pass/fail/pending counts
- Check list: icon (pass/fail/pending/running), name, duration, status
- Failed checks auto-expand to show annotations with file:line
- j/k navigation
- Enter: view full log for selected check (bubbles/viewport)
- l: view full log
- o: open check in browser
- R: re-run failed checks
- Tab: switch to comments view
- q: quit

**Pipe mode (non-TTY):**
```
pr_number: 42
head_sha: "abc123"
overall_status: "failure"
checks:
  - name: "lint"
    status: "failure"
    annotations:
      - path: "internal/api/client.go"
        line: 45
        message: "unused variable 'err'"
    log_excerpt: "..."    # With --logs
```

**Exit codes:**
- `0` — All checks passed
- `1` — One or more checks failed
- `2` — Error
- `3` — Checks still pending (with `--watch`: waits; without: reports pending)

**Acceptance criteria:**
- [ ] FR-CHK-01: TUI shows check list with pass/fail icons
- [ ] FR-CHK-02: Failed checks auto-expand annotations
- [ ] FR-CHK-03: Enter opens log viewer (bubbles/viewport)
- [ ] FR-CHK-04: `--watch` activates watch mode (see §6.7)
- [ ] FR-CHK-05: Pipe mode JSON/XML includes annotations
- [ ] FR-CHK-06: `--logs` includes failing step log excerpt in pipe mode
- [ ] FR-CHK-07: Status aggregation: fail > pending > pass
- [ ] FR-CHK-08: Tab switches to comments view

> **REST check runs API:** `docs/github-api-research.md` §6
> **TUI mockup:** `docs/tui-mockups.html` — checks tab
> **CI aggregation:** `docs/vivecaka-large-pr-patterns-research.md` §13

### 6.4 Resolve Command (`gh ghent resolve`)

**Purpose:** Resolve (or unresolve) review threads from the CLI.

**Non-interactive flags (pipe mode bypass):**
- `--thread <id>` — Specific thread ID to resolve
- `--all` — Resolve all unresolved threads
- `--unresolve` — Unresolve instead of resolve

**TUI mode (TTY):** See `docs/tui-mockups.html` — resolve view
- Multi-select interface with checkboxes
- Space: toggle selection on current thread
- a: select all
- j/k: navigate
- Enter: resolve selected (confirmation bar appears first)
- Esc: cancel / deselect all
- Shows spinner during resolution, updates list with resolved status

**Pipe mode (non-TTY):** Requires `--thread` or `--all`
- Prints confirmation per resolved thread (file:line)

**Exit codes:**
- `0` — All specified threads resolved successfully
- `1` — Some threads failed to resolve (partial success)
- `2` — Error (auth, missing permissions)

**Acceptance criteria:**
- [ ] FR-RES-01: TUI multi-select with checkboxes
- [ ] FR-RES-02: Confirmation bar before executing mutations
- [ ] FR-RES-03: `--thread <id>` resolves single thread in pipe mode
- [ ] FR-RES-04: `--all` resolves all threads in pipe mode
- [ ] FR-RES-05: `--unresolve` unresolves threads
- [ ] FR-RES-06: Shows spinner during resolution in TUI
- [ ] FR-RES-07: Requires write permission; clear error if missing
- [ ] FR-RES-08: Respects `viewerCanResolve`/`viewerCanUnresolve` booleans from API

**Permission check:** The GraphQL thread fetch must capture `viewerCanResolve` and `viewerCanUnresolve` boolean fields for each thread. TUI should disable/hide resolve action for threads where `viewerCanResolve` is false. CLI pipe mode should surface a clear permission error before invoking the mutation.

> **GraphQL mutations:** `docs/github-api-research.md` §2-3
> **TUI mockup:** `docs/tui-mockups.html` — resolve tab

### 6.5 Reply Command (`gh ghent reply`)

**Purpose:** Reply to a specific review thread from the CLI. Enables AI agents to acknowledge feedback, ask clarifying questions, or explain fixes directly in the PR conversation.

**Flags:**
- `--pr <number>` — PR number (required)
- `--thread <id>` — Thread node ID to reply to (required; the PRRT_ ID from `gh ghent comments`)
- `--body <text>` — Reply body text (required; supports markdown)
- `--body-file <path>` — Read reply body from file (alternative to `--body`; `-` for stdin)
- All persistent flags from root

**Behavior:**
1. This is a **pipe-first command** — primarily designed for agent use. No TUI view.
2. Fetches the thread to validate it exists and `viewerCanReply` is true
3. Posts the reply via REST API (`POST .../comments/{comment_id}/replies`)
4. The `comment_id` for the reply target is the last comment in the thread (threading semantics)
5. Outputs the created comment details (id, url, body) in `--format`

**Pipe mode (non-TTY and TTY):**
```
# Reply to a thread
gh ghent reply --pr 42 --thread PRRT_abc123 --body "Fixed in commit abc123"

# Reply with multiline body from stdin
echo "Addressed this by..." | gh ghent reply --pr 42 --thread PRRT_abc123 --body-file -

# Reply and get JSON confirmation
gh ghent reply --pr 42 --thread PRRT_abc123 --body "Done" --format json
```

**Output (JSON example):**
```json
{
  "thread_id": "PRRT_abc123",
  "comment_id": 12345678,
  "url": "https://github.com/owner/repo/pull/42#discussion_r12345678",
  "body": "Fixed in commit abc123",
  "created_at": "2026-02-22T10:00:00Z"
}
```

**Exit codes:**
- `0` — Reply posted successfully
- `1` — Thread not found or `viewerCanReply` is false
- `2` — Error (API failure, auth, missing permissions)

**Acceptance criteria:**
- [ ] FR-REP-01: `--thread` + `--body` posts reply to correct thread
- [ ] FR-REP-02: `--body-file` reads body from file; `-` reads from stdin
- [ ] FR-REP-03: Validates `viewerCanReply` before posting; clear error if false
- [ ] FR-REP-04: Output includes comment URL for cross-referencing
- [ ] FR-REP-05: JSON/XML/MD output formats work correctly
- [ ] FR-REP-06: Markdown in body is preserved (not escaped)
- [ ] FR-REP-07: Mutually exclusive: `--body` and `--body-file` cannot both be set

**API note:** The REST API endpoint `POST /repos/{owner}/{repo}/pulls/{pull_number}/comments/{comment_id}/replies` is used rather than GraphQL because it's simpler for single-comment replies and doesn't require creating a review object. The `comment_id` is the numeric ID of the last comment in the thread (available from the GraphQL thread fetch as the `databaseId` field on comment nodes).

> **REST reply endpoint:** `docs/github-api-research.md` §8 (Review Comments)
> **GraphQL viewerCanReply field:** `docs/github-api-research.md` §5 (Key Types)

### 6.6 Summary Command (`gh ghent summary`)

**Purpose:** Dashboard overview of entire PR state — threads, checks, approvals in one view.

**Flags:**
- `--pr <number>` — PR number (required)
- All persistent flags from root

**TUI mode (TTY):** See `docs/tui-mockups.html` — summary view
- KPI cards row: unresolved count, checks passed, checks failed, approvals
- "NOT READY" / "READY" badge in status bar (merge readiness)
- Review Threads section: top threads with "... and N more" truncation
- CI Checks section: failed checks with annotations, passed count
- Approvals section: reviewer status (approved, changes requested)
- Quick-nav: c → comments, k → checks, r → resolve, o → open PR, R → re-run failed

**Pipe mode (non-TTY):** Combined JSON/XML/MD output with all sections

**Exit codes:**
- `0` — PR is merge-ready (no unresolved threads, all checks pass, approvals met)
- `1` — PR is NOT merge-ready
- `2` — Error

**Acceptance criteria:**
- [ ] FR-SUM-01: TUI shows KPI cards with counts
- [ ] FR-SUM-02: Merge readiness badge (READY/NOT READY)
- [ ] FR-SUM-03: Quick-nav keys (c/k/r) jump to full views
- [ ] FR-SUM-04: Pipe mode includes all sections in one response
- [ ] FR-SUM-05: Exit code reflects merge readiness

> **TUI mockup:** `docs/tui-mockups.html` — summary tab

### 6.7 Watch Mode (`--watch` flag on `checks`)

**Purpose:** Poll CI checks until all complete, with fail-fast behavior.

**TUI mode (TTY):** See `docs/tui-mockups.html` — checks --watch view
- `bubbles/spinner` with braille dot animation
- Progress bar: completed/total checks
- Elapsed time counter
- Event log at bottom (bubbles/viewport): real-time status updates
- Poll every 10s (configurable)
- Fail-fast: any check fails → immediately expand to show error, exit code 1
- All pass → exit code 0

**Pipe mode (non-TTY):**
- Print status on each poll as new JSON/MD lines (no cursor movement)
- One status object per poll cycle

**Acceptance criteria:**
- [ ] FR-WAT-01: Spinner animation in TUI mode
- [ ] FR-WAT-02: Progress bar showing completion
- [ ] FR-WAT-03: Event log with timestamps
- [ ] FR-WAT-04: Fail-fast on first failure
- [ ] FR-WAT-05: Ctrl+C exits cleanly
- [ ] FR-WAT-06: Non-TTY outputs one status per line

> **TUI mockup:** `docs/tui-mockups.html` — checks --watch tab
> **Auto-refresh pattern:** `docs/vivecaka-large-pr-patterns-research.md` §11

### 6.8 Output Formats (`--format` flag, pipe mode)

**Three formats for non-TTY/pipe mode:**

| Format | Default When | Use Case |
|--------|-------------|----------|
| `md` (markdown) | explicit `--format md` | Human reading of piped output |
| `json` | piped/non-TTY | AI agents, jq, scripts |
| `xml` | explicit `--format xml` | Agents preferring XML |

**Format guarantees:**
- JSON: valid JSON, parseable by `jq`, no ANSI codes
- XML: well-formed, single root element, no ANSI codes
- Markdown: plain text (no ANSI) since it's pipe mode

**Acceptance criteria:**
- [ ] FR-FMT-01: `--format json` produces valid JSON
- [ ] FR-FMT-02: `--format xml` produces well-formed XML
- [ ] FR-FMT-03: No ANSI codes in any pipe output
- [ ] FR-FMT-04: Default is `json` when piped
- [ ] FR-FMT-05: All three formats contain identical data fields

> **JSON output patterns:** `docs/gh-extensions-support-research.md` §10
> **Pipe mode mockup:** `docs/tui-mockups.html` — pipe mode tab

---

## 7. Non-Functional Requirements

### 7.1 Performance

| Metric | Target | Rationale |
|--------|--------|-----------|
| TUI startup (initial render) | < 500ms | Data fetch is async; show skeleton immediately |
| Cold start (comments, 10 threads) | < 2s | Single GraphQL query |
| Cold start (checks, 10 runs) | < 2s | Single REST call + annotations |
| Watch mode poll overhead | < 1s | Just the API call + render |
| Memory usage | < 50MB | CLI tool, not a daemon |

### 7.2 Reliability

- Graceful degradation: if annotations fetch fails, still show check status
- Rate limit awareness: check `X-RateLimit-Remaining` header, warn at < 100
- Network timeout: 30s per API call, configurable
- Retry: 1 retry on 5xx errors with 1s backoff, no retry on 4xx

### 7.3 Compatibility

- Go 1.26+ (match current toolchain)
- gh CLI 2.80+ (go-gh v2 compatibility)
- Platforms: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64
- Terminal: any terminal supporting ANSI escape codes (iTerm2, Terminal.app, Alacritty, kitty, Windows Terminal)
- CI environments: works in GitHub Actions, GitLab CI, Jenkins (non-TTY → pipe mode)

### 7.4 Security

- No credential storage — inherits from gh CLI
- No network calls except to GitHub API
- No file writes except to stdout/stderr
- Respects `GH_HOST` for GHES

### 7.5 TUI Quality

- **No background color bleed** — use `termenv.SetBackgroundColor()`, NOT `lipgloss.Background()`
- **No ANSI bleed** — explicit `\033[0m` resets between styled elements
- **No modal padding bugs** — use `strings.Repeat(" ", width)`, NOT empty strings
- **No switch shadowing** — use `typedMsg := msg.(type)` and reassign
- Verified via iterm2-driver visual tests (L4) for every TUI feature

> **Full TUI pitfalls list:** `docs/testing-strategy.md` §7

---

## 8. Implementation Phases

> **Strategy: CLI-first, TUI second.** The entire CLI (pipe mode) is built, tested (L1-L3-L5),
> and hardened before any TUI code is written. This ensures the backend/data layer is rock-solid
> and fully verifiable before adding interactive complexity. TUI consumes the same domain types
> and port interfaces — zero duplication.

### Phase 1: Walking Skeleton

Goal: Minimal binary that installs as a gh extension, responds to commands, detects TTY.

| Task | Description | Depends On | Parallel With |
|------|------------|------------|---------------|
| 1.1 | Repository scaffold (go.mod, Makefile, linter, hooks, CI, scripts stubs) | — | — |
| 1.2 | Cobra CLI skeleton (root + 5 subcommands, global flags, TTY detection) | 1.1 | 1.3 |
| 1.3 | Domain types and port interfaces (ReviewThread, CheckRun, Annotation, ports) | 1.1 | 1.2 |
| 1.4 | GitHub API client wiring (go-gh DefaultGraphQLClient + DefaultRESTClient) | 1.1, 1.3 | — |

**PRD sections needed:** §4 (Tech Stack), §5 (Architecture)
**Verification:** L1 + L3 (binary builds, commands respond)

### Phase 2: CLI Commands

Goal: Each command works end-to-end in pipe mode. Every task delivers a fully testable `gh ghent <cmd>`.

| Task | Description | Depends On | Parallel With |
|------|------------|------------|---------------|
| 2.1 | `gh ghent comments` — GraphQL thread fetch + md/json/xml formatters + Cobra wiring | Phase 1 | 2.2 |
| 2.2 | `gh ghent checks` — REST check runs/annotations + status aggregation + formatters | Phase 1 | 2.1 |
| 2.3 | `gh ghent checks --logs` — REST job log fetch + integrate into checks output | 2.2 | — |
| 2.4 | `gh ghent resolve` — GraphQL resolve/unresolve mutations + pipe mode (--thread/--all) | 2.1 | 2.3 |
| 2.5 | `gh ghent reply` — REST reply to review thread + pipe mode (--thread/--body) | 2.1 | 2.4 |
| 2.6 | `gh ghent summary` — aggregate comments + checks data + formatters + Cobra wiring | 2.1, 2.2 | — |

**PRD sections needed:** §6.2-6.6 (commands), §6.8 (formats)
**Verification:** L1 + L3 + L5 per task (run actual binary, verify JSON with jq, check exit codes)

### Phase 3: CLI Polish

Goal: Watch mode works in pipe, error handling is robust, extension is installable via `gh extension install`.

| Task | Description | Depends On | Parallel With |
|------|------------|------------|---------------|
| 3.1 | Watch mode (pipe) — poll loop, fail-fast logic, `--watch` flag on checks | Phase 2 | 3.2 |
| 3.2 | Error handling hardening — rate limits, auth errors, network timeouts | Phase 2 | 3.1 |
| 3.3 | Extension packaging — gh-extension-precompile action, test install flow | 3.1, 3.2 | 3.4 |
| 3.4 | README + --help text + usage examples for all commands | Phase 2 | 3.3 |

**PRD sections needed:** §6.7 (Watch Mode), §7 (NFRs)
**Verification:** L1 + L3 + L5 (full agent workflow tests)

> **Milestone: CLI complete.** After Phase 3, the CLI is fully functional, tested, and publishable.
> All commands work in pipe mode. AI agents can use ghent. Human developers get formatted markdown output.

### Phase 4: TUI Foundation

Goal: Bubble Tea app shell with view switching, shared components, Tokyo Night theme. TUI renders real data from the same backend.

| Task | Description | Depends On | Parallel With |
|------|------------|------------|---------------|
| 4.1 | Tokyo Night theme + Lipgloss style definitions (styles/ package) | Phase 3 | — |
| 4.2 | Shared TUI components — status bar, help bar, diff hunk renderer | 4.1 | — |
| 4.3 | App shell — root model, view enum, Tab switching, key routing | 4.2 | — |
| 4.4 | Wire TUI to Cobra — TTY detection routes to Bubble Tea program | 4.3 | — |

**PRD sections needed:** §5.3 (TUI view architecture), §6.1 (TTY detection)
**TUI mockup:** `docs/tui-mockups.html` — status bar, help bar, theme
**Verification:** L1 + L3 + **L4** (iterm2-driver: app launches, status bar renders, Tab switches views)

### Phase 5: TUI Views

Goal: All interactive views working per mockups.

| Task | Description | Depends On | Parallel With |
|------|------------|------------|---------------|
| 5.1 | Comments list view — thread list, j/k nav, cursor highlighting | 4.4 | 5.3, 5.4 |
| 5.2 | Comments expanded view — viewport, diff hunk, nested replies, n/p nav | 5.1 | — |
| 5.3 | Checks view — check list, auto-expanded annotations, Enter for log viewer | 4.4 | 5.1, 5.4 |
| 5.4 | Resolve view — multi-select checkboxes, confirmation bar, spinner | 4.4 | 5.1, 5.3 |
| 5.5 | Summary dashboard — KPI cards, section summaries, quick-nav (c/k/r) | 5.1, 5.3 | — |
| 5.6 | Watch mode TUI — spinner, progress bar, event log, fail-fast | 5.3 | — |

**PRD sections needed:** §6.2-6.6 (command TUI behavior)
**TUI mockup:** `docs/tui-mockups.html` — all tabs
**Verification:** L1 + L3 + **L4** per view (iterm2-driver: layout matches mockups, no color bleed, box integrity)

### Phase 6: Agent Optimization (Future)

- [ ] Task 6.1: `--since` flag — only show comments/checks after a timestamp
- [ ] Task 6.2: `--group-by` flag — group by file, author, or status
- [ ] Task 6.3: Summary mode enhancements — one-line-per-thread digest
- [ ] Task 6.4: Batch resolve — resolve by file pattern or author

---

## 9. Testing Strategy

> **Full testing strategy:** `docs/testing-strategy.md`

| Level | What | When | Command |
|-------|------|------|---------|
| L1 | Unit tests | Every code change | `make test` |
| L2 | Integration (HTTP mock) | API adapter changes | `make test -tags=integration` |
| L3 | Binary execution | Every feature completion | `make test-binary` |
| **L4** | **Visual (iterm2-driver)** | **Every TUI feature** | **`make test-visual`** |
| L5 | Agent workflow | Format output, exit codes | `scripts/test-agent-workflow.sh` |

**Critical rules:**
- L3 (running `gh ghent`) is MANDATORY for every feature
- **L4 (iterm2-driver screenshots) is MANDATORY for every TUI feature** — verify box integrity, color scheme, layout matches `docs/tui-mockups.html`

> **Known TUI pitfalls:** `docs/testing-strategy.md` §7
> **iterm2-driver templates:** `docs/testing-strategy.md` §5

---

## 10. Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| TUI color bleed / visual bugs | **High** | High | iterm2-driver L4 tests; apply yukti lessons; `docs/testing-strategy.md` §7 |
| GraphQL API changes | Low | High | Pin query fields; integration tests |
| Rate limiting during --watch | Medium | Medium | Respect X-RateLimit-Remaining; backoff |
| Large PR (100+ threads) | Medium | Medium | Pagination with dual field lists |
| Bubble Tea layout bugs on resize | Medium | Medium | Test at multiple terminal sizes via iterm2-driver |
| ANSI bleed in pipe mode | Medium | High | Strip all ANSI in formatter; test with `\| cat` |

---

## 11. Open Questions

| # | Question | Status | Decision |
|---|----------|--------|----------|
| Q1 | Auto-detect PR from current branch? | Open | Requires `gh pr list --head <branch>` call |
| Q2 | Cache GraphQL responses for watch mode? | Deferred | Skip in v1; add if watch mode is slow |
| Q3 | Support GHES (GitHub Enterprise Server)? | Deferred | go-gh handles GH_HOST; test when available |
| Q4 | Mouse support in TUI? | Open | Bubble Tea supports it; nice-to-have |
| Q5 | Default command (no subcommand → summary)? | Open | Could make `gh ghent` alone show summary |

---

## 12. Change Log

| Date | Version | Change |
|------|---------|--------|
| 2026-02-22 | 1.2 | Added `gh ghent reply` command (§6.5), renumbered §6.6-6.8, added Phase 2 task 2.5 |
| 2026-02-22 | 1.1 | Added Bubble Tea TUI, summary command, updated phases and architecture |
| 2026-02-22 | 1.0 | Initial PRD (incorrectly omitted TUI — corrected in v1.1) |
