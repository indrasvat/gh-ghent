# Task 039: Cloud sessions — vault tokens, auth errors, PR lookup, checks fallback

| Field | Value |
|-------|-------|
| **Status** | DONE |
| **Phase** | 16 (Cloud sessions) |
| **Priority** | P1 |
| **Depends on** | 038 |
| **Issue** | #23 |

## Objective

Make ghent usable from a cloud session: no `gh`, no TTY, a token from a vault
(or none), least-privilege tokens.

## Decisions

- **Bearer is opt-in** (`GH_GHENT_AUTH_SCHEME=bearer`); default stays `token`,
  what gh sends. go-gh never overrides a preset `Authorization` header and
  still sends it only to the resolved host, so no fork is needed.
- **No anonymous mode.** GraphQL rejects anonymous requests and REST review
  comments have no `isResolved`, so a REST fallback would report resolved
  threads as open. Anonymous REST is also 60 requests/hour per IP — a shared
  cloud IP hit that limit after five probe calls. Instead: a clear
  "token required even for public repos" error.
- **403 is not 401.** A non-rate-limit 403 is now a `PermissionError` carrying
  GitHub's message, not "Not authenticated".
- **`--pr` resolves branches** via `GET /pulls?head=OWNER:BRANCH&state=open`
  (GitHub filters on head owner). Omitted → current git branch. Never takes a
  first match: zero or several is an error that says what to pass.
- **Checks fallback only on 403** from check-runs: Actions jobs (job ID =
  check-run ID, so `--logs` keeps working) + combined commit status. Anything
  else, including the fallback itself being denied, fails loudly. Output
  carries `fallback` (`checks_fallback` in compact status).
- **Release checksums.** No release since at least v0.6.0 shipped
  `checksums.txt`, so `install.sh` always failed at its checksum step:
  gh-extension-precompile writes checksums only when GPG signing is set up.
  `build-release.sh` now writes `dist/checksums.txt`, which the action uploads.

## Verification Results

- **L1:** `make ci` green.
- **L3:** `make test-binary` 27/27.
- **Cloud simulation** (`.shux/out/cloudsim/`, not committed): Debian
  container with no `gh`/Go, `GH_TOKEN=<placeholder>`, all HTTPS through a
  mitmproxy "vault" that swaps the real token in only on
  `Authorization: Bearer <placeholder>` to `api.github.com`. Before: every
  case "Not authenticated". After: default scheme → 401 message with the Bearer
  hint; bearer → blessed command with no `--pr` returns PR #2 JSON;
  no token → "No GitHub token found…"; `--debug` shows `Bearer ████`.
- **Least-privilege, real:** private scratch repo, Actions token without
  `checks` → check-runs 403 "Resource not accessible by integration". `main`:
  "Not authenticated". Branch: `checks --logs` and `status` (no `--pr`) return
  jobs + commit statuses, the `fallback` note, and the failing job's log excerpt.
- **L4:** iTerm2 not running, so iterm2-driver scripts could not connect;
  `test_ghent_errors.py` / `test_ghent_pipe.py` updated for the new
  `--pr is required` message. TUI verified in a shux pane instead: `status`
  with `--pr <branch>` renders PR #2.

## Not done

- TUI checks view does not show the `fallback` note (pipe output does).
- `uploads.github.com` evidence uploads (issue row 4) are a vault allowlist
  matter, not ghent's.
