# Exit Codes Reference

gh-ghent uses structured exit codes so agents can branch logic without parsing output.

## Exit Code Table

| Command | Exit 0 | Exit 1 | Exit 2 | Exit 3 |
|---------|--------|--------|--------|--------|
| `summary` | PR is merge-ready | Not merge-ready | Auth/rate-limit/not-found error | — |
| `comments` | No unresolved threads | Has unresolved threads | Auth/rate-limit/not-found error | — |
| `checks` | All checks pass | At least one failure | Auth/rate-limit/not-found error | Checks still pending |
| `resolve` | All threads resolved successfully | Partial failure (some resolved) | Total failure (none resolved) | — |
| `reply` | Reply posted successfully | Thread not found | Other error | — |

## Error Exit Code (2)

Exit code 2 always means an infrastructure error, not a PR state issue:

| Error Type | Message Pattern |
|------------|-----------------|
| Authentication | `Not authenticated. Run 'gh auth login' first.` |
| Rate limit | `Rate limit exceeded. Resets at HH:MM.` |
| Not found (repo) | `PR #N in owner/repo not found.` |
| Not found (PR) | `PR #N in owner/repo not found.` |

## Conditional Patterns

### Check if PR is merge-ready
```bash
if gh ghent summary --pr 42 --format json --no-tui > /dev/null 2>&1; then
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
output=$(gh ghent summary --pr 42 --format json --no-tui 2>&1)
exit_code=$?

if [ $exit_code -eq 2 ]; then
  echo "Error: $output" >&2
  exit 1
fi

# Safe to parse JSON
echo "$output" | jq '.is_merge_ready'
```
