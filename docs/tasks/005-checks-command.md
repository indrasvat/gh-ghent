# Task 2.2: Checks Command (`gh ghent checks`)

## Status: TODO

## Depends On
- Phase 1 complete (all of tasks 1.1-1.4)

## Parallelizable With
- Task 2.1: Comments command (independent API + formatters)

## Problem

ghent needs `gh ghent checks` to fetch CI check runs and annotations via REST API, aggregate status (fail > pending > pass), and output through formatters. This task delivers pipe mode for checks — TUI and watch mode come later.

## PRD Reference

- §6.3 (Checks Command) — flags, pipe mode output, exit codes, acceptance criteria FR-CHK-01 through FR-CHK-08
- §6.8 (Output Formats) — format guarantees
- §5.4 (Key Design Decisions) — REST for check runs/logs (GraphQL doesn't expose job logs)

## Research References

- `docs/github-api-research.md` §6 — REST check runs and annotations endpoints
- `docs/github-api-research.md` §7 — Workflow runs and jobs
- `docs/vivecaka-large-pr-patterns-research.md` §13 — CI status aggregation (fail > pending > pass)
- `docs/gh-extensions-support-research.md` §5 — REST API access via go-gh

## Files to Create

- `internal/github/checks.go` — REST check runs + annotations fetch
- `internal/github/checks_test.go` — Unit tests with mock REST responses
- `testdata/rest/check_runs.json` — Mock check runs response
- `testdata/rest/annotations.json` — Mock annotations response

## Files to Modify

- `internal/cli/checks.go` — Replace stub with real RunE implementation
- `internal/github/client.go` — Implement CheckFetcher interface methods
- `internal/formatter/json.go` — Add FormatChecks method
- `internal/formatter/xml.go` — Add FormatChecks method
- `internal/formatter/markdown.go` — Add FormatChecks method
- `internal/formatter/formatter.go` — Add FormatChecks to interface

## Execution Steps

### Step 1: Read context
1. Read PRD §6.3, §6.8
2. Read `docs/github-api-research.md` §6-7
3. Read `docs/vivecaka-large-pr-patterns-research.md` §13 (status aggregation)

### Step 2: Implement REST check fetcher
- `GET /repos/{owner}/{repo}/commits/{ref}/check-runs` — fetch all check runs for HEAD SHA
- For each failed check: `GET /repos/{owner}/{repo}/check-runs/{id}/annotations`
- Map to domain.CheckRun and domain.Annotation types
- Aggregate overall status: fail > pending > pass

### Step 3: Get PR HEAD SHA
- Need to resolve PR number → head SHA via REST or GraphQL
- Use `GET /repos/{owner}/{repo}/pulls/{number}` → `.head.sha`

### Step 4: Add FormatChecks to formatter layer
- Extend all three formatters with checks output
- Include annotations inline with failing checks

### Step 5: Wire checks command
- `checks.go` RunE: get HEAD SHA → fetch checks → fetch annotations → format → stdout
- Exit codes: 0 (all pass), 1 (failures), 2 (error), 3 (pending)

### Step 6: Unit + integration tests
- **L1**: REST response parsing, status aggregation
- **L1**: Formatters for checks data
- **L1**: Exit code logic
- **L2**: HTTP-mocked REST test with `httptest` (per `docs/testing-strategy.md` §3)
- **L2**: Check runs endpoint mock with various statuses
- **L2**: Annotations endpoint mock for failing checks
- **L2**: Error responses (404, 403 rate limit) handled gracefully

## Verification

### L1: Unit Tests
```bash
make test
```

### L3: Binary Execution
```bash
make build
./bin/gh-ghent checks --pr 1 --format json | jq .
./bin/gh-ghent checks --pr 1 --format json | jq '.overall_status'
```

### L5: Agent Workflow
```bash
./bin/gh-ghent checks --pr 1 --format json; echo "exit: $?"
```

## Completion Criteria

1. REST API fetches check runs and annotations
2. Status aggregation: fail > pending > pass
3. All three formatters produce correct checks output
4. Annotations included for failing checks
5. Exit codes: 0/1/2/3 per PRD
6. `make ci` passes
7. PROGRESS.md updated

## Commit

```
feat(checks): add checks command with REST fetch and status aggregation

- REST check runs + annotations fetcher
- Status aggregation: fail > pending > pass
- JSON, XML, Markdown formatters for checks output
- Exit codes: 0 (pass), 1 (fail), 2 (error), 3 (pending)
```

## Session Protocol

1. Read CLAUDE.md
2. Read this task file
3. **Change this task's status to `IN PROGRESS`**
4. Read PRD §6.3, §6.8
5. Read `docs/github-api-research.md` §6-7
6. Execute steps 1-6
7. Run verification (L1 → L2 → L3 → L5) — per `docs/testing-strategy.md` §10 checklist
8. **Change this task's status to `DONE`**
9. Update `docs/PROGRESS.md`
10. Update CLAUDE.md Learnings if needed
11. Commit
