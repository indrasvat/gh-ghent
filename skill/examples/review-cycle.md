# Example: Review Cycle Walkthrough

A complete walkthrough of using gh-ghent to fix review comments on a real PR.

## Scenario

PR #1 on `indrasvat/tbgs` has 2 unresolved review threads and 3 passing CI checks.

## Step 1: Assess the PR

```bash
gh ghent summary -R indrasvat/tbgs --pr 1 --compact --format json --no-tui
```

Output:
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

Exit code: `1` (not merge-ready — 2 unresolved threads).

## Step 2: Read the Review Threads

```bash
gh ghent comments -R indrasvat/tbgs --pr 1 --format json --no-tui
```

Key fields to extract:

```bash
# Get file:line and first comment body for each thread
gh ghent comments -R indrasvat/tbgs --pr 1 --format json --no-tui | \
  jq -r '.threads[] | "Thread: \(.id)\n  File: \(.path):\(.line)\n  Comment: \(.comments[0].body[0:150])...\n"'
```

Output:
```
Thread: PRRT_kwDOQQ76Ts5iIWqn
  File: internal/tmux/client.go:230
  Comment: **P2** Propagate tmux errors when parsing session:window targets...

Thread: PRRT_kwDOQQ76Ts5iIWqx
  File: internal/app/app.go:88
  Comment: **P2** Honor configured tmux socket path...
```

## Step 3: Extract Thread IDs

```bash
# Save thread IDs for later resolve/reply
THREAD_IDS=$(gh ghent comments -R indrasvat/tbgs --pr 1 --format json --no-tui | jq -r '.threads[].id')
echo "$THREAD_IDS"
```

```
PRRT_kwDOQQ76Ts5iIWqn
PRRT_kwDOQQ76Ts5iIWqx
```

## Step 4: Fix the Code

Read each file at the indicated line, understand the reviewer's request,
and apply the fix. The `diff_hunk` field provides code context.

## Step 5: Preview Resolution (Dry Run)

```bash
gh ghent resolve -R indrasvat/tbgs --pr 1 --all --dry-run --format json --no-tui
```

Output:
```json
{
  "results": [
    {
      "thread_id": "PRRT_kwDOQQ76Ts5iIWqn",
      "path": "internal/tmux/client.go",
      "line": 230,
      "is_resolved": false,
      "action": "would_resolve"
    },
    {
      "thread_id": "PRRT_kwDOQQ76Ts5iIWqx",
      "path": "internal/app/app.go",
      "line": 88,
      "is_resolved": false,
      "action": "would_resolve"
    }
  ],
  "success_count": 2,
  "failure_count": 0,
  "dry_run": true
}
```

## Step 6: Resolve All Threads

```bash
gh ghent resolve -R indrasvat/tbgs --pr 1 --all --format json --no-tui
```

## Step 7: Reply to Confirm

```bash
gh ghent reply -R indrasvat/tbgs --pr 1 \
  --thread PRRT_kwDOQQ76Ts5iIWqn \
  --body "Fixed: error propagation added to GetPanes session:window handling."

gh ghent reply -R indrasvat/tbgs --pr 1 \
  --thread PRRT_kwDOQQ76Ts5iIWqx \
  --body "Fixed: socket path now threaded through to tmux.NewClient()."
```

## Step 8: Verify

```bash
gh ghent summary -R indrasvat/tbgs --pr 1 --compact --format json --no-tui | \
  jq '{merge_ready: .is_merge_ready, unresolved: .unresolved}'
```

Expected after resolving: `{"merge_ready": false, "unresolved": 0}`
(Still not merge-ready because no APPROVED review, but threads are resolved.)

## Using --group-by for Efficient Fixing

If you have many threads across many files, group by file to fix all
comments in one file before moving to the next:

```bash
gh ghent comments -R indrasvat/tbgs --pr 1 --group-by file --format json --no-tui | \
  jq -r '.groups[] | "File: \(.key) (\(.threads | length) threads)"'
```

Output:
```
File: internal/app/app.go (1 threads)
File: internal/tmux/client.go (1 threads)
```
