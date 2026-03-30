# Exit Codes Reference

gh-ghent uses structured exit codes so agents can branch logic without parsing output.

## Exit Code Table

| Command | Exit 0 | Exit 1 | Exit 2 | Exit 3 |
|---------|--------|--------|--------|--------|
| `status` | PR is merge-ready | Not merge-ready | Auth/rate-limit/not-found error | — |
| `comments` | No unresolved threads | Has unresolved threads | Auth/rate-limit/not-found error | — |
| `checks` | All checks pass | At least one failure | Auth/rate-limit/not-found error | Checks still pending |
| `resolve` | All threads resolved successfully | Partial failure (some resolved) | Total failure (none resolved) | — |
| `reply` | Reply posted successfully | Thread not found | Other error | — |
| `dismiss` | All stale blockers dismissed, no-op success, or dry-run success | Partial failure | Total failure | — |

## Exit Code 2

Exit code meanings are command-specific. Always use the per-command table above as
the authoritative mapping.

For `status`, `comments`, and `checks`, exit 2 is an infrastructure/access error:

| Error Type | Message Pattern |
|------------|-----------------|
| Authentication | `Not authenticated. Run 'gh auth login' first.` |
| Rate limit | `Rate limit exceeded. Resets at HH:MM.` |
| Not found (repo) | `PR #N in owner/repo not found.` |
| Not found (PR) | `PR #N in owner/repo not found.` |

For `resolve` and `dismiss`, exit 2 is a command-specific total failure state, not
necessarily an infrastructure error.

## Conditional Patterns

### Check if PR is merge-ready
```bash
if gh ghent status --pr 42 --format json --no-tui > /dev/null 2>&1; then
  echo "PR is ready to merge"
else
  echo "PR is not ready"
fi
```

### Branch on CI status
```bash
gh ghent checks --pr 42 --format json --no-tui > /dev/null 2>&1
case $? in
  0) echo "All checks pass" ;;
  1) echo "Checks failed — extract annotations" ;;
  3) echo "Checks still running — wait or poll" ;;
  2) echo "Error accessing checks" ;;
esac
```

### Resolve-then-verify loop
```bash
# Resolve all threads
gh ghent resolve --pr 42 --all --format json --no-tui

# Verify none remain
if gh ghent comments --pr 42 --format json --no-tui > /dev/null 2>&1; then
  echo "All threads resolved"
else
  echo "Some threads still unresolved"
fi
```

### Dismiss stale blockers safely
```bash
gh ghent dismiss --pr 42 --dry-run --format json --no-tui > /dev/null 2>&1
case $? in
  0) echo "Dry-run succeeded, nothing matched, or all dismissals succeeded" ;;
  1) echo "Some stale blockers were dismissed, some failed" ;;
  2) echo "Every attempted dismissal failed" ;;
esac
```

### Poll until CI completes
```bash
while true; do
  gh ghent checks --pr 42 --format json --no-tui > /dev/null 2>&1
  status=$?
  case $status in
    0) echo "All pass"; break ;;
    1) echo "Failure detected"; break ;;
    3) echo "Still pending, waiting..."; sleep 30 ;;
    2) echo "Error"; break ;;
  esac
done
```

### Guard against errors
```bash
output=$(gh ghent status --pr 42 --format json --no-tui 2>&1)
exit_code=$?

if [ $exit_code -eq 2 ]; then
  echo "Error: $output" >&2
  exit 1
fi

# Safe to parse JSON
echo "$output" | jq '.is_merge_ready'
```
