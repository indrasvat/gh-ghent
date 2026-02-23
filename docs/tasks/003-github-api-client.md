# Task 1.4: GitHub API Client Wiring

## Status: DONE

## Depends On
- Task 1.1: Repository scaffold (needs go.mod with go-gh)
- Task 1.3: Domain types (needs port interfaces to implement)

## Parallelizable With
- None (depends on 1.1 + 1.3)

## Problem

ghent needs a GitHub API adapter that implements the domain port interfaces using go-gh's authenticated clients. This wires the GraphQL client (for review threads) and REST client (for check runs) to the domain layer with stub methods — actual queries come in Phase 2.

## PRD Reference

- §4 (Technology Stack) — go-gh v2.13.0
- §5.1 (Architecture) — github/ directory structure
- §5.4 (Key Design Decisions) — GraphQL for threads, REST for checks, errgroup for parallel

## Research References

- `docs/gh-extensions-support-research.md` §4 — go-gh library (DefaultRESTClient, DefaultGraphQLClient)
- `docs/gh-extensions-support-research.md` §6 — Authentication inheritance
- `docs/github-api-research.md` §10 — go-gh library specifics
- `docs/vivecaka-large-pr-patterns-research.md` §3 — errgroup pattern

## Files to Create

- `internal/github/client.go` — Client struct, constructor with functional options
- `internal/github/client_test.go` — Constructor tests, interface satisfaction

## Files to Modify

- `internal/cli/root.go` — Wire client in PersistentPreRunE

## Execution Steps

### Step 1: Read context
1. Read PRD §5.1, §5.4
2. Read `docs/gh-extensions-support-research.md` §4 (go-gh library), §6 (auth)

### Step 2: Create GitHub client
- Functional options: WithRESTClient, WithGraphQLClient (for testing)
- Default: api.DefaultRESTClient(), api.DefaultGraphQLClient()

### Step 3: Add stub methods satisfying all port interfaces

### Step 4: Wire client to Cobra commands

### Step 5: Unit tests
- Constructor with defaults and injected mocks
- Interface satisfaction compile-time checks

## Verification

### L1: Unit Tests
```bash
make test
```

### L3: Binary Execution
```bash
make build
./bin/gh-ghent comments --pr 1
# "not implemented" error — proves client is wired
```

## Completion Criteria

1. Client implements ThreadFetcher, CheckFetcher, ThreadResolver, ThreadReplier
2. Functional options allow mock injection
3. Client wired to Cobra via PersistentPreRunE
4. `make ci` passes
5. PROGRESS.md updated

## Commit

```
feat(github): wire go-gh API clients with port interface stubs

- Client struct wrapping GraphQL + REST from go-gh
- Functional options for test dependency injection
- Implements all domain port interfaces including ThreadReplier (stub methods)
- Wired to Cobra via PersistentPreRunE
```

## Session Protocol

1. Read CLAUDE.md
2. Read this task file
3. **Change this task's status to `IN PROGRESS`**
4. Read PRD §5.1, §5.4
5. Read `docs/gh-extensions-support-research.md` §5-6
6. Execute steps 1-5
7. Run verification (L1 → L3)
8. **Change this task's status to `DONE`**
9. Update `docs/PROGRESS.md`
10. Update CLAUDE.md Learnings if needed
11. Commit
