# Task 035: Smart Await-Review + Skill Hardening

| Field | Value |
|-------|-------|
| **Status** | DONE |
| **Phase** | 13 (Review Monitor Hardening) |
| **Priority** | P0 |
| **Depends on** | 034 (status enhancement), 030 (agent skill), 010 (watch mode), 023 (watch TUI) |
| **Issue** | #13 |

## Objective

Make `gh ghent status --await-review` the single blessed agent workflow for PR review handling, with substantially fewer missed follow-up bot reviews.

This task has two equally important goals:

1. **Internal behavior:** make `--await-review` smarter so it captures more late or bursty bot review activity without asking agents to script sleeps or custom polling loops.
2. **Skill clarity:** make the ghent skill prescribe one unambiguous path so agents do **not** fall back to bare `--watch` after the first cycle and accidentally miss follow-up review comments.

## Problem

The current `--await-review` implementation is better than plain CI polling, but it still has two practical failure modes:

1. **Near-timeout burst loss:** review activity can begin just before the hard timeout, and ghent may exit before the burst is complete.
2. **Slow second-wave bot reviews:** one bot can post during the initial wait window while another posts much later; agents often switch to `--watch` for subsequent cycles, which only watches CI and misses new review comments.

The second failure mode is primarily a workflow/skill problem, not just a watcher bug.

## Non-Goals

- Do **not** add a large menu of public review-wait flags to the agent skill.
- Do **not** introduce a companion GitHub Action or external event service in this task.
- Do **not** require agents to hand-roll `sleep 60` loops in normal usage.

## Agent Contract

### Single blessed command

After PR creation, and again after **every push that addresses review feedback**, the skill should direct agents to run:

```bash
gh ghent status --pr <N> --await-review --solo --logs --format json --no-tui
```

This is the only primary command the skill should recommend for the review/fix/recheck loop.

### Explicit anti-footgun rule

The skill must say this plainly:

- **Do not switch to bare `--watch` after the first cycle when review comments may still arrive.**
- Use bare `--watch` only for CI-only waiting where review state is not part of the question.
- For review handling, always return to `status --await-review`.

### Intended agent loop

1. Run `status --await-review`
2. If checks fail, fix CI
3. If unanswered or unresolved review threads exist, fix/reply/resolve
4. Push
5. Run the **same** `status --await-review` command again
6. Repeat until ghent reports no actionable threads and review monitoring is sufficiently settled

## Design Principles

1. **One path in the skill, more intelligence in the CLI**
2. **Bias toward fewer missed reviews over minimum latency**
3. **Return machine-readable confidence/state, not a pile of tuning knobs**
4. **Use best-effort polling honestly; never imply certainty the API cannot provide**

## Proposed CLI Behavior

Keep the public command shape centered on `--await-review`. Do not require new flags in the normal path.

### Review wait state machine

`status --await-review` should use this bounded multi-phase wait:

1. **Baseline**
   - Capture baseline review activity fingerprint before CI watch begins.
   - Reuse existing baseline behavior so activity during CI is not lost.

2. **Active review polling**
   - Poll lightweight review activity at the normal cadence.
   - Detect thread/review changes as today.
   - Also use review/timeline-derived signals internally where they improve detection of new reviewer activity.

3. **Late-activity grace**
   - If new review activity is detected within the final debounce window before hard timeout, extend the deadline by one bounded grace window.
   - Cap extensions so the wait remains bounded.
   - Purpose: avoid cutting off a review burst that clearly started before timeout.

4. **Settle**
   - When activity has been observed and then remains quiet for the debounce window, do **not** exit immediately.

5. **Tail confirmation**
   - After apparent settle, perform a small number of sparse confirmation probes inside ghent.
   - If any probe detects new review activity, re-arm active polling.
   - If probes stay quiet, finish with higher confidence.

6. **Bounded timeout exit**
   - If the full bounded wait expires, exit with a timeout/provisional state, not an implicitly complete one.

### Tail confirmation defaults

Default behavior should be internal and opinionated:

- Short confirmation probe after initial settle
- One or two additional sparse probes after that
- Total added time should be bounded and modest relative to current behavior

The exact durations can be implementation-tuned, but the product behavior must be:

- noticeably better than immediate exit on first settle
- still bounded
- not exposed as required skill configuration

## Output Contract

Add a new machine-readable review monitoring object to `status` output.

### JSON

```json
"review_monitor": {
  "phase": "settled",
  "confidence": "high",
  "activity_count": 3,
  "wait_seconds": 412,
  "tail_probes": 2,
  "tail_rearmed": false
}
```

Minimum required fields:

- `phase`: `awaiting` | `settled` | `timeout`
- `confidence`: `low` | `medium` | `high`
- `activity_count`
- `wait_seconds`

Optional but useful:

- `tail_probes`
- `tail_rearmed`

### Semantics

- `settled` + `high`: observed activity and quiet remained stable through confirmation probes
- `settled` + `medium`: observed activity and quiet, but confirmation was weaker
- `timeout` + `low`: bounded review wait expired; follow-up comments may still appear

`review_settled` should either be replaced by `review_monitor` or retained only as a compatibility alias while the new field becomes the canonical skill-facing signal.

## Skill Changes

Update `skill/SKILL.md` so the first command, the retry command, and the decision tree all use the same status command.

### Required guidance changes

- Start with `status --await-review`, not `--watch`
- After fixing and pushing, re-run `status --await-review`
- Never recommend bare `--watch` for post-fix review cycles
- Treat low-confidence timeout as “re-check required after push/fix cycle,” not “all clear”

### Required decision tree shape

1. `checks.overall_status == "failure"` → fix CI
2. `comments.unanswered_count > 0` → fix/reply/resolve
3. `comments.unresolved_count > 0` → resolve remaining addressed threads
4. `review_monitor.phase == "timeout"` with actionable work already done → push/re-run `status --await-review`
5. `is_merge_ready == true` and `review_monitor.confidence != "low"` → stop

## Formatter / UX Changes

Human-facing output should make provisional states obvious.

### Markdown / TTY

- When review monitoring exits on timeout/low confidence, print a concise warning that additional bot reviews may still arrive.
- When it exits after settle/high confidence, print a concise “review activity stabilized” signal.

### Pipe mode

- Preserve token efficiency
- Keep the new monitor object small and stable

## Files to Modify

- `internal/github/watcher.go`
- `internal/github/activity.go`
- `internal/github/watcher_review_test.go`
- `internal/tui/watcher.go`
- `internal/tui/watcher_test.go`
- `internal/domain/types.go`
- `internal/cli/status.go`
- `internal/formatter/json.go`
- `internal/formatter/markdown.go`
- `internal/formatter/xml.go`
- `skill/SKILL.md`
- `skill/references/command-reference.md`
- `skill/references/agent-workflows.md`
- `README.md`
- `docs/LEARNINGS.md`
- `docs/PROGRESS.md`
- `.claude/automations/test_ghent_await_review.py`

## L4 Visual Verification

Create or update:

- `.claude/automations/test_ghent_await_review.py`

Capture and review screenshots for:

- `ghent_await_review_cli_timeout.png`
- `ghent_await_review_cli_settled.png`
- `ghent_await_review_ci_phase.png`
- `ghent_await_review_awaiting.png`
- `ghent_await_review_summary.png`
- `ghent_await_review_timeout_warning.png`
- `ghent_await_review_tail_settled.png`

The script must follow the `iterm2-driver` skill conventions:

- Python script with `uv` inline metadata
- use iTerm2 API directly
- poll screen content instead of relying only on sleeps
- save named screenshots under `.claude/screenshots/`
- print concrete PASS/FAIL evidence

## Acceptance Criteria

1. The ghent skill presents one clear review loop centered on `status --await-review`.
2. The skill explicitly forbids switching to bare `--watch` for review-comment follow-up cycles.
3. `--await-review` no longer exits immediately after the first quiet period; it performs bounded internal confirmation.
4. Review activity that begins just before timeout gets a bounded grace window instead of immediate cutoff.
5. `status` output includes machine-readable review-monitor state/confidence.
6. Human-facing output clearly distinguishes stable settle vs provisional timeout.
7. Existing CI-only watch behavior remains available for CI-only use cases.

## Verification

### L1

```bash
make ci-fast
```

Result:
- `make ci-fast` PASS on 2026-03-29
- `DONE 681 tests, 1 skipped in 4.357s`

### L3

Verified against real PRs:

1. Provisional timeout path on quiet PR:

```bash
gh ghent status -R indrasvat/doot --pr 1 --await-review --review-timeout 5s --solo --logs --format json --no-tui
```

Observed final monitor:
- `review_monitor.phase == "timeout"`
- `review_monitor.confidence == "low"`
- compatibility alias `review_settled` present with the same values

2. Settled/medium-confidence path on historical multi-bot PR:

```bash
gh ghent status -R indrasvat/yathaavat --pr 1 --await-review --solo --logs --format json --no-tui
```

Observed final monitor:
- `review_monitor.phase == "settled"`
- `review_monitor.confidence == "medium"`
- `review_monitor.tail_probes == 2`
- `review_monitor.activity_count == 0`
- compatibility alias `review_settled` present with the same values

The verification target is not perfect certainty; it is a materially lower miss rate and a clear machine-readable distinction between stable and provisional exits.

### L4

Add/update visual verification for:

- awaiting review
- review activity detected
- stable settled completion
- timeout/provisional completion with warning text

Executed:

```bash
uv run .claude/automations/test_ghent_await_review.py
```

Result:
- 11/11 PASS on 2026-03-29
- real PR targets: `indrasvat/yathaavat#1` and `indrasvat/doot#1`
- screenshots captured and manually reviewed from the latest run
- harness now asserts that no prefixed iTerm2 test sessions remain after teardown

## Completion Criteria

1. Smart await-review logic implemented and tested
2. Skill rewritten around a single blessed review loop
3. README / command reference updated to match the skill
4. L3 and L4 verification completed
5. `docs/PROGRESS.md` and `docs/LEARNINGS.md` updated

## Visual Test Results

- Reviewed `/Users/indrasvat/code/github.com/gh-ghent/.claude/screenshots/ghent_await_review_ci_phase_20260329_172511.png`: initial watch screen renders correctly with `watching`, elapsed/poll metadata, and event-log placeholder; no blank-frame or paint glitch remains.
- Reviewed `/Users/indrasvat/code/github.com/gh-ghent/.claude/screenshots/ghent_await_review_cli_timeout_20260329_181458.png`: CLI markdown path cleanly shows the `## Review Monitor` block, low-confidence timeout state, warning text, and a `__GHENT_DONE__:0` completion marker in a normal terminal session.
- Reviewed `/Users/indrasvat/code/github.com/gh-ghent/.claude/screenshots/ghent_await_review_cli_settled_20260329_181705.png`: CLI markdown path cleanly shows the settled review-monitor block with `confidence medium`, `activity 0`, and a successful completion marker, without dropping into TUI mode.
- Reviewed `/Users/indrasvat/code/github.com/gh-ghent/.claude/screenshots/ghent_await_review_awaiting_20260329_172512.png` and `/Users/indrasvat/code/github.com/gh-ghent/.claude/screenshots/ghent_await_review_tail_settled_20260329_172543.png`: the intermediate states are visually distinct, with `awaiting reviews` transitioning to `confirming review quiet` and the event log showing the new stabilization message.
- Reviewed `/Users/indrasvat/code/github.com/gh-ghent/.claude/screenshots/ghent_await_review_summary_20260329_175602.png`: historical settled state now shows an amber `Review activity settled` banner with `confidence medium`, `activity 0`, and tail metadata instead of incorrectly claiming high-confidence fresh activity.
- Reviewed `/Users/indrasvat/code/github.com/gh-ghent/.claude/screenshots/ghent_await_review_timeout_warning_20260329_175610.png`: provisional state shows an amber `Review monitor provisional` banner with explicit warning text that more bot comments may still arrive and no phantom activity count.
- Full L4 harness result: `uv run .claude/automations/test_ghent_await_review.py` → 11/11 PASS against real GitHub PRs and zero lingering `ghent-await-review-*` iTerm2 sessions.
