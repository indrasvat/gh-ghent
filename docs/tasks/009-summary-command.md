# Task 2.6: Summary Command (`gh ghent summary`)

## Status: TODO

## Depends On
- Task 2.1: Comments command (needs ThreadFetcher)
- Task 2.2: Checks command (needs CheckFetcher)

## Parallelizable With
- None (needs both comments + checks working)

## Problem

ghent needs `gh ghent summary` to aggregate all PR data (threads, checks, approvals) into a single output. This is the most agent-useful command — one call gives full PR context. Exit code 0 means merge-ready. Pipe mode only in this phase; TUI dashboard comes in Phase 5.

## PRD Reference

- §6.6 (Summary Command) — flags, pipe mode, exit codes, acceptance criteria FR-SUM-01 through FR-SUM-05
- §5.4 (Key Design Decisions) — errgroup for parallel fetch

## Research References

- `docs/vivecaka-large-pr-patterns-research.md` §3 — errgroup pattern for parallel API calls
- `docs/github-api-research.md` §1, §6 — thread + check APIs (already implemented by 2.1/2.2)

## Files to Create

- `internal/github/reviews.go` — GraphQL reviews/approvals query
- `internal/github/reviews_test.go` — Reviews query tests
- `internal/formatter/json_test.go` — Summary format tests (extend existing)
- `testdata/graphql/pr_reviews.json` — Mock reviews/approvals response

## Files to Modify

- `internal/cli/summary.go` — Replace stub with real RunE implementation
- `internal/github/client.go` — Add approval fetching (reviews query)
- `internal/github/threads.go` — May need to expose resolved thread counts
- `internal/formatter/formatter.go` — Add FormatSummary to interface
- `internal/formatter/json.go` — Summary format
- `internal/formatter/xml.go` — Summary format
- `internal/formatter/markdown.go` — Summary format

## Execution Steps

### Step 1: Read context
1. Read PRD §6.6
2. Read `docs/vivecaka-large-pr-patterns-research.md` §3 (errgroup)

### Step 2: Implement parallel data fetch
- Use `errgroup` to fetch threads and checks concurrently
- Graceful degradation: if one fails, still output partial results
- Also fetch PR reviews/approvals via GraphQL

### Step 3: Implement merge readiness logic
- Merge-ready = no unresolved threads AND all checks pass AND sufficient approvals
- Exit code 0 = ready, 1 = not ready, 2 = error

### Step 4: Implement summary formatters
- JSON: single object with `threads`, `checks`, `approvals`, `merge_ready` sections
- XML: `<summary>` root with child sections
- Markdown: KPI line + sections with counts

### Step 5: Wire command
- `summary.go` RunE: parallel fetch → aggregate → merge readiness → format → stdout

### Step 6: Unit + integration tests
- **L1**: Parallel fetch with partial failure
- **L1**: Merge readiness logic (all combinations)
- **L1**: Formatter output validation
- **L2**: HTTP-mocked parallel fetch test (per `docs/testing-strategy.md` §3)
- **L2**: Thread + checks + reviews endpoints mocked
- **L2**: Partial failure (one endpoint 500) → graceful degradation
- **L2**: All-clear scenario → merge_ready = true

## Verification

### L1: Unit Tests
```bash
make test
```

### L3: Binary Execution (real repos)

```bash
make build

# NOT merge-ready: has unresolved threads + passing checks
./bin/gh-ghent summary -R indrasvat/tbgs --pr 1 --format json | jq .
# Expected: is_merge_ready=false (2 unresolved threads)

# NOT merge-ready: no threads but failing checks
./bin/gh-ghent summary -R indrasvat/visarga --pr 1 --format json | jq '{merge_ready: .is_merge_ready, unresolved: .comments.unresolved_count, checks_status: .checks.overall_status}'
# Expected: is_merge_ready=false (checks failing)

# Passing checks, resolved threads (doot — 1 resolved thread, checks pass)
./bin/gh-ghent summary -R indrasvat/doot --pr 1 --format json | jq '.is_merge_ready'
# Expected: true (0 unresolved, checks pass)

# Markdown summary
./bin/gh-ghent summary -R indrasvat/peek-it --pr 2 --format md
```

**Real repo test matrix:**

| Repo | PR | Threads | Checks | Expected Merge Ready |
|------|-----|---------|--------|---------------------|
| `indrasvat/tbgs` | #1 | 2 unresolved | pass | **false** (threads) |
| `indrasvat/visarga` | #1 | 0 | failure | **false** (checks) |
| `indrasvat/doot` | #1 | 1 resolved | pass | **true** |
| `indrasvat/peek-it` | #2 | 1 unresolved | failure | **false** (both) |

### L5: Agent Workflow
```bash
# Agent checks if PR is merge-ready via exit code
./bin/gh-ghent summary -R indrasvat/doot --pr 1 --format json; echo "exit: $?"    # 0 (ready)
./bin/gh-ghent summary -R indrasvat/tbgs --pr 1 --format json; echo "exit: $?"    # 1 (not ready)
```

## Completion Criteria

1. Parallel fetch of threads + checks + approvals via errgroup
2. Graceful degradation on partial API failure
3. Merge readiness logic correct (all three conditions)
4. Exit codes: 0 (ready), 1 (not ready), 2 (error)
5. All three formatters produce combined output
6. `make ci` passes
7. PROGRESS.md updated

## Commit

```
feat(summary): add summary command with parallel fetch and merge readiness

- errgroup parallel fetch: threads + checks + approvals
- Merge readiness: unresolved=0 AND checks=pass AND approvals met
- Graceful degradation on partial API failure
- Exit code 0 = merge-ready, 1 = not ready
```

## Session Protocol

1. Read CLAUDE.md
2. Read this task file
3. **Change this task's status to `IN PROGRESS`**
4. Read PRD §6.6
5. Read `docs/vivecaka-large-pr-patterns-research.md` §3
6. Execute steps 1-6
7. Run verification (L1 → L2 → L3 → L5) — per `docs/testing-strategy.md` §10 checklist
8. **Change this task's status to `DONE`**
9. Update `docs/PROGRESS.md`
10. Update CLAUDE.md Learnings if needed
11. Commit
