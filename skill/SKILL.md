---
name: gh-ghent
description: >
  Structured access to GitHub PR review state for AI coding agents. This skill
  Structured access to GitHub PR review state for AI coding agents. This skill
  should be used when the user needs to check PR merge readiness, diagnose CI
  failures, read unresolved review comments, resolve review threads, monitor
  CI status, or triage bot reviewer findings (Codex, CodeRabbit, Copilot,
  Cursor Bugbot). Use when the user says "check my PR", "is my PR ready to
  merge", "why is CI failing", "CI is red", "build broken", "what are the
  review comments", "any feedback on my PR", "read PR comments", "address
  review feedback", "resolve review threads", "PR status", "are checks
  passing", "did CI pass", "check PR #N", "monitor CI", "wait for checks",
  "what's failing", "review my PR", "wait for bot review", "wait for Codex",
  "wait for CodeRabbit", "has the bot reviewed yet", "is the review done",
  "sweep bot comments", "fix bot findings", "address bot reviews",
  "what did the bots say", "any bot comments", "triage bot feedback",
  or any request to inspect, diagnose, or act on GitHub pull request
  review threads, CI check status, or merge readiness.
  Provides JSON/XML/Markdown output with file:line locations, error log
  excerpts, and annotations from GitHub Actions.
---

# gh-ghent — Agentic PR Monitoring

gh-ghent is a GitHub CLI extension that gives you structured access to PR review
state: unresolved threads, CI check status, annotations, logs, and merge readiness.

**Prerequisite:** `gh extension install indrasvat/gh-ghent`

## When to Use

- Check PR merge readiness in a single call
- Diagnose CI failures with error log excerpts and annotations
- Read unresolved review comments with file:line locations
- **Sweep bot findings** — filter to bot-only or human-only threads, triage, reply+resolve in one step
- Resolve review threads after fixing the requested changes
- Reply to review threads to acknowledge feedback
- Monitor CI in a polling loop until checks complete
- Wait for bot reviewers (Codex, CodeRabbit, Copilot, Cursor Bugbot) to finish reviewing

## Agent Mode

Always use these flags for structured, parseable output:

```bash
gh ghent <command> --pr <N> --format json --no-tui
```

- `--no-tui` forces pipe mode even in pseudo-TTY environments
- `--format json` returns structured JSON (default in pipe mode)
- `--format xml` returns tagged XML with semantic attributes
- `--format md` returns readable Markdown

## Solo Mode — Auto-Detect for Personal Repos

`--solo` skips the approval requirement in merge-readiness checks. Without it, `is_merge_ready`
is always `false` on repos with no reviewers — common for personal projects.

**When to add `--solo` automatically:** Run this check before your first `gh ghent` call:

```bash
# Check if authenticated user owns the repo and it's a personal (non-org) account
OWNER_TYPE=$(gh api repos/{owner}/{repo} --jq '.owner.type' 2>/dev/null)
OWNER_LOGIN=$(gh api repos/{owner}/{repo} --jq '.owner.login' 2>/dev/null)
MY_LOGIN=$(gh api user --jq '.login' 2>/dev/null)

if [ "$OWNER_TYPE" = "User" ] && [ "$OWNER_LOGIN" = "$MY_LOGIN" ]; then
  SOLO_FLAG="--solo"
fi
```

**Heuristics (cheapest → most definitive):**

| Check | API Cost | Confidence | When to Use |
|-------|----------|------------|-------------|
| `owner.type == "User"` AND `owner.login == gh_user` | 0 extra (repo metadata already fetched) | High | Default — covers most personal repos |
| Collaborator count == 1 (`gh api repos/.../collaborators --jq 'length'`) | 1 call | Definitive | When owner check isn't enough (e.g., fork of org repo) |
| PR has 0 APPROVED/CHANGES_REQUESTED reviews in summary output | 0 extra (post-hoc) | Medium | Fallback: if `is_merge_ready=false` and only the approval condition failed, suggest `--solo` |

**Rules:**
- For org repos (`owner.type == "Organization"`), never auto-add `--solo`
- `--solo` still blocks on `CHANGES_REQUESTED` — it only relaxes the "needs approval" requirement
- Prefer `GH_GHENT_SOLO=1` env var for persistent config (set in shell profile)

## Start Here: One Command, Then Branch

Every agent workflow starts with one call. The response tells you what to do next.

```bash
gh ghent summary --pr 42 --await-review --solo --logs --format json --no-tui
```

This blocks until CI passes and bot reviewers settle, then returns everything:
threads (with `is_bot` per comment), checks (with log excerpts), reviews, and `is_merge_ready`.

**Decision tree from the summary response:**

| Condition | Action |
|---|---|
| `is_merge_ready == true` | Merge the PR |
| `checks.overall_status == "failure"` | Fix CI — log excerpts and annotations are inline |
| `comments.unanswered_count > 0` | Bot sweep — threads are inline, fix and `reply --resolve` each |
| `comments.unresolved_count > 0` | Resolve remaining threads — `resolve --all` or per-thread |

**Bot sweep loop** (when `unanswered_count > 0`):
1. Read bot threads from `comments.threads[]` where `comments[0].is_bot == true`
2. Fix code → push
3. `gh ghent reply --pr 42 --thread PRRT_... --body "Fixed" --resolve` per thread
4. `gh ghent summary --pr 42 --await-review --solo --format json --no-tui` (bots may re-trigger)
5. Repeat until `is_merge_ready == true`

No second `comments` call needed — the summary has the full threads. Use `comments --bots-only --unanswered`
only when you want a filtered drill-down view.

## Other Common Calls

```bash
# Merge-readiness gate (silent exit 0 if ready, full output + exit 1 if not)
gh ghent summary --pr 42 --quiet --solo

# CI only (no review wait)
gh ghent summary --pr 42 --watch --logs --format json --no-tui

# Drill-down: bot threads only, human threads only, all threads
gh ghent comments --pr 42 --bots-only --unanswered --format json --no-tui
gh ghent comments --pr 42 --humans-only --format json --no-tui
gh ghent checks --pr 42 --logs --format json --no-tui

# Batch resolve all threads at once
gh ghent resolve --pr 42 --all
```

## Commands

| Command | Purpose | Key Flags |
|---------|---------|-----------|
| `summary` | Full PR status + failure diagnostics | `--logs`, `--watch`, `--await-review`, `--review-timeout`, `--quiet`, `--compact`, `--solo` |
| `comments` | Unresolved review threads | `--group-by`, `--bots-only`, `--humans-only`, `--unanswered` |
| `checks` | CI status + annotations | `--logs`, `--watch` |
| `resolve` | Resolve/unresolve threads | `--thread`, `--all`, `--file`, `--author`, `--dry-run`, `--unresolve` |
| `reply` | Reply to a thread | `--thread`, `--body`, `--body-file`, `--resolve` |

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
| `--solo` | | `false` | Skip approval requirement (or `GH_GHENT_SOLO=1`) |
| `--debug` | | `false` | Debug logging to stderr |

## Tips

- No bot reviewer configured? Use `--watch` instead of `--await-review` to skip the review timeout.
- Output incomplete? Add `--debug` to see API calls, 404s, and rate limits on stderr.

## Exit Codes

Exit codes let you branch logic without parsing output:

| Command | 0 | 1 | 2 | 3 | 4 |
|---------|---|---|---|---|---|
| `summary` | merge-ready | not ready | error | — | — |
| `comments` | no unresolved | has unresolved | error | — | — |
| `checks` | all pass | failure | error | pending | — |
| `resolve` | all success | partial failure | total failure | — | — |
| `reply` | posted | thread not found | error | — | reply ok, resolve failed |

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
