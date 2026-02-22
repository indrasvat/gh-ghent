# Task 6.3: Summary Mode Enhancements

## Status: TODO

## Depends On
- Phase 5 complete (or at minimum Phase 3 — CLI complete)

## Parallelizable With
- Task 6.1: --since filter (independent)
- Task 6.2: --group-by flag (independent)
- Task 6.4: Batch resolve (independent)

## Problem

The summary command can be enhanced with a one-line-per-thread digest mode for quick agent scanning, compact output mode, and additional metadata (PR age, review cycle count, time since last update).

## PRD Reference

- §8 Phase 6 — Task 6.3: Summary mode enhancements
- Original feature guide: `~/.agent/diagrams/ghent-research-report.html` §7 Feature #1 (expanded summary)

## Files to Modify

- `internal/cli/summary.go` — Add `--compact` flag
- `internal/formatter/json.go` — Compact summary format
- `internal/formatter/xml.go` — Compact summary format
- `internal/formatter/markdown.go` — One-line-per-thread digest

## Execution Steps

### Step 1: Add `--compact` flag
- Produces a one-line-per-thread/check digest
- Optimized for agents that need quick scanning

### Step 2: Add additional metadata
- PR age (time since creation)
- Review cycle count (number of review rounds)
- Time since last comment/push
- Lines changed (+/-) summary

### Step 3: Implement compact formatters
- JSON: flat array of `{ file, line, author, body_preview }` objects
- Markdown: table format with columns

### Step 4: Unit tests
- Compact mode produces shorter output
- Additional metadata fields present

## Verification

### L1 + L3
```bash
make test && make build
./bin/gh-ghent summary --pr 1 --compact --format json | jq '. | length'
```

## Completion Criteria

1. `--compact` flag produces digest output
2. Additional metadata: PR age, review cycles, last update
3. All formatters support compact mode
4. `make ci` passes

## Commit

```
feat(summary): add --compact mode and additional PR metadata

- One-line-per-thread compact digest mode
- PR age, review cycle count, time since last update
- Optimized for quick agent scanning
```
