---
name: gh-ghent
description: >
  Monitor and act on GitHub PR reviews with gh-ghent. Use when working with
  pull requests that have unresolved review comments, failing CI checks, or
  need thread resolution and replies. Provides structured JSON/XML/Markdown
  output optimized for AI coding agents.
---

# gh-ghent — Agentic PR Monitoring

gh-ghent is a GitHub CLI extension that gives you structured access to PR review
state: unresolved threads, CI check status, annotations, logs, and merge readiness.

**Prerequisite:** `gh extension install indrasvat/gh-ghent`

## When to Use

Use gh-ghent when you need to:
- Read unresolved review comments on a PR and understand what reviewers want changed
- Check CI status, read failing annotations, and extract error log excerpts
- Resolve review threads after fixing the requested changes
- Reply to review threads to acknowledge feedback or ask clarifying questions
- Assess whether a PR is ready to merge (threads + checks + approvals)
- Monitor CI in a polling loop until checks complete

## Agent Mode

Always use these flags for structured, parseable output:

```bash
gh ghent <command> --pr <N> --format json --no-tui
```

- `--no-tui` forces pipe mode even in pseudo-TTY environments
- `--format json` returns structured JSON (default in pipe mode)
- `--format xml` returns tagged XML with semantic attributes
- `--format md` returns readable Markdown

## Quick Start

### 1. Get full PR status in one call
```bash
gh ghent summary --pr 42 --format json --no-tui
```
Returns: unresolved thread count, CI check status, approval state, `is_merge_ready` boolean.
Exit code 0 = merge-ready, 1 = not ready.

### 2. Read unresolved review threads
```bash
gh ghent comments --pr 42 --format json --no-tui
```
Returns: array of threads with `id`, `path`, `line`, `comments[].body`, `comments[].diff_hunk`.
Exit code 0 = no unresolved threads, 1 = has unresolved threads.

### 3. Check CI status with error details
```bash
gh ghent checks --pr 42 --format json --no-tui --logs
```
Returns: `overall_status` ("pass"/"failure"/"pending"), per-check annotations, log excerpts.
Exit code 0 = all pass, 1 = failure, 3 = pending.

## Commands

| Command | Purpose | Key Flags |
|---------|---------|-----------|
| `summary` | Full PR dashboard | `--compact` |
| `comments` | Unresolved review threads | `--group-by file\|author\|status` |
| `checks` | CI status + annotations | `--logs`, `--watch` |
| `resolve` | Resolve/unresolve threads | `--thread`, `--all`, `--file`, `--author`, `--dry-run`, `--unresolve` |
| `reply` | Reply to a thread | `--thread`, `--body`, `--body-file` |

For complete flag reference, see [references/command-reference.md](references/command-reference.md).

## Global Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--pr` | | (required) | PR number |
| `--repo` | `-R` | current repo | Repository in `OWNER/REPO` format |
| `--format` | `-f` | `json` | Output format: `json`, `md`, `xml` |
| `--no-tui` | | `false` | Force pipe mode (for agents in pseudo-TTY) |
| `--since` | | | Filter by time: ISO 8601 or relative (`1h`, `30m`, `2d`, `1w`) |
| `--verbose` | | `false` | Include additional context |
| `--debug` | | `false` | Debug logging to stderr |

## Core Workflow: Fix All Review Comments

```bash
# 1. Read what reviewers want changed
THREADS=$(gh ghent comments --pr 42 --format json --no-tui)

# 2. For each thread: read file:line, understand the comment, fix the code
echo "$THREADS" | jq -r '.threads[] | "\(.path):\(.line) — \(.comments[0].body)"'

# 3. After fixing, resolve each thread
echo "$THREADS" | jq -r '.threads[].id' | while read id; do
  gh ghent resolve --pr 42 --thread "$id"
done

# 4. Or batch-resolve all at once
gh ghent resolve --pr 42 --all
```

## Exit Codes

Exit codes let you branch logic without parsing output:

| Command | 0 | 1 | 2 | 3 |
|---------|---|---|---|---|
| `summary` | merge-ready | not ready | error | — |
| `comments` | no unresolved | has unresolved | error | — |
| `checks` | all pass | failure | error | pending |
| `resolve` | all success | partial failure | total failure | — |
| `reply` | posted | thread not found | error | — |

Error (exit 2): authentication failure, rate limit, or resource not found.

For detailed exit code usage patterns, see [references/exit-codes.md](references/exit-codes.md).

## Incremental Monitoring with --since

Use `--since` to see only what changed since your last check:

```bash
# Only threads with activity in the last hour
gh ghent comments --pr 42 --since 1h --format json --no-tui

# Only checks completed in the last 30 minutes
gh ghent checks --pr 42 --since 30m --format json --no-tui
```

Supported formats: `1h`, `30m`, `2d`, `1w`, `24h`, or ISO 8601 (`2026-02-23T00:00:00Z`).

## Compact Summary for Minimal Context

```bash
gh ghent summary --pr 42 --compact --format json --no-tui
```

Returns a flat object with `pr_age`, `last_update`, `review_cycles`, `is_merge_ready`,
`unresolved` count, `pass_count`, `fail_count`, and a one-line `threads` digest.
Uses minimal tokens — ideal for agent polling loops.

## Group Comments for Batch Fixing

```bash
# Group by file — fix all comments in one file at a time
gh ghent comments --pr 42 --group-by file --format json --no-tui

# Group by author — address each reviewer's feedback together
gh ghent comments --pr 42 --group-by author --format json --no-tui
```

## Additional Resources

- [Command Reference](references/command-reference.md) — all commands, all flags, output schemas
- [Agent Workflows](references/agent-workflows.md) — step-by-step patterns for common tasks
- [Exit Codes](references/exit-codes.md) — branching logic and conditional patterns
- [Review Cycle Example](examples/review-cycle.md) — full walkthrough: read, fix, resolve, reply
- [CI Monitor Example](examples/ci-monitor.md) — watch CI, extract errors, fix, re-check
