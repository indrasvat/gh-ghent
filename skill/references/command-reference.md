# Command Reference

Complete reference for all gh-ghent commands, flags, and output schemas.

## Global Flags (all commands)

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--pr` | | int | (required) | Pull request number |
| `--repo` | `-R` | string | current repo | Repository in `OWNER/REPO` format |
| `--format` | `-f` | string | `json` | Output format: `json`, `md`, `xml` |
| `--no-tui` | | bool | `false` | Force pipe mode even in TTY |
| `--since` | | string | | Filter by time (ISO 8601 or relative: `1h`, `30m`, `2d`, `1w`) |
| `--verbose` | | bool | `false` | Show additional context (diff hunks, debug info) |
| `--debug` | | bool | `false` | Enable debug logging to stderr |

---

## `gh ghent comments`

Show unresolved review threads for a pull request.

### Flags

| Flag | Short | Type | Description |
|------|-------|------|-------------|
| `--group-by` | | string | Group threads by: `file`, `author`, `status` |
| `--bots-only` | `-b` | bool | Show only bot-originated threads |
| `--humans-only` | `-H` | bool | Show only human-originated threads |
| `--unanswered` | `-a` | bool | Show only threads with no replies |

`--bots-only` and `--humans-only` are mutually exclusive.
`--unanswered` is composable: `--bots-only --unanswered` gives unanswered bot threads.

### Exit Codes

- `0` — no unresolved threads
- `1` — has unresolved threads
- `2` — error (auth, rate limit, not found)

### JSON Output Schema

```json
{
  "pr_number": 1,
  "threads": [
    {
      "id": "PRRT_kwDO...",
      "path": "internal/foo/bar.go",
      "line": 42,
      "start_line": 38,
      "is_resolved": false,
      "is_outdated": false,
      "viewer_can_resolve": true,
      "viewer_can_unresolve": false,
      "viewer_can_reply": true,
      "comments": [
        {
          "id": "PRRC_kwDO...",
          "database_id": 12345678,
          "author": "reviewer-login",
          "is_bot": false,
          "body": "Comment body (markdown)",
          "created_at": "2026-02-23T01:32:00Z",
          "url": "https://github.com/owner/repo/pull/1#discussion_r12345678",
          "diff_hunk": "@@ -38,5 +38,10 @@\n+added line...",
          "path": "internal/foo/bar.go"
        }
      ]
    }
  ],
  "total_count": 2,
  "resolved_count": 0,
  "unresolved_count": 2,
  "bot_thread_count": 1,
  "unanswered_count": 0,
  "since": "2026-01-23T00:00:00Z"
}
```

### Grouped Output (with --group-by)

```json
{
  "pr_number": 1,
  "group_by": "file",
  "groups": [
    {
      "key": "internal/app/app.go",
      "threads": [...]
    },
    {
      "key": "internal/tmux/client.go",
      "threads": [...]
    }
  ],
  "total_count": 2,
  "resolved_count": 0,
  "unresolved_count": 2
}
```

### Key Fields for Agents

- `threads[].id` — thread node ID (PRRT_...) — needed for `resolve` and `reply`
- `threads[].path` + `threads[].line` — file location to fix
- `threads[].comments[0].body` — what the reviewer wants changed
- `threads[].comments[0].diff_hunk` — code context around the comment
- `unresolved_count` — quick check if there's work to do

---

## `gh ghent checks`

Show CI check runs, their status, annotations, and log excerpts.

### Flags

| Flag | Type | Description |
|------|------|-------------|
| `--logs` | bool | Include failing job log excerpts in output |
| `--watch` | bool | Poll until all checks complete (fail-fast on failure) |

### Exit Codes

- `0` — all checks pass
- `1` — at least one failure
- `2` — error
- `3` — checks still pending

### JSON Output Schema

```json
{
  "pr_number": 1,
  "head_sha": "abc123...",
  "overall_status": "failure",
  "checks": [
    {
      "id": 12345678,
      "name": "build-test (1.22.x)",
      "status": "completed",
      "conclusion": "failure",
      "started_at": "2026-02-23T00:00:00Z",
      "completed_at": "2026-02-23T00:05:00Z",
      "html_url": "https://github.com/owner/repo/actions/runs/.../job/...",
      "annotations": [
        {
          "path": ".github",
          "start_line": 7,
          "end_line": 7,
          "annotation_level": "failure",
          "title": "",
          "message": "Process completed with exit code 1."
        }
      ],
      "log_excerpt": "error: cannot find module..."
    }
  ],
  "pass_count": 1,
  "fail_count": 1,
  "pending_count": 0
}
```

### Key Fields for Agents

- `overall_status` — "pass", "failure", or "pending"
- `checks[].annotations[]` — structured lint/build errors with file:line
- `checks[].log_excerpt` — error-relevant lines from CI logs (only with `--logs`)
- `checks[].html_url` — link to the check run in GitHub

### Watch Mode (--watch)

In pipe mode, emits NDJSON (one JSON object per poll cycle):
```json
{"timestamp":"...","overall_status":"pending","completed":1,"total":3,"events":[{"name":"Lint","conclusion":"success"}]}
```

Polls every 10 seconds. Exits immediately on first failure (fail-fast).

---

## `gh ghent resolve`

Resolve or unresolve PR review threads.

### Flags

| Flag | Type | Description |
|------|------|-------------|
| `--thread` | string | Thread ID to resolve (PRRT_... node ID) |
| `--all` | bool | Resolve all unresolved threads |
| `--file` | string | Resolve threads in files matching glob (e.g., `internal/api/*.go`) |
| `--author` | string | Resolve threads started by a specific author |
| `--dry-run` | bool | Show what would be resolved without executing |
| `--unresolve` | bool | Unresolve instead of resolve |

### Flag Constraints

- `--thread` is mutually exclusive with `--file`, `--author`, and `--all`
- `--file` and `--author` can be combined (intersection logic)
- `--dry-run` requires `--file`, `--author`, or `--all`

### Exit Codes

- `0` — all resolved successfully (or dry-run)
- `1` — partial failure (some resolved, some failed)
- `2` — total failure (none resolved)

### JSON Output Schema

```json
{
  "results": [
    {
      "thread_id": "PRRT_kwDO...",
      "path": "internal/tmux/client.go",
      "line": 230,
      "is_resolved": true,
      "action": "resolved"
    }
  ],
  "success_count": 2,
  "failure_count": 0,
  "dry_run": false
}
```

When `--dry-run` is set, `action` is `"would_resolve"` (or `"would_unresolve"`).

---

## `gh ghent reply`

Reply to a review thread. Designed primarily for AI agent use.

### Flags

| Flag | Type | Description |
|------|------|-------------|
| `--thread` | string | Thread ID to reply to (PRRT_... node ID, required) |
| `--body` | string | Reply body text (supports markdown) |
| `--body-file` | string | Read reply body from file (use `-` for stdin) |
| `--resolve` | bool | Resolve the thread after posting the reply |

### Flag Constraints

- `--body` and `--body-file` are mutually exclusive; at least one required
- `--thread` is required
- `--resolve` can be combined with any body flag

### Exit Codes

- `0` — reply posted (and resolved if `--resolve`)
- `1` — thread not found
- `2` — other error
- `4` — reply posted but resolve failed (partial success)

### JSON Output Schema

```json
{
  "thread_id": "PRRT_kwDO...",
  "comment_id": 12345678,
  "url": "https://github.com/owner/repo/pull/1#discussion_r12345678",
  "body": "Fixed in latest commit",
  "created_at": "2026-02-23T12:00:00Z",
  "resolved": {
    "thread_id": "PRRT_kwDO...",
    "path": "internal/foo/bar.go",
    "line": 42,
    "is_resolved": true,
    "action": "resolved"
  }
}
```

---

## `gh ghent dismiss`

Dismiss stale blocking reviews that no longer apply to the current PR head.

### Flags

| Flag | Type | Description |
|------|------|-------------|
| `--review` | string | Review node ID (`PRR_...`) or numeric review ID |
| `--author` | string | Restrict to one review author |
| `--bots-only` | bool | Restrict to stale blocking bot reviews |
| `--message` | string | Dismissal message sent to GitHub (required unless `--dry-run`) |
| `--dry-run` | bool | Preview matching stale blockers without dismissing |

### Safety Contract

- Only targets `CHANGES_REQUESTED` reviews
- Only targets stale reviews whose `commit_id` differs from the current PR head
- Never dismisses current reviews

### Exit Codes

- `0` — all selected stale blockers dismissed successfully (or dry-run success)
- `1` — partial failure
- `2` — total failure

### JSON Output Schema

```json
{
  "results": [
    {
      "review_id": "PRR_kwDO...",
      "database_id": 12345678,
      "author": "coderabbitai",
      "is_bot": true,
      "state": "DISMISSED",
      "commit_id": "abc123def456",
      "is_stale": true,
      "dismissed": true,
      "action": "dismissed",
      "message": "superseded by current HEAD",
      "submitted_at": "2026-03-30T12:00:00Z"
    }
  ],
  "success_count": 1,
  "failure_count": 0,
  "dry_run": false
}
```

When `--dry-run` is set, `action` is `"would_dismiss"` and `dismissed` is `false`.

---

## `gh ghent status`

### Additional Review Fields

`status` includes both the full `reviews` array and a helper `stale_reviews` array for automation:

```json
{
  "reviews": [
    {"id": "PRR_kwDO...", "author": "alice", "state": "APPROVED"},
    {"id": "PRR_kwDO...", "author": "coderabbitai", "state": "CHANGES_REQUESTED", "commit_id": "abc123", "is_stale": true}
  ],
  "stale_reviews": [
    {"id": "PRR_kwDO...", "database_id": 12345678, "author": "coderabbitai", "is_bot": true, "state": "CHANGES_REQUESTED", "commit_id": "abc123", "is_stale": true}
  ]
}
```

`stale_reviews` is the safe input set for `gh ghent dismiss`.

### Stdin Example

```bash
echo "Acknowledged. Fixed in abc123." | gh ghent reply --pr 42 --thread PRRT_abc --body-file -
```

---

## `gh ghent status`

Combined PR status dashboard with merge-readiness assessment.

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--compact` | bool | `false` | One-line-per-thread compact digest (optimized for agents) |
| `--logs` | bool | `false` | Include failing job log excerpts and annotations in output |
| `--watch` | bool | `false` | Poll until all checks complete, then output full status |
| `--await-review` | bool | `false` | After CI completes, wait for review activity to settle (implies `--watch`) |
| `--review-timeout` | duration | `5m` | Hard timeout for `--await-review` |
| `--quiet` | bool | `false` | Silent on merge-ready (exit 0), full output on not-ready (exit 1) |
| `--solo` | bool | `false` | Skip approval requirement for single-maintainer repos |

### Exit Codes

- `0` — merge-ready
- `1` — not merge-ready

### Merge Readiness Logic

`is_merge_ready = true` when ALL three conditions are met:
1. No unresolved threads (`unresolved_count == 0`)
2. All checks pass (`overall_status == "pass"`)
3. At least one APPROVED review with no CHANGES_REQUESTED

With `--solo` (or `GH_GHENT_SOLO=1`), condition 3 is relaxed: no approval is required,
but `CHANGES_REQUESTED` still blocks. Useful for personal repos with no collaborators.

### Watch Mode (--watch)

In non-TTY mode, watch progress streams to **stderr** and the final status output to **stdout**.
This lets you pipe the result directly: `gh ghent status --pr 42 --watch --format json 2>/dev/null | jq`

In TTY mode, launches the interactive watch TUI.

### Review Await Mode (--await-review)

After CI checks complete, continues polling for review activity (comments, reviews) to settle.
Uses a lightweight GraphQL activity probe with fingerprint-based change detection — any new
thread, edited thread (via `updatedAt`), resolved thread, new review, or review state change
resets the debounce timer.

**Baseline comparison:** A fingerprint is taken *before* CI watch starts. When the review
phase begins, the current snapshot is compared against this baseline. If different (e.g., a
bot reviewed during CI), activity is detected immediately and the debounce is armed — no
wasted timeout waiting for activity that already happened.

**Debounce:** Enters review quiet detection after 30s of no new review activity. Only fires
after at least one activity change has been detected — prevents premature settlement while a
bot is still working.

**Tail confirmation:** After the first quiet period, ghent performs bounded sparse confirmation
probes before treating the review window as stable. If new activity appears during those probes,
ghent re-arms active review polling automatically.

**Late-activity grace:** If review activity starts right near the timeout boundary, ghent extends
the deadline by a small bounded grace window so it does not cut off a burst mid-stream.

**Hard timeout:** Configurable via `--review-timeout` (default 5m). Safety valve when no
activity is ever detected (e.g., no bot configured, or bot gave silent approval via emoji).

**Head SHA change:** If a new push is detected during review wait, restarts CI watch automatically
(max 3 restarts).

In non-TTY mode, review watch status streams to stderr alongside CI watch status.
The final status output includes a canonical `review_monitor` field:

```json
{
  "review_monitor": {
    "phase": "settled",
    "confidence": "high",
    "activity_count": 3,
    "wait_seconds": 154,
    "tail_probes": 2
  }
}
```

`review_settled` is still emitted today as a compatibility alias for older consumers.

Interpretation:

- `"settled"` + `"high"` → review activity stabilized through bounded confirmation probes
- `"settled"` + `"medium"` → quiet observed, but confirmation was weaker
- `"timeout"` + `"low"` → provisional result; additional bot comments may still arrive

**Agent rule:** When review comments may still matter, always re-run `status --await-review`
after each fix push. Do **not** switch to bare `--watch` for follow-up review cycles.

In TTY mode, the watcher shows "awaiting reviews" with idle/timeout counters,
then transitions to the status dashboard when reviews settle.

Works with any reviewer — Codex, CodeRabbit, Copilot, human reviewers, or any bot
that leaves review comments.

### Quiet Mode (--quiet)

- Merge-ready: exit 0, no output (silence = success)
- Not merge-ready: exit 1, full status output

Ideal for CI gates: `gh ghent status --pr 42 --quiet || echo "Not ready"`

### Full JSON Output Schema

With `--logs`, failing checks include `annotations` and `log_excerpt`:

```json
{
  "pr_number": 1,
  "comments": {
    "pr_number": 1,
    "threads": [...],
    "total_count": 2,
    "resolved_count": 0,
    "unresolved_count": 2
  },
  "checks": {
    "pr_number": 1,
    "head_sha": "abc123...",
    "overall_status": "failure",
    "checks": [
      {
        "id": 12345,
        "name": "build-test",
        "status": "completed",
        "conclusion": "failure",
        "annotations": [
          {"path": "src/main.go", "start_line": 42, "annotation_level": "failure", "message": "unused variable: x"}
        ],
        "log_excerpt": "Error: unused variable x\nsrc/main.go:42:5: x declared and not used"
      }
    ],
    "pass_count": 2,
    "fail_count": 1,
    "pending_count": 0
  },
  "reviews": [
    {
      "id": "PRR_kwDO...",
      "author": "reviewer-login",
      "state": "APPROVED",
      "body": "LGTM",
      "submitted_at": "2026-02-23T00:00:00Z"
    }
  ],
  "is_merge_ready": false,
  "pr_age": "14w",
  "last_update": "1d",
  "review_cycles": 3
}
```

### Compact JSON Output (with --compact)

With `--logs`, includes `failed_checks` with annotations and log excerpts:

```json
{
  "pr_number": 1,
  "unresolved": 2,
  "check_status": "failure",
  "pass_count": 2,
  "fail_count": 1,
  "is_merge_ready": false,
  "pr_age": "14w",
  "last_update": "1d",
  "review_cycles": 3,
  "threads": [
    {"file": "internal/tmux/client.go", "line": 230, "author": "reviewer", "body_preview": "..."}
  ],
  "failed_checks": [
    {
      "name": "build-test",
      "annotations": [{"path": "src/main.go", "line": 42, "level": "failure", "message": "unused variable: x"}],
      "log_excerpt": "Error: unused variable x..."
    }
  ]
}
```

Review states: `APPROVED`, `CHANGES_REQUESTED`, `COMMENTED`, `PENDING`, `DISMISSED`.
