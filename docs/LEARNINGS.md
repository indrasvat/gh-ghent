# Learnings

> Concrete, actionable insights discovered during development.
> Format: `- **YYYY-MM-DD (task NNN):** [insight]`
>
> Previously in CLAUDE.md — moved here to keep CLAUDE.md focused on workflow and conventions.

## Go / Toolchain

- **2026-02-22 (task 000):** goimports requires blank line between external (`github.com/spf13/cobra`) and internal (`github.com/indrasvat/gh-ghent/...`) imports — golangci-lint v2 enforces this via the `local-prefixes` setting
- **2026-02-22 (task 000):** go-gh v2.13.0 pins lipgloss to a pre-release commit (`v1.1.1-0.20250319...`), not `@latest` — always let go-gh's version win for lipgloss
- **2026-02-22 (task 014):** `go get github.com/charmbracelet/bubbletea@latest` can downgrade go-gh from v2.13.0 to v2.11.2 — always re-pin go-gh after adding charmbracelet dependencies: `go get github.com/cli/go-gh/v2@v2.13.0`
- **2026-02-22 (task 016):** golangci-lint `unused` linter catches methods on unexported types — remove unused methods (like `isTopLevel`) rather than keeping them "for later"; re-add when actually needed
- **2026-02-23 (phase 6):** `path.Match` in Go stdlib only matches single path segments — for nested globs like `internal/*/*.go`, it works because `*` doesn't match `/`. For recursive `**` patterns, use `filepath.Match` or a dedicated glob library instead.

## GitHub API / go-gh

- **2026-02-23 (task 006):** go-gh REST `DoWithContext` expects JSON responses — use `RequestWithContext` for plain-text endpoints like job logs (`/actions/jobs/{id}/logs`)
- **2026-02-23 (task 006):** Not all check run IDs map to GitHub Actions job IDs — external CI checks (e.g., third-party integrations) return 404 on the logs endpoint. Graceful degradation (skip failed log fetch) is essential.
- **2026-02-22 (task 007):** `FetchThreads` only returns unresolved threads — any feature needing resolved threads must use `FetchResolvedThreads` (e.g., `--all --unresolve`)
- **2026-03-25 (await-review):** `PullRequestReviewThread` in GitHub GraphQL does NOT have `updatedAt` — use `comments(last: 1) { nodes { updatedAt } }` on the comment instead to detect edits
- **2026-03-25 (await-review):** Debounce on zero activity is wrong for review-await — a bot may take 2-4 min before posting its first comment (Codex shows 👀 during this time). Only debounce after at least one fingerprint change; use hard timeout as the safety valve for zero-activity cases
- **2026-03-25 (await-review):** When sorting parallel slices (IDs + metadata), sort as struct entries — `sort.Strings(ids)` alone mis-pairs the metadata slices. Codex P1 caught this.
- **2026-03-25 (await-review):** Take a baseline activity fingerprint BEFORE CI watch starts, not after — activity that happens during CI (fast bot reviews) is invisible if the initial probe is taken post-CI. Compare baseline vs post-CI to detect it.
- **2026-03-26 (bot-sweep):** GitHub GraphQL `author { __typename }` returns `"Bot"` for all GitHub App bots — this is the authoritative bot detection signal, superior to login-string matching. GraphQL author.login omits the `[bot]` suffix that REST includes.
- **2026-03-26 (bot-sweep):** GitHub's `resolveReviewThread` mutation is idempotent — resolving an already-resolved thread succeeds silently. No special "already resolved" handling needed.
- **2026-03-26 (bot-sweep):** Compute merge-readiness BEFORE applying display filters (--bots-only) — filtering mutates thread counts, which would make the PR appear merge-ready when unresolved human threads are hidden.
- **2026-03-26 (bot-sweep):** Codex bot review comments use `**<sub><sub>![P1 Badge]...` markup with "Useful? React with 👍 / 👎" footer. CodeRabbit uses `_⚠️ Potential issue_ | _🔴 Critical_` severity markers with `<!-- fingerprinting:... -->` HTML comments.
- **2026-03-29 (task 035):** Review stabilization cannot rely only on fingerprint deltas after CI. If a PR already has review threads when review-wait begins, treat existing thread presence as observed activity or the smart waiter will incorrectly fall through to low-confidence timeout on otherwise stable PRs.
- **2026-03-29 (task 035):** Historical review state and fresh review activity are different signals. Existing threads at watch start can justify a bounded quiet-check, but they must not increment `activity_count` or upgrade the result to `confidence=high`; only fingerprint changes during the watch window should do that.
- **2026-03-30 (task 036):** GitHub's documented `dismiss_stale_reviews_on_push` rule only auto-dismisses **approvals** on new reviewable pushes. It does not clear stale `CHANGES_REQUESTED` reviews, so ghent must treat stale blockers as an explicit dismissal workflow, not a passive branch-protection side effect.
- **2026-03-30 (task 036):** The stale-review feature should target **blocking review states**, not "bots" as the primary abstraction. GitHub's own Copilot review docs say Copilot always leaves `COMMENT` reviews, so bot-ness alone does not imply merge blocking.
- **2026-03-30 (task 036):** GitHub rejects self-authored blocking reviews on your own PR (`422 Review Can not request changes on your own pull request`). End-to-end stale-review testing therefore needs a second actor: another user, an installed bot, or a GitHub Actions workflow that posts reviews as `github-actions[bot]`.
- **2026-03-30 (task 036):** A branch-resident `push` workflow can synthesize stale-review fixtures even before the workflow exists on `main`, but `workflow_dispatch` cannot. For dogfooding on a feature branch, trigger on `push` with an explicit commit-message tag (for example `[synthetic-review]`), post the review to the branch PR, then push once more to stale it.

## Cobra CLI

- **2026-02-24 (task 031):** Cobra renders `--help`/`--version` templates before `PersistentPreRunE` runs — TTY detection for template funcs must use `term.FromEnv().IsTerminalOutput()` lazily inside the func, not rely on `Flags.IsTTY` which isn't set yet.
- **2026-02-24 (task 031):** Set `cmd.Version` to the raw version string (e.g., `version.Version`), not `version.String()` — custom version templates compose their own layout from `.Version` + helper funcs, so a pre-formatted string causes duplication.
- **2026-02-23 (task 009):** `gh` extension wrapper may duplicate output to stderr on non-zero exit codes — this is a gh CLI artifact, not a binary bug. Always test with `./bin/gh-ghent` directly to verify.
- **2026-02-23 (task 009):** Test repos without real PR approvals will have `is_merge_ready=false` even with clean threads and passing checks — the IsMergeReady logic correctly requires at least 1 APPROVED review.

## TUI / Bubble Tea

- **2026-02-24 (task 033):** Help bar declarations in `helpbar.go` don't auto-wire handlers — each key advertised in the help bar must have an explicit `key.Matches()` case in the corresponding view's `Update()` method or in `app.handleKey()`. Always verify keybindings with L4 tests after adding them.
- **2026-02-24 (task 033):** For actions that don't visually change the TUI screen (clipboard copy, browser open, async API calls), L4 tests need alternative verification: `pbpaste` for clipboard, `verify_tui_responsive()` for fire-and-forget commands, process spawn checks for browser opens.
- **2026-02-23 (task 023):** When a TUI view has separate `dur` and `status` columns concatenated on the right side, avoid setting the same label in both for a given state — causes duplicate text (e.g., `running... running...`). Only live testing against real in-progress CI caught this; 419 unit tests missed it entirely. Always trigger real CI runs and iterm2-driver test against live data for watch/polling features.
- **2026-02-24 (task 034):** When adding `--watch` to a command that already has pipe-mode output, stream watch status to **stderr** and final output to **stdout** — this lets users pipe stdout to `jq` while seeing progress on stderr. Pattern: `WatchChecks(ctx, os.Stderr, f, ...)` then fall through to normal output on `os.Stdout`.
- **2026-03-29 (task 035):** iTerm2 visual tests for watch/TUI startup must wait for the first rendered token (`watching`, `Event Log`, etc.) before capturing screenshots. Taking a screenshot immediately after `async_send_text()` can capture the shell instead of the TUI and produce a false visual pass.
- **2026-03-29 (task 035):** TUI polling loops must cap by the **remaining deadline**, not the configured total timeout. Comparing `reviewTimeout` to the poll interval is dead code once the watch has already been running.

## Testing

- **2026-02-22 (tasks 005/007/008):** `make install` symlinks bin/gh-ghent into gh extensions dir — always use `gh ghent` (not `./bin/gh-ghent`) for L3 testing to match real user experience
- **2026-02-22 (task 017):** L4 iterm2-driver tests may fail to find JSON markers if output is very long and scrolls off screen — check for multiple possible markers including end-of-output fields

## Parallel Agents / Worktrees

- **2026-02-22 (tasks 005/007/008):** When running parallel agents in worktrees that modify shared files (client.go, formatter.go), agents may step on each other's changes — verify the integrated result builds and passes lint after merging
- **2026-02-23 (phase 6):** When running 4 parallel agents in worktrees that all modify `domain/types.go` and `domain/ports.go`, merge one at a time and run `make ci-fast` after each — sequential merge prevents conflicts from compounding. Test count progression (419→443→465→470→489) provides confidence each merge is clean.
