# Task 6.4: Batch Resolve

## Status: DONE

## Depends On
- Phase 5 complete (or at minimum Phase 3 — CLI complete)

## Parallelizable With
- Task 6.1: --since filter (independent)
- Task 6.2: --group-by flag (independent)
- Task 6.3: Summary enhancements (independent)

## Problem

Agents often address all comments in a file at once. Batch resolve allows resolving threads by file pattern or author, rather than listing individual thread IDs. This reduces the number of commands an agent needs to run.

## PRD Reference

- §8 Phase 6 — Task 6.4: Batch resolve by file pattern or author
- Original feature guide: `~/.agent/diagrams/ghent-research-report.html` §7 (implied by resolve + group-by)

## Files to Modify

- `internal/cli/resolve.go` — Add `--file` and `--author` filter flags
- `internal/github/resolve.go` — Batch resolve with filtering

## Execution Steps

### Step 1: Add filter flags to resolve
- `--file <glob>` — resolve all threads in files matching glob (e.g., `internal/api/*.go`)
- `--author <login>` — resolve all threads started by a specific author
- Combinable: `--file` + `--author` = intersection

### Step 2: Implement batch logic
- Fetch all unresolved threads (reuse ThreadFetcher)
- Apply file/author filters
- Resolve each matching thread
- Report: N resolved, M skipped (no permission), K failed

### Step 3: Safety check
- Show count of threads to be resolved before executing
- In pipe mode: proceed immediately (agent use case)
- `--dry-run` flag: show what would be resolved without executing

### Step 4: Unit tests
- File glob matching
- Author filtering
- Combined filters
- Dry-run mode

## Verification

### L1 + L3
```bash
make test && make build
./bin/gh-ghent resolve --pr 1 --file "internal/api/*.go" --dry-run --format json
./bin/gh-ghent resolve --pr 1 --author reviewer1 --format json
```

## Completion Criteria

1. `--file <glob>` filters threads by file path
2. `--author <login>` filters by comment author
3. `--dry-run` shows what would be resolved
4. Batch execution with per-thread results
5. `make ci` passes

## Commit

```
feat(resolve): add batch resolve with --file and --author filters

- --file glob filter for path-based batch resolution
- --author filter for author-based batch resolution
- --dry-run mode for safe preview
- Per-thread result reporting
```
