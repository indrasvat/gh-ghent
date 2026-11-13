# Task 038: Repo flag agent surface

| Field | Value |
|-------|-------|
| **Status** | IN PROGRESS |
| **Phase** | 15 (Repo-flag agent surface) |
| **Priority** | P1 |
| **Depends on** | None |
| **Issue** | #21 |

## Objective

Make `-R` / `--repo` real for agents: the flag already works in the binary,
but `SKILL.md` never teaches it, README Quick Start never shows it, `GH_REPO`
is undocumented, and `resolveRepo` has zero tests.

## Problem

An agent following `SKILL.md` alone has no working path to a foreign repo.
The blessed command has no repo slot, and the PR-number step is bare
`gh pr view --json number -q .number` (current-repo only). `resolveRepo`
(`internal/cli/repo.go`) is untested, and its `SplitN(flag, "/", 2)` parsing
silently accepts `a/b/c` as owner `a`, repo `b/c`. The existing flag test
(`TestGlobalFlagsAccessibleFromSubcommand`) only proves `-R` is inherited,
not that any subcommand uses it.

## Desired Behavior

- `SKILL.md` teaches `-R owner/repo` on the blessed `status` command and on
  the `gh pr view` PR-number step, with a short foreign-repo paragraph.
- README Quick Start shows one `-R` foreign-repo example.
- `GH_REPO` fallback is documented (flag beats env beats remote) in README
  and `skill/references/command-reference.md`; `GH_GHENT_REPO` is called out
  as unrelated (release-repo override in `install.sh`).
- `resolveRepo` rejects anything but exactly `OWNER/REPO`.
- Tests prove: valid/invalid flag values, `GH_REPO` fallback, no-repo error
  naming `-R`, flag beats the checkout remote, and every subcommand parses
  `-R` into `Flags.Repo`. Binary tests prove each subcommand targets the
  `-R` repo without needing gh auth (dummy token + `--debug`).

## Files to Create

- `internal/cli/repo_test.go` — `resolveRepo` unit tests + per-subcommand
  flag-wiring test
- `docs/tasks/038-repo-flag-agent-surface.md` — this file

## Files to Modify

- `internal/cli/repo.go` — reject non-`OWNER/REPO` flag values
- `skill/SKILL.md` — `-R` on blessed command + `gh pr view` step, new
  "Targeting another repository" section
- `README.md` — Quick Start `-R` example, Global Flags precedence note
- `skill/references/command-reference.md` — "Repository resolution" section
- `scripts/test-binary.sh` — no-auth `--debug` repo-flag section (L3)
- `docs/PROGRESS.md` — Phase 15, session log
- `docs/LEARNINGS.md` — `resolveRepo` / go-gh findings

## Verification

### L1

```bash
make ci-fast
```

### L3

```bash
make build
bash scripts/test-binary.sh  # Repo Flag Tests section: 8/8, no auth needed
```

From a bare directory, with a dummy token (`--debug` proves targeting; the
401s are expected):

```bash
./bin/gh-ghent status -R oven-sh/bun --pr 27364 --format json --no-tui --debug
# fetching checks owner=oven-sh repo=bun pr=27364
./bin/gh-ghent comments -R cli/cli --pr 1 --format json --no-tui --debug
# request goes out for cli/cli, then 401 on the dummy token
```

### L5

```bash
bash scripts/test-agent-workflow.sh
```

### Interactive (shux)

Launch the built binary in a shux pane and exercise `-R` as a human would.
Without gh auth the TUI cannot render real PR data — the honest check is
that the binary reaches the fetch for the `-R` repo (visible via `--debug`)
instead of the checkout's repo. Screenshots are evidence only; they are
never committed (see issue #21 evidence rule).

## Verification Results

- **L1:** `make ci` PASS on 2026-11-12 (lint 0 issues; `go test -race
  -shuffle=on ./...` 767 passed, 1 skipped by design — the remote-fallback
  test skips when go-gh has no known hosts; `go vet` PASS).
- **L3:** Repo Flag Tests 8/8 PASS (all six subcommands target `-R
  oven-sh/bun`; invalid `-R` rejected; no-repo error names `-R`). Full
  `scripts/test-binary.sh`: 18 passed, 6 failed — the 6 are the pre-existing
  Real Repo Tests, which require gh auth and fail without it.
- **L5:** `bash scripts/test-agent-workflow.sh`: 3 passed, 8 failed — every
  failure is `Error: Not authenticated` (no gh auth in this cloud), not a
  behavior failure.
- **Interactive:** shux captures in `~/workspace/review/evidence/` (never
  committed): `tui-status-foreign-120x40.png` shows the real TUI headed
  `oven-sh/bun PR #48758` with all-zero counts (401 empty state — no auth);
  `pipe-debug-proof-200x60.png` shows `--debug` lines targeting
  `oven-sh/bun#48758` and `cli/cli#15003` before the 401s. No video: the
  change touches no TUI behavior.

## Completion Criteria

- [ ] An agent following `SKILL.md` alone, from a directory with no git
  checkout, runs the blessed command with `-R` against a real foreign repo
  (`oven-sh/bun#48758`, `cli/cli#15003` — both open, verified 2026-11-12)
  and gets real JSON instead of a repo-resolution error.
  NOT verified in this cloud: targeting is proven via `--debug` only (401s
  on the dummy token). Needs gh auth (one-time creds offered by the user).
- [ ] The skill's PR-number step returns the right PR number for a foreign
  repo (`gh pr view -R owner/repo <branch>` — the bare form with no selector
  fails client-side: `argument required when using the --repo flag`).
  NOT fully verified in this cloud: the fixed forms pass client-side
  validation (dummy token reaches the API call, then 401s); a real
  authenticated run is still blocked by the Bearer/`token` scheme gap.
  The right numbers were confirmed via direct authenticated API calls
  (`oven-sh/bun#48758`, `cli/cli#15003`).
- [x] README Quick Start shows a `-R` foreign-repo example.
- [x] `GH_REPO` fallback is documented (flag > env > remote), with a note
  that `GH_GHENT_REPO` is unrelated.
- [x] `resolveRepo` has unit tests (valid / invalid / missing-repo), and a
  CLI test proves each subcommand honors `-R` and that `-R` beats the
  checkout remote. The new tests fail if `resolveRepo` is bypassed.
- [x] `make test` (or `make ci`) passes.

## Commit

```text
test(cli): cover -R / --repo resolution and flag wiring

docs(skill): teach -R / --repo in the agent surface
```
