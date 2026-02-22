# Task 1.1: Repository Scaffold

## Status: TODO

## Depends On
- None (first task)

## Parallelizable With
- None (all other tasks depend on this)

## Problem

ghent needs a Go project skeleton with build tooling, linting, git hooks, and release configuration before any feature code can be written. Must include Bubble Tea and Lipgloss as core dependencies for the TUI.

## PRD Reference

- §4 (Technology Stack) — Go 1.26, go-gh v2.13.0, Cobra v1.10+, Bubble Tea v1.3+, Lipgloss v1.1.x, Bubbles, golangci-lint v2.9.0, lefthook 2.1.1, gh-extension-precompile v2
- §5.1 (Directory Structure) — full directory layout including tui/ package

## Research References

- `docs/go-project-patterns-research.md` §5 — Comprehensive Makefile with all targets
- `docs/go-project-patterns-research.md` §9 — golangci-lint v2 configuration
- `docs/go-project-patterns-research.md` §8 — lefthook configuration
- `docs/go-project-patterns-research.md` §10 — GoReleaser patterns (reference only; using gh-extension-precompile instead)
- `docs/popular-extensions-research.md` §13 — gh-extension-precompile GitHub Action
- `docs/gh-extensions-support-research.md` §3 — Precompiled binary distribution

## Files to Create

- `go.mod` — Module `github.com/indrasvat/ghent`, Go 1.26
- `cmd/ghent/main.go` — Minimal entry point (calls `cli.Execute()`)
- `internal/cli/root.go` — Placeholder root command
- `Makefile` — Full target set per CLAUDE.md
- `.golangci.yml` — Curated linter set for golangci-lint v2
- `lefthook.yml` — Pre-push hook → `make ci`
- `.github/workflows/ci.yml` — Lint + test on push/PR
- `.github/workflows/release.yml` — `gh-extension-precompile` on tag push (handles binary naming for gh extensions)

Note: Use `gh-extension-precompile` (not GoReleaser) for release binaries. It auto-handles the `gh-ghent-<os>-<arch>` naming convention. GoReleaser is NOT needed — precompile covers cross-compilation + checksums.

## Files to Modify

- `.gitignore` — Add Go entries (bin/, dist/, coverage/)

## Execution Steps

### Step 1: Read context
1. Read CLAUDE.md (conventions, architecture)
2. Read PRD §4 (tech stack) and §5.1 (directory structure)
3. Read `docs/go-project-patterns-research.md` §5 (Makefile patterns)

### Step 2: Initialize Go module
- `go mod init github.com/indrasvat/ghent`
- Add all deps: go-gh v2.13.0, Cobra, Bubble Tea, Lipgloss, Bubbles
- `go mod tidy`

### Step 3: Create minimal main.go and root command
- `cmd/ghent/main.go`: import `internal/cli`, call `cli.Execute()`
- `internal/cli/root.go`: minimal `NewRootCmd()`, `Execute()` function

### Step 4: Create Makefile
- All targets listed in CLAUDE.md
- Binary output to `bin/gh-ghent`
- Version injection via ldflags

### Step 5: Create linter, hooks, release configs

### Step 6: Create CI workflows

### Step 7: Create scripts/ stubs and testdata/
- `scripts/test-binary.sh` — minimal L3 stub: builds binary, runs `gh-ghent --version`, checks exit 0
- `scripts/test-agent-workflow.sh` — minimal L5 stub: placeholder that exits 0
- `testdata/` — empty directory with `.gitkeep` (fixtures added in Phase 2)

### Step 8: Update .gitignore

## Verification

### L1: Unit Tests
```bash
make test    # Should pass (no tests yet, no errors)
```

### L3: Binary Execution
```bash
make build
./bin/gh-ghent --help
make ci      # Should pass all checks
```

## Completion Criteria

1. `make build` produces `bin/gh-ghent`
2. `./bin/gh-ghent --help` shows help text
3. `make ci` passes
4. `go mod tidy` reports no changes
5. go.mod includes go-gh, Cobra, Bubble Tea, Lipgloss, Bubbles
6. PROGRESS.md updated

## Commit

```
feat(scaffold): initialize repository with build tooling

- Go module with go-gh v2, Cobra, Bubble Tea, Lipgloss dependencies
- Comprehensive Makefile with ci, lint, test, build targets
- golangci-lint v2, lefthook, gh-extension-precompile configs
- GitHub Actions CI and release workflows
```

## Session Protocol

1. Read CLAUDE.md
2. Read this task file
3. **Change this task's status to `IN PROGRESS`**
4. Read PRD §4, §5.1
5. Read `docs/go-project-patterns-research.md` §3-4, §6, §8
6. Execute steps 1-7
7. Run verification (L1 → L3)
8. **Change this task's status to `DONE`** (or `BLOCKED` with details in PROGRESS.md)
9. Update `docs/PROGRESS.md` — mark task done + session log entry
10. Update CLAUDE.md Learnings section if new insights
11. Commit with message above
