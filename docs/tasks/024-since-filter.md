# Task 6.1: `--since` Filter

## Status: DONE

## Depends On
- Phase 5 complete (or at minimum Phase 3 — CLI complete)

## Parallelizable With
- Task 6.2: --group-by flag (independent)
- Task 6.3: Summary enhancements (independent)
- Task 6.4: Batch resolve (independent)

## Problem

AI agents running in loops need to see only what changed since their last check. The `--since` flag filters comments and checks by timestamp, reducing noise in long-running review cycles.

## PRD Reference

- §8 Phase 6 — Task 6.1: `--since` flag
- Original feature guide: `~/.agent/diagrams/ghent-research-report.html` §7 Feature #4

## Files to Modify

- `internal/cli/flags.go` — Add `--since` to GlobalFlags
- `internal/cli/comments.go` — Pass since filter to fetcher
- `internal/cli/checks.go` — Pass since filter to fetcher
- `internal/cli/summary.go` — Pass since filter
- `internal/github/threads.go` — Filter threads by `createdAt` >= since
- `internal/github/checks.go` — Filter check runs by `completed_at` >= since

## Execution Steps

### Step 1: Add `--since` flag
- Accepts: ISO 8601 timestamp or relative duration (e.g., `1h`, `30m`, `2d`)
- Parse to `time.Time`
- Persistent flag on root (available to all commands)

### Step 2: Implement client-side filtering
- Comments: filter threads where newest comment `createdAt` >= since
- Checks: filter check runs where `completed_at` >= since (or `started_at` for running)
- Summary: apply to both

### Step 3: Update formatters
- Include `since` filter metadata in output (so agent knows the filter applied)

### Step 4: Unit tests
- Timestamp parsing (ISO 8601 + relative durations)
- Filtering logic for comments and checks
- Edge case: no results after filter

## Verification

### L1 + L3
```bash
make test && make build
./bin/gh-ghent comments --pr 1 --since 2026-02-22T00:00:00Z --format json
./bin/gh-ghent comments --pr 1 --since 1h --format json
```

## Completion Criteria

1. `--since` accepts ISO 8601 and relative durations
2. Filters applied to comments and checks
3. Output includes filter metadata
4. `make ci` passes

## Commit

```
feat(cli): add --since flag to filter by timestamp

- ISO 8601 and relative duration parsing (1h, 30m, 2d)
- Client-side filtering for comments and checks
- Filter metadata included in formatter output
```
