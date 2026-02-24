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

| Flag | Type | Description |
|------|------|-------------|
| `--group-by` | string | Group threads by: `file`, `author`, `status` |

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

### Flag Constraints

- `--body` and `--body-file` are mutually exclusive; at least one required
- `--thread` is required

### Exit Codes

- `0` — reply posted
- `1` — thread not found
- `2` — other error

### JSON Output Schema

```json
{
  "thread_id": "PRRT_kwDO...",
  "comment_id": 12345678,
  "url": "https://github.com/owner/repo/pull/1#discussion_r12345678",
  "body": "Fixed in latest commit",
  "created_at": "2026-02-23T12:00:00Z"
}
```

### Stdin Example

```bash
echo "Acknowledged. Fixed in abc123." | gh ghent reply --pr 42 --thread PRRT_abc --body-file -
```

---

## `gh ghent summary`

Combined PR status dashboard with merge-readiness assessment.

### Flags

| Flag | Type | Description |
|------|------|-------------|
| `--compact` | bool | One-line-per-thread compact digest (optimized for agents) |

### Exit Codes

- `0` — merge-ready
- `1` — not merge-ready

### Merge Readiness Logic

`is_merge_ready = true` when ALL three conditions are met:
1. No unresolved threads (`unresolved_count == 0`)
2. All checks pass (`overall_status == "pass"`)
3. At least one APPROVED review with no CHANGES_REQUESTED

### Full JSON Output Schema

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
    "overall_status": "pass",
    "checks": [...],
    "pass_count": 3,
    "fail_count": 0,
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

```json
{
  "pr_number": 1,
  "unresolved": 2,
  "check_status": "pass",
  "pass_count": 3,
  "fail_count": 0,
  "is_merge_ready": false,
  "pr_age": "14w",
  "last_update": "1d",
  "review_cycles": 3,
  "threads": [
    "internal/tmux/client.go:230 (chatgpt-codex-connector)",
    "internal/app/app.go:88 (chatgpt-codex-connector)"
  ]
}
```

Review states: `APPROVED`, `CHANGES_REQUESTED`, `COMMENTED`, `PENDING`, `DISMISSED`.
