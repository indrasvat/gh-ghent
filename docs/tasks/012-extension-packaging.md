# Task 3.3: Extension Packaging

## Status: TODO

## Depends On
- Task 3.1: Watch mode pipe
- Task 3.2: Error handling hardening

## Parallelizable With
- Task 3.4: README + help text (partially)

## Problem

ghent needs to be installable as a `gh` extension via `gh extension install indrasvat/ghent`. The release workflow (gh-extension-precompile), binary naming convention, and install-test flow must all work end-to-end.

## PRD Reference

- §4 (Technology Stack) — gh-extension-precompile v2, binary naming
- §7.3 (Compatibility) — platforms: linux/darwin/windows x amd64/arm64

## Research References

- `docs/popular-extensions-research.md` §13 — gh-extension-precompile GitHub Action
- `docs/gh-extensions-support-research.md` §3 — Precompiled binary distribution
- `docs/gh-extensions-support-research.md` §2 — Extension naming (`gh-ghent` binary → `gh ghent` command)

## Files to Create

- `.claude/automations/test_ghent_install.py` — iterm2-driver L4 visual test for extension install flow (per `docs/testing-strategy.md` §8)

## Files to Modify

- `.github/workflows/release.yml` — Verify and finalize precompile configuration
- `.github/workflows/ci.yml` — Ensure binary builds on all platforms
- `Makefile` — Add `install-local` target for dev testing

## Execution Steps

### Step 1: Read context
1. Read PRD §4 (naming convention: repo `ghent`, binary `gh-ghent`)
2. Read `docs/gh-extensions-support-research.md` §2-3

### Step 2: Verify release workflow
- Ensure `gh-extension-precompile@v2` handles `gh-ghent-<os>-<arch>` naming
- Verify `generate_attestations: true` is set
- Verify permissions: `contents: write`, `id-token: write`, `attestations: write`

### Step 3: Add local install target
- `make install-local`: builds binary, copies to `$(gh extension list --directory)/gh-ghent/`
- Allows testing `gh ghent` command locally without a release

### Step 4: Test install flow
- Create a test tag locally (don't push)
- Verify `go build -o gh-ghent` produces correct binary
- Test: `gh extension install .` from repo root

### Step 5: Cross-platform build verification
- `GOOS=linux GOARCH=amd64 go build -o gh-ghent`
- `GOOS=darwin GOARCH=arm64 go build -o gh-ghent`
- `GOOS=windows GOARCH=amd64 go build -o gh-ghent.exe`

## Verification

### L3: Binary Execution
```bash
make build
# Test local extension install
gh extension install .
gh ghent --version
gh ghent --help
gh extension remove ghent
```

### L4: Visual (iterm2-driver)
Create `.claude/automations/test_ghent_install.py` following canonical template in `docs/testing-strategy.md` §5:
```bash
uv run .claude/automations/test_ghent_install.py
```
- Verify: `gh extension install .` succeeds
- Verify: `gh ghent --version` shows version string
- Verify: `gh extension list` includes ghent
- Verify: `gh extension remove ghent` succeeds
- Screenshots: `ghent_install.png`, `ghent_list.png`

## Completion Criteria

1. Release workflow configured correctly
2. Binary naming: `gh-ghent` (not `ghent`)
3. Local install via `gh extension install .` works
4. Cross-platform builds succeed
5. `make ci` passes
6. PROGRESS.md updated

## Commit

```
chore(release): finalize extension packaging and install flow

- Verified gh-extension-precompile release workflow
- Added install-local Makefile target
- Cross-platform build verification
```

## Session Protocol

1. Read CLAUDE.md
2. Read this task file
3. **Change this task's status to `IN PROGRESS`**
4. Read PRD §4
5. Read `docs/gh-extensions-support-research.md` §2-3
6. Execute steps 1-5
7. Run verification (L3 → L4)
8. **Change this task's status to `DONE`**
9. Update `docs/PROGRESS.md`
10. Update CLAUDE.md Learnings if needed
11. Commit
