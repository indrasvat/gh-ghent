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

## 2. Monitor CI Until Green

Poll CI checks until they complete, then get full summary with failure diagnostics.

```bash
# Recommended: summary --watch gets everything after CI completes
gh ghent summary --pr 42 --watch --logs --format json --no-tui 2>/dev/null | jq
# Watch status → stderr, full summary with logs → stdout.
# After CI completes, parse failures from the summary:
# jq '.checks.checks[] | select(.conclusion=="failure") | {name, log_excerpt, annotations}'

# Alternative: checks --watch for CI-only monitoring
gh ghent checks --pr 42 --watch --format json --no-tui
# Exit 0 = all pass, 1 = failure, 3 = still pending after timeout.
```

### Extract failure details from summary

```bash
# Get annotations and log excerpts for failed checks
SUMMARY=$(gh ghent summary --pr 42 --logs --format json --no-tui)
echo "$SUMMARY" | jq -r '.checks.checks[] | select(.conclusion == "failure") | .annotations[]? | "\(.path):\(.start_line) [\(.annotation_level)] \(.message)"'
echo "$SUMMARY" | jq -r '.checks.checks[] | select(.log_excerpt != null and .log_excerpt != "") | "--- \(.name) ---\n\(.log_excerpt)"'
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

# Step 1: Assess current state
SUMMARY=$(gh ghent summary --pr $PR --compact --format json --no-tui)
echo "$SUMMARY" | jq '{merge_ready: .is_merge_ready, unresolved: .unresolved, checks: .check_status}'

# Step 2: Fix review comments (if any)
UNRESOLVED=$(echo "$SUMMARY" | jq '.unresolved')
if [ "$UNRESOLVED" -gt 0 ]; then
  THREADS=$(gh ghent comments --pr $PR --format json --no-tui)
  # ... read each thread, apply fixes ...
  gh ghent resolve --pr $PR --all --format json --no-tui
fi

# Step 3: Fix CI failures (if any)
CHECK_STATUS=$(echo "$SUMMARY" | jq -r '.check_status')
if [ "$CHECK_STATUS" = "failure" ]; then
  gh ghent checks --pr $PR --format json --no-tui --logs | \
    jq -r '.checks[] | select(.conclusion == "failure") | .annotations[]? | "\(.path):\(.start_line) \(.message)"'
  # ... apply fixes based on annotations ...
fi

# Step 4: Wait for new CI run
sleep 10
gh ghent checks --pr $PR --watch --format json --no-tui

# Step 5: Final verification
if gh ghent summary --pr $PR --format json --no-tui > /dev/null 2>&1; then
  echo "PR is merge-ready!"
else
  echo "Still needs work"
fi
```

## 4. Incremental Delta Check

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

## 5. Selective Resolve by File or Author

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
