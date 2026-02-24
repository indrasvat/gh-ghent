---
name: gh-ghent
description: >
  Structured access to GitHub PR review state for AI coding agents. This skill
  should be used when the user needs to check PR merge readiness, diagnose CI
  failures, read unresolved review comments, resolve review threads, or monitor
  CI status. Provides JSON/XML/Markdown output with file:line locations,
  error log excerpts, and annotations from GitHub Actions.
---

# gh-ghent — Agentic PR Monitoring

gh-ghent is a GitHub CLI extension that gives you structured access to PR review
state: unresolved threads, CI check status, annotations, logs, and merge readiness.

**Prerequisite:** `gh extension install indrasvat/gh-ghent`

## When to Use

- Check PR merge readiness in a single call
- Diagnose CI failures with error log excerpts and annotations
- Read unresolved review comments with file:line locations
- Resolve review threads after fixing the requested changes
- Reply to review threads to acknowledge feedback
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

## Recommended: Start with Summary

For most agent tasks, start with `gh ghent summary --pr N --logs --format json --no-tui`.
This single call returns: unresolved threads with file:line, CI check status with annotations
and error log excerpts, approval state, and merge readiness. Only use individual commands
(`comments`, `checks`) when you need deeper detail or specific actions (`resolve`, `reply`).

## Quick Start

### 1. Get full PR status with failure diagnostics
```bash
gh ghent summary --pr 42 --logs --format json --no-tui
```
Returns: unresolved threads, CI checks with annotations and log excerpts, approvals, `is_merge_ready`.
Exit code 0 = merge-ready, 1 = not ready.

### 2. Wait for CI, then get full report
```bash
gh ghent summary --pr 42 --watch --format json --no-tui
```
Polls CI until terminal status (stderr), then outputs full summary (stdout).
Pipe-friendly: `gh ghent summary --pr 42 --watch --format json 2>/dev/null | jq`

### 3. Quick merge-readiness gate
```bash
gh ghent summary --pr 42 --quiet
```
Silent exit 0 if merge-ready, full output + exit 1 if not ready.

### 4. Drill down when needed
```bash
gh ghent comments --pr 42 --format json --no-tui     # Detailed threads
gh ghent checks --pr 42 --logs --format json --no-tui # Detailed check runs
gh ghent resolve --pr 42 --all                         # Batch resolve
gh ghent reply --pr 42 --thread PRRT_... --body "Fixed" # Reply to thread
```

## Commands

| Command | Purpose | Key Flags |
|---------|---------|-----------|
| `summary` | Full PR status + failure diagnostics | `--logs`, `--watch`, `--quiet`, `--compact` |
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

## Core Workflow

```bash
# 1. Get everything in one call
SUMMARY=$(gh ghent summary --pr 42 --logs --format json --no-tui)

# 2. Check merge readiness
echo "$SUMMARY" | jq '.is_merge_ready'

# 3. If CI failing, read the log excerpts
echo "$SUMMARY" | jq '.checks.checks[] | select(.conclusion=="failure") | {name, log_excerpt, annotations}'

# 4. If threads need fixing, read them
echo "$SUMMARY" | jq '.comments.threads[] | {path, line, body: .comments[0].body}'

# 5. Fix code, then resolve + reply
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
gh ghent summary --pr 42 --compact --logs --format json --no-tui
```

Returns a flat object with `pr_age`, `last_update`, `review_cycles`, `is_merge_ready`,
`unresolved` count, `pass_count`, `fail_count`, a `threads` digest, and `failed_checks`
with annotations and log excerpts. Uses minimal tokens — ideal for agent polling loops.

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
