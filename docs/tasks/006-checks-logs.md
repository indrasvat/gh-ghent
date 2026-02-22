# Task 2.3: Checks Logs (`gh ghent checks --logs`)

## Status: TODO

## Depends On
- Task 2.2: Checks command (needs check runs fetch + formatter integration)

## Parallelizable With
- None (depends on 2.2)

## Problem

The `--logs` flag on `gh ghent checks` should include failing job log excerpts in pipe mode output. This requires fetching job logs via REST API, extracting relevant error lines, and integrating into the existing checks formatter output.

## PRD Reference

- §6.3 (Checks Command) — `--logs` flag, FR-CHK-06 (log excerpt in pipe mode)
- §5.1 (Architecture) — `internal/github/logs.go`

## Research References

- `docs/github-api-research.md` §7 — Workflow runs and jobs, log download
- `docs/github-api-research.md` §9 — gh CLI log viewing (known issues with ZIP format)

## Files to Create

- `internal/github/logs.go` — REST job log fetcher + error line extraction
- `internal/github/logs_test.go` — Log parsing tests
- `testdata/rest/job_log.txt` — Mock job log output

## Files to Modify

- `internal/github/checks.go` — Integrate log fetch when `--logs` is set
- `internal/cli/checks.go` — Pass `--logs` flag through to fetcher
- `internal/formatter/json.go` — Include `log_excerpt` field
- `internal/formatter/xml.go` — Include `<log_excerpt>` element
- `internal/formatter/markdown.go` — Include log excerpt code block

## Execution Steps

### Step 1: Read context
1. Read PRD §6.3 (checks --logs behavior)
2. Read `docs/github-api-research.md` §7, §9

### Step 2: Implement log fetcher
- For each failed check run, resolve the workflow job ID
- `GET /repos/{owner}/{repo}/actions/jobs/{job_id}/logs` — returns plain text
- Extract error-relevant lines (compile errors, test failures, lint warnings)
- Parse file:line references from log output

### Step 3: Integrate with checks output
- Add `LogExcerpt` field to domain.CheckRun
- Only populate when `--logs` flag is set (avoid unnecessary API calls)
- Truncate excerpts to reasonable length (last 50 lines of failing step)

### Step 4: Update formatters
- JSON: `"log_excerpt": "..."` field on failing checks
- XML: `<log_excerpt>` child element
- Markdown: fenced code block after annotations

### Step 5: Unit + integration tests
- **L1**: Log parsing extracts error lines
- **L1**: Formatters include log excerpt when present
- **L1**: Log excerpt omitted when `--logs` not set
- **L2**: HTTP-mocked REST test for job log endpoint (per `docs/testing-strategy.md` §3)
- **L2**: Log with lint errors → extracted correctly
- **L2**: Log with test failures → extracted correctly

## Verification

### L1: Unit Tests
```bash
make test
```

### L3: Binary Execution
```bash
make build
./bin/gh-ghent checks --pr 1 --logs --format json | jq '.checks[0].log_excerpt'
./bin/gh-ghent checks --pr 1 --format json | jq '.checks[0].log_excerpt'  # should be null/absent
```

## Completion Criteria

1. Job logs fetched for failing checks only
2. Error lines extracted with file:line references
3. Formatters include log excerpt when `--logs` set
4. No extra API calls when `--logs` not set
5. `make ci` passes
6. PROGRESS.md updated

## Commit

```
feat(checks): add --logs flag for failing job log excerpts

- REST job log fetcher with error line extraction
- Log excerpts included in JSON/XML/MD when --logs set
- No extra API calls without the flag
```

## Session Protocol

1. Read CLAUDE.md
2. Read this task file
3. **Change this task's status to `IN PROGRESS`**
4. Read PRD §6.3
5. Read `docs/github-api-research.md` §7, §9
6. Execute steps 1-5
7. Run verification (L1 → L2 → L3) — per `docs/testing-strategy.md` §10 checklist
8. **Change this task's status to `DONE`**
9. Update `docs/PROGRESS.md`
10. Update CLAUDE.md Learnings if needed
11. Commit
