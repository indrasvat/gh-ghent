# Task 2.1: Comments Command (`gh ghent comments`)

## Status: DONE

## Depends On
- Phase 1 complete (all of tasks 1.1-1.4)

## Parallelizable With
- Task 2.2: Checks command (independent API + formatters)

## Problem

ghent needs its first real command: `gh ghent comments` fetches unresolved PR review threads via GraphQL and outputs them through the formatter layer (md/json/xml). This task delivers end-to-end pipe mode for comments — the TUI comes in Phase 5.

## PRD Reference

- §6.2 (Comments Command) — flags, pipe mode output, exit codes, acceptance criteria FR-COM-01 through FR-COM-11
- §6.8 (Output Formats) — JSON/XML/MD format guarantees
- §5.4 (Key Design Decisions) — GraphQL for threads (REST doesn't expose `isResolved`)

## Research References

- `docs/github-api-research.md` §1 — GraphQL query for review threads (full query with pagination)
- `docs/github-api-research.md` §5 — Key GraphQL types (ReviewThread, Comment fields, viewerCanResolve)
- `docs/vivecaka-large-pr-patterns-research.md` §2 — Pagination pattern (pageInfo.hasNextPage/endCursor)
- `docs/gh-extensions-support-research.md` §10 — JSON output patterns

## Files to Create

- `internal/github/threads.go` — GraphQL query implementation with pagination
- `internal/github/threads_test.go` — Unit tests with mock GraphQL responses
- `internal/formatter/formatter.go` — Formatter interface + factory (NewFormatter)
- `internal/formatter/json.go` — JSON formatter
- `internal/formatter/xml.go` — XML formatter
- `internal/formatter/markdown.go` — Markdown formatter
- `internal/formatter/json_test.go` — JSON output validation
- `internal/formatter/xml_test.go` — XML well-formedness
- `internal/formatter/markdown_test.go` — Markdown structure
- `testdata/graphql/review_threads.json` — Mock GraphQL response fixture
- `testdata/graphql/review_threads_page2.json` — Pagination fixture
- `.claude/automations/test_ghent_pipe.py` — iterm2-driver L4 visual test for pipe mode output (canonical template from `docs/testing-strategy.md` §5)

## Files to Modify

- `internal/cli/comments.go` — Replace stub with real RunE implementation
- `internal/github/client.go` — Implement ThreadFetcher interface methods

## Execution Steps

### Step 1: Read context
1. Read PRD §6.2, §6.8
2. Read `docs/github-api-research.md` §1, §5
3. Read `docs/vivecaka-large-pr-patterns-research.md` §2 (pagination)

### Step 2: Implement GraphQL thread fetcher
- Query with pagination: `reviewThreads(first: 100, after: $cursor)`
- Loop until `pageInfo.hasNextPage` is false
- Map GraphQL response to domain.ReviewThread types
- Client-side filter: only return threads where `isResolved == false`
- Capture `viewerCanResolve`, `viewerCanUnresolve`, `viewerCanReply` per thread

### Step 3: Implement formatter layer
- `Formatter` interface: `FormatComments(w io.Writer, result domain.CommentsResult) error`
- JSON: `json.NewEncoder` with indent
- XML: `encoding/xml` with root element `<comments>`
- Markdown: headers, code blocks, file:line references
- No ANSI codes in any formatter output

### Step 4: Wire comments command
- `comments.go` RunE: fetch threads → format → write to stdout
- Exit code: 0 if no unresolved, 1 if unresolved, 2 on error
- Respect `--format` flag (default: json when piped)

### Step 5: Unit + integration tests
- **L1**: GraphQL response parsing (happy path, empty, pagination)
- **L1**: Each formatter produces valid output
- **L1**: Exit codes correct
- **L1**: No ANSI in piped output
- **L2**: HTTP-mocked GraphQL test with `httptest` (per `docs/testing-strategy.md` §3)
- **L2**: Unresolved filter with mock response
- **L2**: Pagination with mock multi-page response
- **L2**: Rate limit response → graceful error

### Step 6: Create test fixtures
- `testdata/graphql/review_threads.json` — 3 threads, mix of resolved/unresolved
- `testdata/graphql/review_threads_page2.json` — second page

## Verification

### L1: Unit Tests
```bash
make test
```

### L3: Binary Execution
```bash
make build
./bin/gh-ghent comments --pr 1 --format json | jq .    # Valid JSON
./bin/gh-ghent comments --pr 1 --format xml             # Well-formed XML
./bin/gh-ghent comments --pr 1 --format md               # Readable markdown
./bin/gh-ghent comments --pr 1 | cat                     # No ANSI codes
```

### L4: Visual (iterm2-driver)
Create `.claude/automations/test_ghent_pipe.py` following canonical template in `docs/testing-strategy.md` §5:
```bash
uv run .claude/automations/test_ghent_pipe.py
```
- Verify: JSON output is valid (no ANSI codes, parseable by `python3 -m json.tool`)
- Verify: XML output is well-formed
- Verify: markdown output has file paths and comment bodies
- Screenshots: `ghent_pipe_json.png`, `ghent_pipe_md.png`

### L5: Agent Workflow
```bash
# Verify exit codes
./bin/gh-ghent comments --pr 1 --format json; echo "exit: $?"
# Verify jq parseable
./bin/gh-ghent comments --pr 1 --format json | jq '.threads[0].file'
```

## Completion Criteria

1. GraphQL query fetches all threads with pagination
2. Client-side filters to unresolved only
3. All three formatters produce correct output
4. No ANSI codes in pipe output
5. Exit codes: 0 (clean), 1 (unresolved), 2 (error)
6. `make ci` passes
7. PROGRESS.md updated

## Commit

```
feat(comments): add comments command with GraphQL fetch and formatters

- GraphQL review thread fetcher with pagination (pageInfo/endCursor)
- Client-side isResolved filtering
- JSON, XML, Markdown formatters with no ANSI bleed
- Exit codes: 0 (clean), 1 (unresolved), 2 (error)
```

## Visual Test Results

**L4 Script:** `.claude/automations/test_ghent_pipe.py`
**Date:** 2026-02-22
**Status:** PASS (7/7)

| Test | Result | Detail |
|------|--------|--------|
| Build binary | PASS | `make build` succeeded |
| JSON command ran | PASS | Exit code captured |
| JSON valid | PASS | `python3 -m json.tool` validates |
| JSON no ANSI | PASS | No escape sequences in piped output |
| Markdown command ran | PASS | Exit code captured |
| XML well-formed | PASS | `xml.etree.ElementTree.parse` validates |
| --pr required check | PASS | Error message correctly shown |

**Screenshots reviewed:**
- `ghent_pipe_json.png` — JSON output visible in terminal, valid structure
- `ghent_pipe_md.png` — Markdown output visible in terminal, headers and formatting correct

**Findings:** All pipe-mode formats produce clean output with no ANSI bleed. JSON is parseable by jq/python, XML is well-formed, --pr flag validation works correctly.

## Session Protocol

1. Read CLAUDE.md
2. Read this task file
3. **Change this task's status to `IN PROGRESS`**
4. Read PRD §6.2, §6.8
5. Read `docs/github-api-research.md` §1, §5
6. Execute steps 1-6
7. Run verification (L1 → L2 → L3 → L4 → L5) — per `docs/testing-strategy.md` §10 checklist
8. **Change this task's status to `DONE`**
9. Update `docs/PROGRESS.md`
10. Update CLAUDE.md Learnings if needed
11. Commit
