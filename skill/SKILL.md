---
name: gh-ghent
description: >
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
  IMPORTANT - also trigger this skill immediately after creating a PR
  (gh pr create, git push + PR URL) to monitor CI and review state.
  Provides JSON/XML/Markdown output with file:line locations, error log
  excerpts, and annotations from GitHub Actions.
---

# gh-ghent — Agentic PR Monitoring

**Prerequisite:** `gh extension install indrasvat/gh-ghent`

**All commands require:** `--pr <N> --format json --no-tui`

Get PR number: `gh pr view --json number -q .number`

## First Command After PR Creation

```bash
PR=$(gh pr view --json number -q .number)
gh ghent summary --pr $PR --watch --logs --format json --no-tui
```

`--watch` blocks until CI completes, then returns everything. If bots review during CI,
their findings are already in the response. The agent does NOT need to know whether bots
exist, whether the repo is personal or org, or whether humans will review — the response
data tells it what happened.

### Other entrypoints (non-default)

- **Inspect now** (non-blocking): `gh ghent summary --pr <N> --format json --no-tui`
- **Re-check after fix** (wait for bot re-review): `gh ghent summary --pr <N> --await-review --format json --no-tui`

Use `--await-review` only in round 2+ of the bot sweep loop, when you know bots will re-trigger.

## Response Shape (summary)

```json
{
  "is_merge_ready": false,
  "comments": {
    "threads": [{"id": "PRRT_...", "path": "foo.go", "line": 42,
                  "comments": [{"author": "coderabbitai", "is_bot": true, "body": "..."}]}],
    "unresolved_count": 2,
    "bot_thread_count": 2,
    "unanswered_count": 1
  },
  "checks": {
    "overall_status": "pass",
    "checks": [{"name": "CI", "conclusion": "success", "annotations": [], "log_excerpt": ""}]
  },
  "reviews": [{"author": "alice", "state": "APPROVED"}]
}
```

## Decision Order

Act on the **first matching** condition — fix it, then re-run summary:

1. **Exit code 2** → auth / rate limit / not-found error. Fix credentials.
2. **`checks.overall_status == "failure"`** → Fix CI. Log excerpts and annotations are inline.
3. **`checks.overall_status == "pending"`** → Re-run with `--watch` to wait.
4. **`comments.unanswered_count > 0`** → Bot sweep (see below).
5. **`comments.unresolved_count > 0`** → `gh ghent resolve --pr <N> --all`
6. **`is_merge_ready == true`** → Merge.

## Bot Sweep (when `unanswered_count > 0`)

The summary already contains the full threads — no second call needed.

1. Read threads from `comments.threads[]` where `comments[0].is_bot == true`
2. Fix code → push
3. Per thread: `gh ghent reply --pr <N> --thread PRRT_... --body "Fixed" --resolve`
4. Re-check: `gh ghent summary --pr <N> --await-review --format json --no-tui` (bots may re-trigger)
5. Repeat until `is_merge_ready == true`

## Solo Mode

Add `--solo` only when the repo owner is the authenticated user on a personal (non-org) repo.
Never auto-add for org repos. If merge readiness is false only because approval is missing
on a personal repo, retry with `--solo`.

## Commands

| Command | Purpose | Key Flags |
|---------|---------|-----------|
| `summary` | Full PR status + merge readiness | `--logs`, `--watch`, `--await-review`, `--quiet`, `--compact`, `--solo` |
| `comments` | Unresolved review threads | `--bots-only`, `--humans-only`, `--unanswered`, `--group-by` |
| `checks` | CI status + annotations | `--logs`, `--watch` |
| `resolve` | Resolve/unresolve threads | `--thread`, `--all`, `--file`, `--author`, `--unresolve`, `--dry-run` |
| `reply` | Reply to a thread | `--thread`, `--body`, `--body-file`, `--resolve` |

## Exit Codes

| Command | 0 | 1 | 2 | 3 | 4 |
|---------|---|---|---|---|---|
| `summary` | merge-ready | not ready | error | — | — |
| `comments` | no unresolved | has unresolved | error | — | — |
| `checks` | all pass | failure | error | pending | — |
| `resolve` | all success | partial failure | total failure | — | — |
| `reply` | posted | thread not found | error | — | reply ok, resolve failed |

Exit 2 = auth failure, rate limit, or resource not found.

## Other Patterns

```bash
# Merge-readiness gate (silent exit 0 if ready, exit 1 + full output if not)
gh ghent summary --pr <N> --quiet --solo

# Drill-down: bot threads only
gh ghent comments --pr <N> --bots-only --unanswered --format json --no-tui

# Group by file for batch fixing
gh ghent comments --pr <N> --group-by file --format json --no-tui

# Compact summary (minimal tokens for polling loops)
gh ghent summary --pr <N> --compact --format json --no-tui
```

## References

- [Command Reference](references/command-reference.md) — all flags, full output schemas
- [Agent Workflows](references/agent-workflows.md) — step-by-step patterns
- [Exit Codes](references/exit-codes.md) — branching logic
- [Review Cycle Example](examples/review-cycle.md) — read, fix, resolve, reply walkthrough
- [CI Monitor Example](examples/ci-monitor.md) — watch CI, extract errors, fix, re-check
