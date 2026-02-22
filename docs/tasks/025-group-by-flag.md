# Task 6.2: `--group-by` Flag

## Status: TODO

## Depends On
- Phase 5 complete (or at minimum Phase 3 — CLI complete)

## Parallelizable With
- Task 6.1: --since filter (independent)
- Task 6.3: Summary enhancements (independent)
- Task 6.4: Batch resolve (independent)

## Problem

Agents fixing review comments work file-by-file. The `--group-by file` flag groups comments by file path so agents can batch-fix all comments in a single file before moving to the next. Also supports grouping by author or status.

## PRD Reference

- §8 Phase 6 — Task 6.2: `--group-by` flag
- Original feature guide: `~/.agent/diagrams/ghent-research-report.html` §7 Feature #6

## Files to Modify

- `internal/cli/comments.go` — Add `--group-by` flag (file, author, status)
- `internal/formatter/json.go` — Grouped JSON output structure
- `internal/formatter/xml.go` — Grouped XML output structure
- `internal/formatter/markdown.go` — Grouped markdown with headers

## Execution Steps

### Step 1: Add `--group-by` flag
- Values: `file` (default), `author`, `status` (resolved/unresolved)
- Only applies to comments command

### Step 2: Implement grouping logic
- Group threads by the selected dimension
- Sort groups: file (alphabetical), author (alphabetical), status (unresolved first)

### Step 3: Update formatters
- JSON: `{ "groups": [ { "key": "path/to/file.go", "threads": [...] } ] }`
- XML: `<groups><group key="..."><threads>...</threads></group></groups>`
- Markdown: `## path/to/file.go` headers with threads underneath

### Step 4: Unit tests
- Each grouping mode produces correct structure
- Edge cases: empty groups, single file, single author

## Verification

### L1 + L3
```bash
make test && make build
./bin/gh-ghent comments --pr 1 --group-by file --format json | jq '.groups[0].key'
./bin/gh-ghent comments --pr 1 --group-by author --format json
```

## Completion Criteria

1. `--group-by` with file/author/status options
2. All three formatters support grouped output
3. Default grouping is by file
4. `make ci` passes

## Commit

```
feat(comments): add --group-by flag for file/author/status grouping

- Group review threads by file path, author, or resolution status
- All formatters support grouped output structure
- Helps agents batch-fix comments per file
```
