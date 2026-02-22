# Task 2.4: Resolve Command (`gh ghent resolve`)

## Status: TODO

## Depends On
- Task 2.1: Comments command (needs GraphQL thread fetch + thread IDs)

## Parallelizable With
- Task 2.3: Checks logs (independent)

## Problem

ghent needs `gh ghent resolve` to resolve (or unresolve) review threads via GraphQL mutations. In pipe mode, this works with `--thread <id>` for a single thread or `--all` for bulk resolution. The TUI multi-select interface comes in Phase 5.

## PRD Reference

- §6.4 (Resolve Command) — flags, pipe mode, exit codes, acceptance criteria FR-RES-01 through FR-RES-08
- §6.4 Permission check — `viewerCanResolve`/`viewerCanUnresolve` booleans

## Research References

- `docs/github-api-research.md` §2 — GraphQL `resolveReviewThread` mutation
- `docs/github-api-research.md` §3 — GraphQL `unresolveReviewThread` mutation
- `docs/github-api-research.md` §5 — Key types (thread node ID format: `PRRT_`)

## Files to Create

- `internal/github/resolve.go` — GraphQL resolve/unresolve mutations
- `internal/github/resolve_test.go` — Mutation tests with mock responses
- `testdata/graphql/resolve_thread.json` — Mock mutation response
- `testdata/graphql/unresolve_thread.json` — Mock unresolve response

## Files to Modify

- `internal/cli/resolve.go` — Replace stub with real RunE implementation
- `internal/github/client.go` — Implement ThreadResolver interface methods
- `internal/formatter/formatter.go` — Add FormatResolveResult to interface
- `internal/formatter/json.go` — Add resolve result formatting
- `internal/formatter/xml.go` — Add resolve result formatting
- `internal/formatter/markdown.go` — Add resolve result formatting

## Execution Steps

### Step 1: Read context
1. Read PRD §6.4
2. Read `docs/github-api-research.md` §2-3

### Step 2: Implement resolve/unresolve mutations
- `resolveReviewThread(input: { threadId: $id })` → returns `thread { id, isResolved }`
- `unresolveReviewThread(input: { threadId: $id })` → returns `thread { id, isResolved }`
- Validate `viewerCanResolve` / `viewerCanUnresolve` before mutation

### Step 3: Implement pipe mode logic
- `--thread <id>`: resolve single thread, output result
- `--all`: fetch all unresolved threads (reuse ThreadFetcher), resolve each, output results
- `--unresolve`: flip mutation to unresolve
- Mutually exclusive: `--thread` and `--all`

### Step 4: Permission checking
- Before resolving, verify `viewerCanResolve` is true for the thread
- Surface clear error message if permission denied

### Step 5: Wire command + formatters
- Exit codes: 0 (all resolved), 1 (partial failure), 2 (error/auth)
- Output: per-thread result with file:line and status

### Step 6: Unit + integration tests
- **L1**: Resolve mutation success/failure
- **L1**: Permission check enforcement
- **L1**: `--all` resolves multiple threads
- **L1**: `--unresolve` uses correct mutation
- **L2**: HTTP-mocked GraphQL test for resolve mutation (per `docs/testing-strategy.md` §3)
- **L2**: Success and error mutation responses
- **L2**: Multiple thread resolution sequence

## Verification

### L1: Unit Tests
```bash
make test
```

### L3: Binary Execution
```bash
make build
./bin/gh-ghent resolve --pr 1 --thread PRRT_test123 --format json
./bin/gh-ghent resolve --pr 1 --all --format json
./bin/gh-ghent resolve --pr 1 --thread PRRT_test123 --unresolve
```

## Completion Criteria

1. Resolve mutation works for single thread
2. `--all` resolves all unresolved threads
3. `--unresolve` uses unresolve mutation
4. Permission check before mutation
5. Clear error for missing permissions
6. Exit codes: 0/1/2 per PRD
7. `make ci` passes
8. PROGRESS.md updated

## Commit

```
feat(resolve): add resolve command with GraphQL mutations

- resolveReviewThread and unresolveReviewThread mutations
- --thread (single) and --all (bulk) pipe mode
- --unresolve flag for reversal
- viewerCanResolve permission check before mutation
```

## Session Protocol

1. Read CLAUDE.md
2. Read this task file
3. **Change this task's status to `IN PROGRESS`**
4. Read PRD §6.4
5. Read `docs/github-api-research.md` §2-3
6. Execute steps 1-6
7. Run verification (L1 → L2 → L3) — per `docs/testing-strategy.md` §10 checklist
8. **Change this task's status to `DONE`**
9. Update `docs/PROGRESS.md`
10. Update CLAUDE.md Learnings if needed
11. Commit
