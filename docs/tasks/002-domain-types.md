# Task 1.3: Domain Types and Port Interfaces

## Status: TODO

## Depends On
- Task 1.1: Repository scaffold (needs go.mod)

## Parallelizable With
- Task 1.2: Cobra CLI skeleton (no dependency between them)

## Problem

ghent needs domain types (ReviewThread, CheckRun, Annotation, etc.) and port interfaces (ThreadFetcher, CheckFetcher, ThreadResolver, Formatter) before any adapter, TUI, or command code can be written. These define the contract between all layers.

## PRD Reference

- §5.1 (Architecture) — domain/ directory with types.go and ports.go
- §5.4 (Key Design Decisions) — interface-based ports for testability
- §6.2 (Comments) — ReviewThread and Comment field requirements, output structure
- §6.3 (Checks) — CheckRun and Annotation field requirements, output structure
- §6.5 (Summary) — combined data requirements

## Research References

- `docs/github-api-research.md` §1 — GraphQL reviewThreads response shape
- `docs/github-api-research.md` §5 — Key GraphQL types (ReviewThread, Comment fields)
- `docs/github-api-research.md` §6 — REST check runs response shape
- `docs/vivecaka-large-pr-patterns-research.md` §13 — CI status aggregation (fail > pending > pass)

## Files to Create

- `internal/domain/types.go` — All domain types
- `internal/domain/ports.go` — Port interfaces
- `internal/domain/types_test.go` — Tests for status aggregation + JSON serialization

## Execution Steps

### Step 1: Read context
1. Read PRD §6.2, §6.3 (output structures)
2. Read `docs/github-api-research.md` §1, §5, §6

### Step 2: Define review thread types
- ReviewThread, Comment, CommentsResult

### Step 3: Define check run types
- CheckRun, Annotation, OverallStatus, ChecksResult
- AggregateStatus: fail > pending > pass

### Step 4: Define summary type
- SummaryResult combining threads + checks + approvals

### Step 5: Define port interfaces
- ThreadFetcher, CheckFetcher, ThreadResolver, Formatter

### Step 6: Unit tests
- AggregateStatus logic, JSON serialization, zero value safety

## Verification

### L1: Unit Tests
```bash
make test
```

## Completion Criteria

1. All domain types defined with correct JSON tags
2. Port interfaces defined (ThreadFetcher, CheckFetcher, ThreadResolver, Formatter)
3. AggregateStatus tested: fail > pending > pass
4. `make ci` passes
5. PROGRESS.md updated

## Commit

```
feat(domain): add review thread, check run types and port interfaces

- ReviewThread, Comment, CheckRun, Annotation domain types
- CommentsResult, ChecksResult, SummaryResult wrapper types
- Port interfaces for all adapters
- AggregateStatus with fail > pending > pass precedence
```

## Session Protocol

1. Read CLAUDE.md
2. Read this task file
3. **Change this task's status to `IN PROGRESS`**
4. Read PRD §6.2, §6.3, §6.5
5. Read `docs/github-api-research.md` §1, §5, §6
6. Execute steps 1-6
7. Run verification (L1)
8. **Change this task's status to `DONE`**
9. Update `docs/PROGRESS.md`
10. Update CLAUDE.md Learnings if needed
11. Commit
