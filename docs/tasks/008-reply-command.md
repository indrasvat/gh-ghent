# Task 2.5: Reply Command (`gh ghent reply`)

## Status: TODO

## Depends On
- Task 2.1: Comments command (needs GraphQL thread fetch to validate thread + get comment IDs)

## Parallelizable With
- Task 2.4: Resolve command (independent mutations)

## Problem

ghent needs `gh ghent reply` so AI agents can reply to review threads directly from the CLI. This is a pipe-first command (no TUI view) that posts a reply to a specific thread using the REST API. Agents use this to acknowledge feedback, explain fixes, or ask clarifying questions.

## PRD Reference

- §6.5 (Reply Command) — flags, behavior, output, exit codes, acceptance criteria FR-REP-01 through FR-REP-07
- §6.5 API note — REST endpoint `POST .../comments/{comment_id}/replies`

## Research References

- `docs/github-api-research.md` §8 — REST review comments, reply endpoint
- `docs/github-api-research.md` §5 — Key types (`viewerCanReply` field)
- `docs/github-api-research.md` §1 — GraphQL thread fetch (needed to get last comment's databaseId)

## Files to Create

- `internal/github/reply.go` — REST reply implementation
- `internal/github/reply_test.go` — Reply tests with mock REST responses
- `testdata/rest/reply_comment.json` — Mock reply response

## Files to Modify

- `internal/cli/reply.go` — Replace stub with real RunE implementation
- `internal/github/client.go` — Implement ThreadReplier interface methods
- `internal/formatter/formatter.go` — Add FormatReplyResult to interface
- `internal/formatter/json.go` — Add reply result formatting
- `internal/formatter/xml.go` — Add reply result formatting
- `internal/formatter/markdown.go` — Add reply result formatting

## Execution Steps

### Step 1: Read context
1. Read PRD §6.5
2. Read `docs/github-api-research.md` §8 (REST reply endpoint)

### Step 2: Implement reply logic
- Fetch thread via GraphQL to validate it exists
- Check `viewerCanReply` boolean — error if false
- Get `databaseId` of last comment in thread (reply target)
- `POST /repos/{owner}/{repo}/pulls/{pull_number}/comments/{comment_id}/replies`
- Request body: `{ "body": "<reply text>" }`

### Step 3: Implement body input
- `--body <text>`: inline body text
- `--body-file <path>`: read from file; `-` reads from stdin
- Mutually exclusive validation: error if both set

### Step 4: Wire command + formatters
- Output: thread_id, comment_id, url, body, created_at
- Exit codes: 0 (success), 1 (thread not found / can't reply), 2 (error)

### Step 5: Unit + integration tests
- **L1**: Reply posts to correct endpoint
- **L1**: `--body-file -` reads from stdin
- **L1**: `viewerCanReply` check enforced
- **L1**: Mutually exclusive `--body` / `--body-file` validation
- **L2**: HTTP-mocked REST test for reply endpoint (per `docs/testing-strategy.md` §3)
- **L2**: GraphQL thread validation mock (exists + viewerCanReply)
- **L2**: Error responses (thread not found, no permission)

## Verification

### L1: Unit Tests
```bash
make test
```

### L3: Binary Execution
```bash
make build
./bin/gh-ghent reply --pr 1 --thread PRRT_test123 --body "Fixed" --format json
echo "Fixed in abc123" | ./bin/gh-ghent reply --pr 1 --thread PRRT_test123 --body-file -
```

### L5: Agent Workflow
```bash
# Typical agent workflow: read comments, fix code, reply
./bin/gh-ghent comments --pr 1 --format json | jq '.threads[0].thread_id'
./bin/gh-ghent reply --pr 1 --thread PRRT_abc --body "Addressed in commit xyz"
```

## Completion Criteria

1. REST reply posts to correct thread
2. Thread validation via GraphQL (exists + viewerCanReply)
3. `--body` and `--body-file` both work (stdin via `-`)
4. Mutually exclusive validation
5. Output includes comment URL
6. Exit codes: 0/1/2 per PRD
7. `make ci` passes
8. PROGRESS.md updated

## Commit

```
feat(reply): add reply command for agent thread responses

- REST reply via POST .../comments/{id}/replies
- Thread validation with viewerCanReply check
- --body and --body-file (stdin via -) input modes
- JSON/XML/MD output with comment URL
```

## Session Protocol

1. Read CLAUDE.md
2. Read this task file
3. **Change this task's status to `IN PROGRESS`**
4. Read PRD §6.5
5. Read `docs/github-api-research.md` §8
6. Execute steps 1-5
7. Run verification (L1 → L2 → L3 → L5) — per `docs/testing-strategy.md` §10 checklist
8. **Change this task's status to `DONE`**
9. Update `docs/PROGRESS.md`
10. Update CLAUDE.md Learnings if needed
11. Commit
