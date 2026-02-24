# Example: CI Monitor Walkthrough

A complete walkthrough of monitoring CI checks, extracting errors, fixing them,
and verifying the fix.

## Scenario

PR #2 on `indrasvat/peek-it` has failing CI checks with annotations.
PR #1 on `indrasvat/visarga` has a mix of failed and skipped checks.

## Step 1: Check CI Status

```bash
gh ghent checks -R indrasvat/peek-it --pr 2 --format json --no-tui
```

Key fields:
```bash
gh ghent checks -R indrasvat/peek-it --pr 2 --format json --no-tui | \
  jq '{status: .overall_status, pass: .pass_count, fail: .fail_count}'
```

Output:
```json
{
  "status": "failure",
  "pass": 0,
  "fail": 2
}
```

Exit code: `1` (failure detected).

## Step 2: Extract Failing Check Names

```bash
gh ghent checks -R indrasvat/peek-it --pr 2 --format json --no-tui | \
  jq -r '.checks[] | select(.conclusion != "success" and .conclusion != "neutral" and .conclusion != "skipped") | "\(.name): \(.conclusion)"'
```

Output:
```
build-test (1.23.x): cancelled
build-test (1.22.x): failure
```

## Step 3: Extract Annotations (Structured Errors)

```bash
gh ghent checks -R indrasvat/peek-it --pr 2 --format json --no-tui | \
  jq -r '.checks[].annotations[]? | "\(.path):\(.start_line) [\(.annotation_level)] \(.message)"'
```

Output:
```
.github:1 [failure] The strategy configuration was canceled because "build-test._1_22_x" failed
.github:203 [failure] The operation was canceled.
.github:17 [warning] Restore cache failed: Dependencies file is not found...
.github:7 [failure] Process completed with exit code 1.
.github:17 [warning] Restore cache failed: Dependencies file is not found...
```

## Step 4: Get Log Excerpts (if available)

```bash
gh ghent checks -R indrasvat/peek-it --pr 2 --format json --no-tui --logs | \
  jq -r '.checks[] | select(.log_excerpt != null and .log_excerpt != "") | "--- \(.name) ---\n\(.log_excerpt)\n"'
```

Note: Log excerpts are only available for GitHub Actions jobs. External CI checks
(third-party integrations) may return empty log excerpts — this is expected graceful degradation.

## Step 5: Mixed Check Scenario

```bash
gh ghent checks -R indrasvat/visarga --pr 1 --format json --no-tui | \
  jq -r '.checks[] | "\(.name): \(.conclusion)"'
```

Output:
```
Tests: skipped
Tauri build (macOS): skipped
Lint: skipped
Format: failure
```

## Step 6: Use summary --watch for Full Report After CI

Wait for CI to complete and get a full summary with failure diagnostics in one call:

```bash
# Watch status goes to stderr, final summary to stdout
gh ghent summary -R owner/repo --pr 42 --watch --logs --format json --no-tui 2>/dev/null | jq
```

This is the recommended approach: it waits for CI, then fetches threads + reviews + checks
with log excerpts in a single output. Parse failures:
```bash
gh ghent summary --pr 42 --watch --logs --format json --no-tui 2>/dev/null | \
  jq '.checks.checks[] | select(.conclusion=="failure") | {name, log_excerpt, annotations}'
```

### Alternative: checks --watch for CI-only monitoring

```bash
# Blocks until all checks complete or a failure is detected
gh ghent checks -R owner/repo --pr 42 --watch --format json --no-tui
```

Watch mode:
- Polls every 10 seconds
- Fail-fast: exits immediately when any check fails
- Exit 0 = all passed, Exit 1 = failure detected, Exit 3 = timed out still pending

## Step 7: CI Fix Cycle

After fixing the code and pushing, monitor the new CI run:

```bash
# Push your fix
git add -A && git commit -m "fix: resolve CI failures" && git push

# Wait a moment for CI to start
sleep 10

# Watch and get full report with failure diagnostics
gh ghent summary --pr 42 --watch --logs --format json --no-tui 2>/dev/null | \
  jq '{merge_ready: .is_merge_ready, checks: .check_status, failures: [.checks.checks[] | select(.conclusion=="failure") | .name]}'
```

## Step 8: Use --since for Only New Checks

If you've already seen previous check results and only want new completions:

```bash
# Checks completed in the last 30 minutes
gh ghent checks --pr 42 --since 30m --format json --no-tui | \
  jq '{status: .overall_status, checks: [.checks[].name]}'
```

## Multiple Failing Repos Pattern

Check CI across multiple PRs efficiently:

```bash
for repo_pr in "indrasvat/peek-it:2" "indrasvat/visarga:1" "indrasvat/context-lens:1"; do
  REPO="${repo_pr%%:*}"
  PR="${repo_pr##*:}"
  STATUS=$(gh ghent checks -R "$REPO" --pr "$PR" --format json --no-tui 2>/dev/null | jq -r '.overall_status')
  echo "$REPO PR#$PR: $STATUS"
done
```

Output:
```
indrasvat/peek-it PR#2: failure
indrasvat/visarga PR#1: failure
indrasvat/context-lens PR#1: failure
```
