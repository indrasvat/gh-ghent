# Task 3.1: Watch Mode — Pipe (`gh ghent checks --watch`)

## Status: TODO

## Depends On
- Phase 2 complete (needs checks command working end-to-end)

## Parallelizable With
- Task 3.2: Error handling hardening (independent)

## Problem

`gh ghent checks --watch` needs to poll CI checks until all complete, with fail-fast behavior. In pipe mode, this outputs one status object per poll cycle (no cursor movement, no ANSI). The TUI watch mode (spinner, progress bar) comes in Phase 5.

## PRD Reference

- §6.7 (Watch Mode) — poll behavior, fail-fast, pipe mode output, acceptance criteria FR-WAT-01 through FR-WAT-06
- §7.1 (Performance) — watch mode poll overhead < 1s

## Research References

- `docs/vivecaka-large-pr-patterns-research.md` §11 — Auto-refresh pattern
- `docs/github-api-research.md` §6 — Check runs endpoint (for polling)

## Files to Create

- `internal/github/watcher.go` — Poll loop with fail-fast logic
- `internal/github/watcher_test.go` — Watcher unit tests (mock time, mock API)

## Files to Modify

- `internal/cli/checks.go` — Wire `--watch` flag to watcher
- `internal/formatter/json.go` — Add FormatWatchStatus method (per-poll output)
- `internal/formatter/xml.go` — Add FormatWatchStatus method
- `internal/formatter/markdown.go` — Add FormatWatchStatus method
- `internal/formatter/formatter.go` — Add FormatWatchStatus to interface

## Execution Steps

### Step 1: Read context
1. Read PRD §6.7
2. Read `docs/vivecaka-large-pr-patterns-research.md` §11

### Step 2: Implement poll loop
- Poll every 10s (configurable via internal constant)
- On each poll: fetch check runs, compute aggregate status
- Fail-fast: if any check has `conclusion == "failure"` → stop, exit 1
- All pass: stop, exit 0
- All pending: continue polling
- Respect context cancellation (Ctrl+C → clean exit)

### Step 3: Implement pipe mode output
- One JSON/XML/MD status object per poll cycle
- Each object: timestamp, completed/total, overall_status, newly_completed checks
- Newline-delimited (NDJSON for JSON mode)
- No cursor movement, no ANSI codes

### Step 4: Wire to checks command
- `--watch` flag enables watcher
- Without `--watch` + pending checks → exit 3 (report pending, don't wait)

### Step 5: Unit tests
- Poll loop terminates on all-pass
- Fail-fast triggers on first failure
- Context cancellation (Ctrl+C) exits cleanly
- Pipe output is one-per-line, no ANSI

## Verification

### L1: Unit Tests
```bash
make test
```

### L3: Binary Execution (real repos)
```bash
make build

# Already-complete checks — should return immediately
./bin/gh-ghent checks -R indrasvat/doot --pr 1 --watch --format json
# Expected: one status line, exit 0 (all pass, already complete)

# Already-failed checks — should fail-fast immediately
./bin/gh-ghent checks -R indrasvat/peek-it --pr 2 --watch --format json
# Expected: one status line, exit 1 (failure detected)

# Ctrl+C test
./bin/gh-ghent checks -R indrasvat/visarga --pr 1 --watch --format json
# Ctrl+C should exit cleanly
```

**Real repo test matrix:**

| Repo | PR | Watch Behavior | Expected Exit |
|------|-----|---------------|---------------|
| `indrasvat/doot` | #1 | Instant complete (all pass) | 0 |
| `indrasvat/peek-it` | #2 | Instant fail-fast (failures) | 1 |
| `indrasvat/visarga` | #1 | Instant fail-fast (format fail) | 1 |

### L5: Agent Workflow
```bash
# Agent waits for CI, acts on result
./bin/gh-ghent checks -R indrasvat/doot --pr 1 --watch --format json
echo "CI result: $?"
```

## Completion Criteria

1. Poll loop works with 10s interval
2. Fail-fast on first check failure
3. Clean exit on all-pass or Ctrl+C
4. Pipe output: one status per line, no ANSI
5. Exit codes: 0 (pass), 1 (fail), 2 (error), 3 (pending without --watch)
6. `make ci` passes
7. PROGRESS.md updated

## Commit

```
feat(watch): add --watch flag with fail-fast polling for checks

- Poll loop with 10s interval and fail-fast on first failure
- Pipe mode: NDJSON one-status-per-line output
- Clean Ctrl+C handling via context cancellation
- Exit codes: 0 (all pass), 1 (failure detected)
```

## Session Protocol

1. Read CLAUDE.md
2. Read this task file
3. **Change this task's status to `IN PROGRESS`**
4. Read PRD §6.7
5. Read `docs/vivecaka-large-pr-patterns-research.md` §11
6. Execute steps 1-5
7. Run verification (L1 → L3 → L5)
8. **Change this task's status to `DONE`**
9. Update `docs/PROGRESS.md`
10. Update CLAUDE.md Learnings if needed
11. Commit
