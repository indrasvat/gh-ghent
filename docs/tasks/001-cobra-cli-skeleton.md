# Task 1.2: Cobra CLI Skeleton

## Status: TODO

## Depends On
- Task 1.1: Repository scaffold (needs go.mod, Makefile, main.go)

## Parallelizable With
- Task 1.3: Domain types (can proceed independently once scaffold exists)

## Problem

ghent needs a Cobra command tree with root, comments, checks, resolve, and summary subcommands. All global flags must be wired, including TTY detection for dual-mode (TUI vs pipe) routing.

## PRD Reference

- §5.1 (Architecture) — cli/ directory structure
- §5.2 (Dual-Mode Data Flow) — TTY detection, TUI vs pipe routing
- §6.1 (Root Command) — global flags, version, repo resolution, TTY detection, `--no-tui`
- §6.2-6.5 — Subcommand flag definitions

## Research References

- `docs/go-project-patterns-research.md` §7 — Cobra RunE pattern, shared flags structs
- `docs/popular-extensions-research.md` §14 — Cross-cutting CLI patterns
- `docs/gh-extensions-support-research.md` §9 — CLI flags, output formatting, TTY detection

## Files to Create

- `internal/cli/comments.go` — Comments subcommand stub
- `internal/cli/checks.go` — Checks subcommand stub
- `internal/cli/resolve.go` — Resolve subcommand stub
- `internal/cli/summary.go` — Summary subcommand stub
- `internal/cli/flags.go` — GlobalFlags struct (repo, format, verbose, noTUI)
- `internal/version/version.go` — Version, Commit, Date + Print function

## Files to Modify

- `internal/cli/root.go` — Add subcommands, persistent flags, TTY detection in PersistentPreRunE
- `cmd/ghent/main.go` — Pass ldflags to version package

## Execution Steps

### Step 1: Read context
1. Read CLAUDE.md
2. Read PRD §6.1 (root command), §5.2 (dual-mode flow)

### Step 2: Create version package

### Step 3: Create shared flags with TTY detection
- `GlobalFlags` struct: Repo, Format, Verbose, NoTUI, IsTTY (resolved at runtime)
- PersistentPreRunE: resolve repo, detect TTY via `term.FromEnv()`, set IsTTY

### Step 4: Flesh out root command with all 4 subcommands
- Each stub returns "not implemented yet" error

### Step 5: Update main.go with ldflags

### Step 6: Unit tests
- Root has 4 subcommands
- `--version` works
- Global flags accessible from subcommands
- TTY detection logic

## Verification

### L1: Unit Tests
```bash
make test
```

### L3: Binary Execution
```bash
make build
./bin/gh-ghent --version
./bin/gh-ghent --help
./bin/gh-ghent comments --help
./bin/gh-ghent summary --help
./bin/gh-ghent comments --pr 1   # "not implemented" error
```

## Completion Criteria

1. `--help` lists comments, checks, resolve, summary subcommands
2. Each subcommand has its specific flags
3. `--no-tui` flag recognized
4. TTY detection wired (IsTTY set in PersistentPreRunE)
5. `make ci` passes
6. PROGRESS.md updated

## Commit

```
feat(cli): add Cobra command skeleton with TTY detection

- Root command with -R, --format, --verbose, --no-tui persistent flags
- Comments, checks, resolve, summary subcommands with specific flags
- TTY detection via term.FromEnv() for dual-mode routing
- Version info via ldflags
```

## Session Protocol

1. Read CLAUDE.md
2. Read this task file
3. **Change this task's status to `IN PROGRESS`**
4. Read PRD §5.2, §6.1-6.5
5. Read `docs/go-project-patterns-research.md` §7
6. Execute steps 1-6
7. Run verification (L1 → L3)
8. **Change this task's status to `DONE`**
9. Update `docs/PROGRESS.md`
10. Update CLAUDE.md Learnings if needed
11. Commit
