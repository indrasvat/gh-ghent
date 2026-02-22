# Task 3.5: Debug Logging & Tracing

## Status: TODO

## Depends On
- Task 1.4: GitHub API client wiring (needs client.go to add Log field)
- Task 1.2: Cobra CLI skeleton (needs root command for --debug flag)

## Parallelizable With
- Task 3.1: Watch mode pipe (independent)
- Task 3.2: Error handling hardening (independent; both modify client.go but different fields)

## Problem

ghent has no debug logging infrastructure. When something goes wrong — slow API calls, unexpected pagination, auth issues, rate limits approaching — there's no way to see what's happening internally. The `gh` extension ecosystem uses `GH_DEBUG` to enable API transport logging, and popular extensions (gh-dash, gh-poi, gh-workflow-stats) add `--debug` flags for app-level tracing. ghent needs both layers: go-gh API transport logging via `GH_DEBUG`, and structured application-level debug logging via `log/slog` to stderr.

## PRD Reference

- §6.1 (Root Command) — `--verbose` flag exists for output enrichment; `--debug` is separate (internal tracing)
- §5.2 (Dual-Mode Data Flow) — stdout must stay clean for JSON/XML/MD and TUI; debug goes to stderr only
- §7.2 (Reliability) — rate limit warnings need a logging channel

## Research References

- `docs/gh-extensions-support-research.md` §6 — `GH_DEBUG` env var, `api.ClientOptions{Log, LogVerboseHTTP, LogColorize}`
- `docs/popular-extensions-research.md` §3 — gh-dash `--debug` flag pattern
- `docs/popular-extensions-research.md` §5 — gh-poi `--debug` flag pattern
- `docs/popular-extensions-research.md` §9 — gh-grep `DEBUG` env var pattern
- `docs/go-project-patterns-research.md` §7 — vivecaka uses `log/slog` (sibling project precedent)

## Files to Create

- `internal/debug/debug.go` — Debug logging package: `Init(enabled bool)`, `Log() *slog.Logger`, `Enabled() bool`
- `internal/debug/debug_test.go` — Tests: disabled produces no output, enabled produces structured stderr, GH_DEBUG activates

## Files to Modify

- `internal/cli/root.go` — Add `--debug` persistent flag; in `PersistentPreRunE` call `debug.Init()` based on `--debug` OR `GH_DEBUG` env var
- `internal/github/client.go` — When debug enabled: set `Log: os.Stderr`, `LogVerboseHTTP: true`, `LogColorize: true` on `api.ClientOptions`
- `internal/github/threads.go` — Add `slog.Debug()` at API call start, response received, pagination
- `internal/github/checks.go` — Add `slog.Debug()` at API call start, response received
- `internal/github/resolve.go` — Add `slog.Debug()` at mutation start, result
- `internal/github/reply.go` — Add `slog.Debug()` at reply start, result
- `internal/github/logs.go` — Add `slog.Debug()` at log fetch start, byte count received

## Execution Steps

### Step 1: Read context
1. Read CLAUDE.md
2. Read `docs/gh-extensions-support-research.md` §6 (go-gh ClientOptions, GH_DEBUG)
3. Read `docs/popular-extensions-research.md` §3, §5, §9 (extension --debug patterns)

### Step 2: Create debug package
- `internal/debug/debug.go`:
  - `Init(enabled bool)` — configures the default `slog.Logger`
    - When enabled: `slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))` with `source: true` for file:line
    - When disabled: `slog.New(slog.NewTextHandler(io.Discard, nil))`
  - `Log() *slog.Logger` — returns the configured logger
  - `Enabled() bool` — returns whether debug mode is active
  - Activation: `--debug` flag OR `os.Getenv("GH_DEBUG") != ""`
- Keep it minimal — no log rotation, no file output, no log levels beyond debug/warn

### Step 3: Add --debug flag to root command
- Add `Debug bool` to `GlobalFlags` struct
- Register `--debug` as persistent flag on root command: `"Enable debug logging to stderr"`
- In `PersistentPreRunE`: `debug.Init(flags.Debug || os.Getenv("GH_DEBUG") != "")`

### Step 4: Wire go-gh API transport logging
- In `client.go`, when `debug.Enabled()`:
  ```go
  opts := api.ClientOptions{
      Log:            os.Stderr,
      LogVerboseHTTP: true,
      LogColorize:    true,
  }
  ```
- When not enabled: default (go-gh respects `GH_DEBUG` automatically for transport, but we make it explicit)

### Step 5: Add slog.Debug calls to GitHub adapters
- Each adapter function gets entry/exit logging:
  - `threads.go`: `slog.Debug("fetching review threads", "owner", owner, "repo", repo, "pr", prNumber, "page", page)`
  - `checks.go`: `slog.Debug("fetching check runs", "ref", ref, "count", len(runs))`
  - `resolve.go`: `slog.Debug("resolving thread", "threadID", id)`
  - `reply.go`: `slog.Debug("posting reply", "threadID", threadID, "bodyLen", len(body))`
  - `logs.go`: `slog.Debug("fetching job logs", "jobID", jobID, "bytes", len(logData))`
- Add timing: `start := time.Now()` ... `slog.Debug("fetch complete", "duration", time.Since(start))`
- Rate limit logging: `slog.Warn("rate limit approaching", "remaining", remaining, "resetAt", resetAt)` (integrates with Task 3.2)

### Step 6: Verify stderr isolation
- Ensure ALL debug output goes to stderr
- Verify: `./bin/gh-ghent comments --pr 1 --debug --format json 2>/dev/null` produces clean JSON on stdout
- Verify: `./bin/gh-ghent comments --pr 1 --debug --format json 2>debug.log` captures debug in file
- Verify: TUI mode with `--debug` — Bubble Tea on stdout unaffected, debug on stderr

### Step 7: Unit tests
- `debug.Init(false)` → `debug.Enabled()` returns false, logging to discard
- `debug.Init(true)` → `debug.Enabled()` returns true, output captured in buffer
- `GH_DEBUG=1` without `--debug` → debug mode still activates
- Structured output contains expected keys (owner, repo, pr, duration)

## Verification

### L1: Unit Tests
```bash
make test
```

### L3: Binary Execution
```bash
make build
# Debug off (default): no debug output
./bin/gh-ghent comments --pr 1 --format json 2>debug.log
wc -l debug.log  # Should be 0 lines

# Debug on via flag
./bin/gh-ghent comments --pr 1 --debug --format json 2>debug.log
cat debug.log  # Should show structured debug lines
# Verify stdout is still clean JSON
./bin/gh-ghent comments --pr 1 --debug --format json 2>/dev/null | python3 -m json.tool

# Debug on via GH_DEBUG
GH_DEBUG=1 ./bin/gh-ghent comments --pr 1 --format json 2>debug.log
cat debug.log  # Should show both slog lines AND go-gh HTTP traces
```

## Completion Criteria

1. `--debug` persistent flag on root command
2. `GH_DEBUG` env var also activates debug mode
3. `log/slog` structured logging to stderr with file:line source
4. go-gh API transport logging activated when debug enabled
5. All GitHub adapter functions have entry/exit debug logs with timing
6. Debug output on stderr does NOT pollute stdout (clean JSON/XML/MD)
7. Debug disabled by default — zero overhead in production
8. All existing tests still pass
9. `make ci` passes
10. PROGRESS.md updated

## Commit

```
feat(debug): add structured debug logging with --debug flag and GH_DEBUG support

- log/slog to stderr with structured key-value output
- --debug flag and GH_DEBUG env var both activate debug mode
- go-gh API transport logging (LogVerboseHTTP) wired when enabled
- Entry/exit slog.Debug calls with timing on all GitHub adapters
- Zero overhead when disabled (log to io.Discard)
```

## Session Protocol

1. Read CLAUDE.md
2. Read this task file
3. **Change this task's status to `IN PROGRESS`**
4. Read `docs/gh-extensions-support-research.md` §6
5. Read `docs/popular-extensions-research.md` §3, §5, §9
6. Execute steps 1-7
7. Run verification (L1 → L3)
8. **Change this task's status to `DONE`**
9. Update `docs/PROGRESS.md`
10. Update CLAUDE.md Learnings if needed
11. Commit
