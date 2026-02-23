# Task 3.2: Error Handling Hardening

## Status: TODO

## Depends On
- Phase 2 complete (needs all commands working to add error handling)

## Parallelizable With
- Task 3.1: Watch mode pipe (independent)

## Problem

All commands need robust error handling: rate limit awareness, auth errors, network timeouts, graceful degradation on partial API failures. This task hardens the entire CLI for production use.

## PRD Reference

- §7.2 (Reliability) — graceful degradation, rate limits, timeouts, retry
- §7.4 (Security) — no credential storage, GH_HOST respect

## Research References

- `docs/github-api-research.md` §11 — Rate limiting (5000 REST/hour, 5000 GraphQL points/hour)
- `docs/gh-extensions-support-research.md` §6 — Auth inheritance, error states

## Files to Create

- `internal/github/errors.go` — Custom error types (RateLimitError, AuthError, NotFoundError)
- `internal/github/errors_test.go` — Error type tests
- `internal/github/ratelimit.go` — Rate limit checking from response headers
- `internal/github/ratelimit_test.go` — Rate limit tests
- `.claude/automations/test_ghent_errors.py` — iterm2-driver L4 visual test for error handling (per `docs/testing-strategy.md` §8)

## Files to Modify

- `internal/github/client.go` — Add timeout configuration, retry logic, rate limit checking
- `internal/github/threads.go` — Wrap errors with context
- `internal/github/checks.go` — Wrap errors with context
- `internal/github/resolve.go` — Wrap errors with context
- `internal/github/reply.go` — Wrap errors with context
- `internal/github/logs.go` — Wrap errors with context
- `internal/cli/root.go` — User-friendly error messages in PersistentPostRunE

## Execution Steps

### Step 1: Read context
1. Read PRD §7.2, §7.4
2. Read `docs/github-api-research.md` §11

### Step 2: Define custom error types
- `RateLimitError` — includes reset time, remaining calls
- `AuthError` — token missing, expired, insufficient scope
- `NotFoundError` — repo, PR, thread not found
- All implement `error` interface with user-friendly messages

### Step 3: Add rate limit awareness
- Check `X-RateLimit-Remaining` header on every response
- Warn (to stderr) when remaining < 100
- Return `RateLimitError` when remaining == 0

### Step 4: Add timeout + retry
- 30s timeout per API call (configurable via functional option)
- 1 retry on 5xx errors with 1s backoff
- No retry on 4xx errors

### Step 5: Add graceful degradation
- If annotations fetch fails, still return check runs (checks command)
- If one parallel fetch fails in summary, return partial results
- Always indicate what failed in output

### Step 6: Wire user-friendly messages
- Auth: "Not authenticated. Run `gh auth login` first."
- Rate limit: "Rate limit exceeded. Resets at <time>."
- Not found: "PR #42 not found in owner/repo."

### Step 7: Unit tests
- Each error type produces correct message
- Rate limit detection from headers
- Retry on 5xx, no retry on 4xx
- Graceful degradation (partial results)

## Verification

### L1: Unit Tests
```bash
make test
```

### L3: Binary Execution (real repos + error cases)
```bash
make build

# Invalid repo → NotFoundError
./bin/gh-ghent comments --pr 1 -R nonexistent/repo; echo "exit: $?"
# Expected: user-friendly "not found" message, exit 2

# Invalid PR number → NotFoundError
./bin/gh-ghent comments --pr 99999 -R indrasvat/tbgs; echo "exit: $?"
# Expected: "PR #99999 not found", exit 2

# Missing --pr flag
./bin/gh-ghent comments -R indrasvat/tbgs; echo "exit: $?"
# Expected: "--pr flag is required", exit 2

# Valid request (should still work after hardening)
./bin/gh-ghent comments -R indrasvat/tbgs --pr 1 --format json | python3 -m json.tool > /dev/null
echo "exit: $?"
# Expected: valid JSON, exit 1 (has unresolved)

# Rate limit check (inspect stderr for warnings)
./bin/gh-ghent comments -R indrasvat/tbgs --pr 1 --format json 2>/tmp/ghent_stderr.txt
cat /tmp/ghent_stderr.txt  # Should be empty unless rate limit is low
```

### L4: Visual (iterm2-driver)
Create `.claude/automations/test_ghent_errors.py` following canonical template in `docs/testing-strategy.md` §5:
```bash
uv run .claude/automations/test_ghent_errors.py
```
- Verify: nonexistent repo → "PR #N not found in owner/repo" (not a stack trace)
- Verify: no auth → "Not authenticated. Run `gh auth login` first."
- Verify: exit code 2 for all error cases
- Screenshots: `ghent_error_notfound.png`, `ghent_error_auth.png`

## Completion Criteria

1. Custom error types with user-friendly messages
2. Rate limit awareness (warn at < 100, error at 0)
3. 30s timeout per API call
4. Retry: 1x on 5xx with 1s backoff, no retry on 4xx
5. Graceful degradation on partial failures
6. All existing tests still pass
7. `make ci` passes
8. PROGRESS.md updated

## Commit

```
feat(errors): add error handling with rate limits, retries, and graceful degradation

- Custom error types: RateLimitError, AuthError, NotFoundError
- Rate limit awareness from X-RateLimit-Remaining header
- 30s timeout, 1 retry on 5xx with 1s backoff
- Graceful degradation on partial API failures
```

## Session Protocol

1. Read CLAUDE.md
2. Read this task file
3. **Change this task's status to `IN PROGRESS`**
4. Read PRD §7.2, §7.4
5. Read `docs/github-api-research.md` §11
6. Execute steps 1-7
7. Run verification (L1 → L3 → L4)
8. **Change this task's status to `DONE`**
9. Update `docs/PROGRESS.md`
10. Update CLAUDE.md Learnings if needed
11. Commit
