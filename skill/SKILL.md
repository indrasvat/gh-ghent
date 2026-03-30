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
  Also handles stale blocking reviews via safe, stale-only dismissal.
---

# gh-ghent — Agentic PR Monitoring

**Prerequisite:** `gh extension install indrasvat/gh-ghent`

**All commands require:** `--pr <N> --format json --no-tui`

Get PR number: `gh pr view --json number -q .number`

## First Command After PR Creation Or Review-Fix Push

```bash
PR=$(gh pr view --json number -q .number)
gh ghent status --pr $PR --await-review --solo --logs --format json --no-tui
```

This is the **single blessed command** for PR review handling.
Use it:

- immediately after PR creation
- again after **every push** that addresses review or CI feedback

It waits for CI, performs bounded review monitoring, and returns everything in one response:

- threads with `is_bot`
- checks with log excerpts
- reviews
- `review_monitor`
- `is_merge_ready`

**Drop `--solo`** for org repos with required review policies.
**Always include `--await-review`** when review comments may still arrive.
**Do not switch to bare `--watch`** after the first cycle if review comments still matter — `--watch` is CI-only and can miss follow-up bot comments.
**Drop `--logs`** only on narrow re-checks where CI failure detail is definitely not needed.

## Response Shape (status)

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
  "reviews": [{"author": "alice", "state": "APPROVED"}],
  "stale_reviews": [{"id": "PRR_...", "author": "coderabbitai", "state": "CHANGES_REQUESTED", "commit_id": "abc123", "is_stale": true}],
  "review_monitor": {
    "phase": "settled",
    "confidence": "high",
    "activity_count": 3,
    "wait_seconds": 154
  }
}
```

## Decision Order

Act on the **first matching** condition — fix it, then re-run status:

1. **Exit code 2** → auth / rate limit / not-found error. Fix credentials.
2. **`checks.overall_status == "failure"`** → Fix CI. Log excerpts and annotations are inline.
3. **`checks.overall_status == "pending"`** → Re-run the **same** `status --await-review` command. Do not switch to `--watch` while review comments may still appear.
4. **`comments.unanswered_count > 0`** → Bot sweep (see below).
5. **`stale_reviews | length > 0`** → Dismiss only those stale blockers: `gh ghent dismiss --pr <N> --message "superseded by current HEAD"` (optionally `--bots-only`).
6. **`comments.unresolved_count > 0`** → `gh ghent resolve --pr <N> --all`
7. **`review_monitor.phase == "timeout"` or `review_monitor.confidence == "low"`** → Treat result as provisional. If you just pushed fixes, re-run the **same** `status --await-review` command after the push settles.
8. **`is_merge_ready == true` and `review_monitor.confidence != "low"`** → Merge / stop.

## Anti-Footgun Rule

When review comments may still arrive:

- use `gh ghent status --await-review ...`
- after every fix push, use `gh ghent status --await-review ...` again
- do **not** switch to `gh ghent checks --watch`
- do **not** switch to `gh ghent status --watch`

Bare `--watch` is only for CI-only waiting when review state is irrelevant.

## Bot Sweep (when `unanswered_count > 0`)

The status already contains the full threads — no second call needed.

1. Read threads from `comments.threads[]` where `comments[0].is_bot == true`
2. Fix code → push
3. Per thread: `gh ghent reply --pr <N> --thread PRRT_... --body "Fixed" --resolve`
4. Re-check with the **same** command: `gh ghent status --pr <N> --await-review --solo --logs --format json --no-tui`
5. Repeat until `is_merge_ready == true` and `review_monitor.confidence != "low"`

## Solo Mode

Add `--solo` only when the repo owner is the authenticated user on a personal (non-org) repo.
Never auto-add for org repos. If merge readiness is false only because approval is missing
on a personal repo, retry with `--solo`.

## Commands

| Command | Purpose | Key Flags |
|---------|---------|-----------|
| `status` | Full PR status + merge readiness | `--logs`, `--watch`, `--await-review`, `--quiet`, `--compact`, `--solo` |
| `comments` | Unresolved review threads | `--bots-only`, `--humans-only`, `--unanswered`, `--group-by` |
| `checks` | CI status + annotations | `--logs`, `--watch` |
| `resolve` | Resolve/unresolve threads | `--thread`, `--all`, `--file`, `--author`, `--unresolve`, `--dry-run` |
| `reply` | Reply to a thread | `--thread`, `--body`, `--body-file`, `--resolve` |
| `dismiss` | Dismiss stale blocking reviews only | `--review`, `--author`, `--bots-only`, `--message`, `--dry-run` |

## Exit Codes

| Command | 0 | 1 | 2 | 3 | 4 |
|---------|---|---|---|---|---|
| `status` | merge-ready | not ready | error | — | — |
| `comments` | no unresolved | has unresolved | error | — | — |
| `checks` | all pass | failure | error | pending | — |
| `resolve` | all success | partial failure | total failure | — | — |
| `reply` | posted | thread not found | error | — | reply ok, resolve failed |
| `dismiss` | all dismissed / dry-run success | partial dismissal failure | total dismissal failure | — | — |

Exit 2 = auth failure, rate limit, or resource not found.

## Other Patterns

```bash
# Merge-readiness gate (silent exit 0 if ready, exit 1 + full output if not)
gh ghent status --pr <N> --quiet --solo

# Drill-down: bot threads only
gh ghent comments --pr <N> --bots-only --unanswered --format json --no-tui

# Group by file for batch fixing
gh ghent comments --pr <N> --group-by file --format json --no-tui

# Compact status (minimal tokens for polling loops)
gh ghent status --pr <N> --compact --format json --no-tui

# Clear stale blocking bot reviews after a superseding push
gh ghent dismiss --pr <N> --bots-only --message "superseded by current HEAD" --format json --no-tui
```

## References

- [Command Reference](references/command-reference.md) — all flags, full output schemas
- [Agent Workflows](references/agent-workflows.md) — step-by-step patterns
- [Exit Codes](references/exit-codes.md) — branching logic
- [Review Cycle Example](examples/review-cycle.md) — read, fix, resolve, reply walkthrough
- [CI Monitor Example](examples/ci-monitor.md) — watch CI, extract errors, fix, re-check
