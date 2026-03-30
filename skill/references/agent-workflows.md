# Agent Workflows

Step-by-step patterns for common PR monitoring tasks with gh-ghent.

## 1. Fix All Review Comments

Read all unresolved threads, fix the code, and resolve each thread.

```bash
# Step 1: Get all unresolved threads
THREADS=$(gh ghent comments --pr 42 --format json --no-tui)

# Step 2: Check if there's work to do
COUNT=$(echo "$THREADS" | jq '.unresolved_count')
if [ "$COUNT" -eq 0 ]; then
  echo "No unresolved threads"
  exit 0
fi

# Step 3: Extract file:line and comment body for each thread
echo "$THREADS" | jq -r '.threads[] | "\(.path):\(.line) — \(.comments[0].body[0:200])"'

# Step 4: Fix each file (agent reads file, applies fix)
# ... your code changes here ...

# Step 5: Resolve all threads at once
gh ghent resolve --pr 42 --all --format json --no-tui

# Step 6: Reply to confirm the fix (optional)
echo "$THREADS" | jq -r '.threads[].id' | while read THREAD_ID; do
  gh ghent reply --pr 42 --thread "$THREAD_ID" --body "Fixed in latest commit."
done
```

### Group by file for efficient batch fixing

```bash
# Get threads grouped by file
GROUPED=$(gh ghent comments --pr 42 --group-by file --format json --no-tui)

# Process one file at a time
echo "$GROUPED" | jq -r '.groups[] | .key' | while read FILE; do
  echo "=== Fixing: $FILE ==="
  echo "$GROUPED" | jq -r --arg f "$FILE" '.groups[] | select(.key == $f) | .threads[] | "  Line \(.line): \(.comments[0].body[0:100])"'
  # ... apply all fixes to this file, then move to next ...
done
```

## 2. Monitor PR Until CI And Reviews Stabilize

Use the same command for CI waiting and review follow-up when comments may still arrive.

```bash
# Recommended when review comments may still matter:
gh ghent status --pr 42 --await-review --logs --format json --no-tui 2>/dev/null | jq
# Watch progress → stderr, full status with review_monitor + logs → stdout.

# CI-only alternative when review state is irrelevant:
gh ghent checks --pr 42 --watch --format json --no-tui
# Exit 0 = all pass, 1 = failure, 3 = still pending after timeout.
```

### Extract failure details from status

```bash
# Get annotations and log excerpts for failed checks
STATUS=$(gh ghent status --pr 42 --logs --format json --no-tui)
echo "$STATUS" | jq -r '.checks.checks[] | select(.log_excerpt) | .annotations[]? | "\(.path):\(.start_line) [\(.annotation_level)] \(.message)"'
echo "$STATUS" | jq -r '.checks.checks[] | select(.log_excerpt) | "--- \(.name) [\(.conclusion)] ---\n\(.log_excerpt)"'
```

### Standalone checks with logs

```bash
# Get log excerpts for failed checks (checks command directly)
gh ghent checks --pr 42 --format json --no-tui --logs | \
  jq -r '.checks[] | select(.log_excerpt != null and .log_excerpt != "") | "--- \(.name) ---\n\(.log_excerpt)"'
```

## 3. Full PR Review Cycle

Complete cycle: assess state, fix comments, fix CI, resolve, verify.

```bash
PR=42

# Step 1: Run the single blessed command
STATUS=$(gh ghent status --pr $PR --await-review --solo --logs --format json --no-tui)

# Step 2: Inspect CI failures if present
echo "$STATUS" | jq '.checks.checks[] | select(.log_excerpt) | {name, annotations, log_excerpt}'

# Step 3: Inspect unanswered review threads if present
echo "$STATUS" | jq '.comments.threads[] | {id, path, line, author: .comments[0].author, body: .comments[0].body}'

# Step 4: Push fixes, then run the same command again
STATUS=$(gh ghent status --pr $PR --await-review --solo --logs --format json --no-tui)

# Step 5: If stale blockers remain from an older head, dismiss only those stale blockers
echo "$STATUS" | jq '.stale_reviews'
gh ghent dismiss --pr $PR --bots-only --message "superseded by current HEAD" --format json --no-tui

# Step 6: Stop only when merge-ready and review monitoring is not low-confidence
echo "$STATUS" | jq '{merge_ready: .is_merge_ready, stale_reviews: .stale_reviews, review_monitor: .review_monitor}'
```

## 4. Clear Stale Blocking Reviews Safely

When a push supersedes an earlier `CHANGES_REQUESTED` review, use `status` to discover stale blockers and `dismiss` to clear only that safe subset.

```bash
STATUS=$(gh ghent status --pr 42 --format json --no-tui)

# Inspect exactly what is stale
echo "$STATUS" | jq '.stale_reviews'

# Preview before acting
gh ghent dismiss --pr 42 --dry-run --format json --no-tui

# Dismiss stale bot blockers only
gh ghent dismiss --pr 42 --bots-only --message "superseded by current HEAD" --format json --no-tui
```

Never dismiss broad review sets. `dismiss` is designed for stale blockers only.

## 5. Incremental Delta Check

Use `--since` to see only new activity since your last check.

```bash
LAST_CHECK="2026-02-23T10:00:00Z"

# New comments since last check
NEW=$(gh ghent comments --pr 42 --since "$LAST_CHECK" --format json --no-tui)
NEW_COUNT=$(echo "$NEW" | jq '.unresolved_count')
echo "New threads since $LAST_CHECK: $NEW_COUNT"

# New check completions since last check
gh ghent checks --pr 42 --since "$LAST_CHECK" --format json --no-tui | \
  jq '{new_checks: [.checks[].name], overall: .overall_status}'

# With relative duration (simpler)
gh ghent comments --pr 42 --since 1h --format json --no-tui
gh ghent checks --pr 42 --since 30m --format json --no-tui
```

## 6. Selective Resolve by File or Author

Resolve only threads matching specific criteria.

```bash
# Preview what would be resolved (dry run)
gh ghent resolve --pr 42 --file "internal/api/*.go" --dry-run --format json --no-tui

# Resolve all threads in a specific directory
gh ghent resolve --pr 42 --file "internal/api/*.go" --format json --no-tui

# Resolve threads from a specific reviewer
gh ghent resolve --pr 42 --author "reviewer-login" --format json --no-tui

# Combine filters (intersection: must match both)
gh ghent resolve --pr 42 --file "*.go" --author "reviewer-login" --format json --no-tui

# Unresolve a thread (reopen for discussion)
gh ghent resolve --pr 42 --thread PRRT_abc123 --unresolve --format json --no-tui
```

## Error Handling

All commands use consistent exit codes:

```bash
output=$(gh ghent comments --pr 42 --format json --no-tui 2>&1)
exit_code=$?

case $exit_code in
  0|1) # Success — parse the JSON
    echo "$output" | jq '.unresolved_count' ;;
  2) # Infrastructure error — log and skip
    echo "Error: $output" >&2 ;;
esac
```

Common errors:
- **Exit 2 + "Not authenticated"** — run `gh auth login`
- **Exit 2 + "Rate limit exceeded"** — wait until reset time shown in message
- **Exit 2 + "not found"** — check repo name and PR number
