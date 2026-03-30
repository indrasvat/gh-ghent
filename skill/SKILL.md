---
name: gh-ghent
description: >
  Structured GitHub PR state for AI coding agents. Use for PR readiness,
  CI failure diagnosis, unresolved review threads, bounded review waiting,
  bot-review triage, and safe stale-review dismissal. Trigger immediately
  after PR creation and again after every review-fix push. Default path is
  `gh ghent status --await-review --logs --format json --no-tui`; only drop
  to narrower commands when status already tells you what to do next.
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

Use narrower commands (`comments`, `checks`, `resolve`, `reply`, `dismiss`) only when:

- the user asked for a narrow operation directly
- `status` already identified the next specific action
- `status` failed and you need targeted fallback inspection

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
- do **not** start with `comments` or `checks` if a full PR-state decision is needed
- do **not** switch to `gh ghent checks --watch`
- do **not** switch to `gh ghent status --watch`

Bare `--watch` is only for CI-only waiting when review state is irrelevant.

## Bot Sweep (when `unanswered_count > 0`)

The `status` result already contains the full threads. Do not make a second
`comments` call unless you need a narrower filtered view.

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

Default for agents: start with `status`, not `comments` or `checks`.

## Exit Codes

| Command | 0 | 1 | 2 | 3 | 4 |
|---------|---|---|---|---|---|
| `status` | merge-ready | not ready | error | — | — |
| `comments` | no unresolved | has unresolved | error | — | — |
| `checks` | all pass | failure | error | pending | — |
| `resolve` | all success | partial failure | total failure | — | — |
| `reply` | posted | thread not found | error | — | reply ok, resolve failed |
| `dismiss` | all dismissed / no-op / dry-run success | partial dismissal failure | total dismissal failure | — | — |

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
