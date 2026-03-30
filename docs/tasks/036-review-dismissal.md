# Task 036: Stale Blocking Review Dismissal

| Field | Value |
|-------|-------|
| **Status** | DONE |
| **Phase** | 14 (Stale Review Dismissal) |
| **Priority** | P0 |
| **Depends on** | 009 (status), 007 (resolve), 008 (reply), 030 (agent skill), 034 (status enhancement), 035 (await-review hardening) |
| **Issue** | #15 |

## Objective

Add first-class support for detecting and dismissing stale blocking reviews so ghent can drive a PR from "fixed" to actually mergeable without requiring raw `gh api` fallback calls.

This task has three linked deliverables:

1. **Review intelligence:** enrich review data with commit SHA, bot classification, and staleness.
2. **Review action:** add `gh ghent dismiss` for safe, filtered review dismissal.
3. **Operator guidance:** surface stale blockers in `status` output and update the skill/workflow docs.

## Validated Problem

The gap is real for ghent, independent of the issue text:

1. ghent currently blocks merge readiness on **any** `CHANGES_REQUESTED` review in `IsMergeReady`, with no notion of whether that review is attached to the current PR head commit.
2. ghent can resolve review **threads**, but it has no primitive for dismissing review **verdicts**.
3. GitHub documents a REST dismissal endpoint for reviews and includes `commit_id` on review objects, so the platform supports this workflow directly.
4. GitHub's documented stale-review automation only covers **approvals on push**, not stale `CHANGES_REQUESTED` reviews.
5. Bot behavior is not uniform. Some bots, such as GitHub Copilot code review, never block merges because they leave `COMMENT` reviews only. The product abstraction must therefore be "dismiss stale blocking reviews," with optional bot filtering, not "handle bots" generically.

## Non-Goals

- Do **not** silently auto-dismiss reviews from `status`, `resolve`, or `reply`.
- Do **not** change `is_merge_ready` to ignore stale blockers before GitHub has actually dismissed them.
- Do **not** add a new TUI tab for dismissal in this task.
- Do **not** hard-code behavior for one vendor bot beyond existing reusable bot-detection helpers.
- Do **not** assume every stale review should be dismissed; keep author/review/staleness filters explicit.
- Do **not** add a broad dismissal mode. This command is stale-only by design.

## Product Shape

### 1. Domain and API

- Extend `domain.Review` with:
  - review numeric ID for REST dismissal
  - commit SHA / commit ID
  - `IsStale`
  - author type metadata sufficient for bot detection
- Extend the review fetch path to compare review commit SHA against the PR head SHA.
- Add a dedicated dismiss result type with per-review success/failure reporting.

### 2. `gh ghent dismiss`

- Pipe-first command, no TUI.
- Target selection:
  - `--review`
  - `--author`
  - `--bots-only`
- Safety:
  - only stale `CHANGES_REQUESTED` reviews are ever eligible
  - `--message` required unless `--dry-run`
  - `--dry-run` emits exact dismissal candidates
  - partial-failure exit code behavior matches existing batch commands

### 3. `gh ghent status`

- Preserve conservative merge-readiness semantics.
- Annotate stale blocking reviews in TUI/markdown.
- Add `stale_reviews` helper output in JSON/XML for agents.
- Recommend the exact dismiss command when stale blockers are present.

### 4. Skill / Workflow

- Insert stale-review dismissal into the post-fix agent loop after comment resolution and before merge.
- Keep `status --await-review` as the single monitoring entry point.
- Teach agents to dismiss only when the review is stale and the reason is explicit.

## PRD Reference

- `docs/PRD.md` §6.5.1 — Dismiss Command
- `docs/PRD.md` §6.6 — Status Command

## Research References

- `docs/github-api-research.md` §8 — REST pull request reviews and dismissals
- GitHub Docs: REST API pull request reviews
- GitHub Docs: branch protection / rules `dismiss_stale_reviews_on_push`
- GitHub Docs: GitHub Copilot code review behavior

## Files to Modify

- `internal/domain/types.go` — review metadata + dismiss result types
- `internal/domain/ports.go` — dismissal port
- `internal/github/reviews.go` — fetch review commit SHA, head SHA, staleness
- `internal/github/reviews_test.go`
- `internal/github/dismiss.go` — REST dismissal client
- `internal/github/client.go` — interface satisfaction
- `internal/cli/dismiss.go` — new Cobra command
- `internal/cli/status.go` — stale review helper output + suggestion text
- `internal/cli/summary_test.go` — merge-readiness remains conservative
- `internal/tui/status.go` — stale markers in approvals section
- `internal/tui/status_test.go`
- `internal/formatter/json.go`
- `internal/formatter/xml.go`
- `internal/formatter/markdown.go`
- `internal/formatter/json_test.go`
- `internal/formatter/xml_test.go`
- `internal/formatter/markdown_test.go`
- root CLI wiring location
- `skill/SKILL.md`
- `skill/references/command-reference.md`
- `skill/references/agent-workflows.md`
- `README.md`
- `.claude/automations/test_ghent_dismiss.py`
- `.github/workflows/synthetic-review.yml`
- `scripts/test-binary.sh`
- `scripts/test-agent-workflow.sh`
- `docs/LEARNINGS.md`
- `docs/PROGRESS.md`

## Execution Plan

### Step 1: Review model enrichment

- Fetch PR head SHA alongside reviews.
- Capture review commit SHA from the API response.
- Compute `IsStale = review.commit_sha != pr.head_sha`.
- Preserve current review ordering and existing formatter compatibility.

### Step 2: REST dismiss client

- Implement `DismissReview(ctx, owner, repo, pr, reviewID, message)`.
- Accept numeric review ID for the REST path.
- Return dismissed review metadata with new state.
- Reuse existing retry/error-classification patterns from reply/checks code.

### Step 3: CLI command

- Add `gh ghent dismiss`.
- Implement target discovery/filtering against fetched reviews.
- Hard-code the safety boundary: stale `CHANGES_REQUESTED` only.
- Support dry-run preview plus batched dismissal.
- Define exit code behavior:
  - `0` all success
  - `1` partial success
  - `2` hard failure / invalid invocation

### Step 4: Status enrichment

- Keep `IsMergeReady` blocking on stale `CHANGES_REQUESTED`.
- Add `stale_reviews` helper data to structured output.
- Mark stale blocking reviews visually in status output.
- Print an explicit recommendation line when stale blockers exist, for example:
  - `Suggested: gh ghent dismiss --pr 42 --message "..."`

### Step 5: Skill and documentation

- Update the skill decision tree to check stale blockers after threads are handled.
- Document the distinction between threads vs reviews.
- Add examples for dry-run and filtered dismissal.

### Step 6: Verification

- L1 unit coverage for filtering, staleness detection, REST dismissal request shaping, formatter fields, and status rendering.
- L3 real-repo verification against at least one PR with stale blocking reviews and one control PR without them.
- L5 agent workflow verification for dry-run output and suggested command flow.
- L4 visual verification is required because the approvals section now shows stale markers and stale counts.

## Acceptance Criteria

1. `status` exposes stale blocking reviews without falsely marking the PR merge-ready.
2. `dismiss` can safely target one review or a filtered batch.
3. Bot-only targeting is optional, not the primary abstraction.
4. Dry-run output is sufficient for agent or human confirmation.
5. Skill guidance closes the current raw-`gh api` escape hatch.
6. Documentation clearly explains review threads vs review verdicts.

## Verification

### L1

```bash
make ci-fast
```

### L3

```bash
make install

# Inspect stale review metadata
gh ghent status -R <owner>/<repo> --pr <n> --format json --no-tui | jq '.stale_reviews'

# Preview dismissal candidates
gh ghent dismiss -R <owner>/<repo> --pr <n> --dry-run --format json --no-tui

# Execute filtered dismissal
gh ghent dismiss -R <owner>/<repo> --pr <n> --bots-only --message "Superseded by current HEAD" --format json --no-tui
```

### L5

```bash
gh ghent status -R <owner>/<repo> --pr <n> --await-review --format json --no-tui | jq '{merge_ready: .is_merge_ready, stale_reviews: .stale_reviews}'
```

## Verification Results

- **L1:** `make ci-fast` PASS
- **L3:** `bash scripts/test-binary.sh` PASS (19/19)
  - control repos: `indrasvat/tbgs#1`, `indrasvat/doot#1`
  - stale-review repo: `clayliddell/AgentVM#10`
- **L4:** `uv run .claude/automations/test_ghent_dismiss.py` PASS (5/5)
- **L5:** `bash scripts/test-agent-workflow.sh` PASS (11/11)
- **Dogfood:** `indrasvat/gh-ghent#16`
  - synthetic `github-actions[bot]` `REQUEST_CHANGES` review created by `.github/workflows/synthetic-review.yml`
  - follow-up push made it stale
  - `gh ghent status` surfaced the stale blocker
  - `gh ghent dismiss` successfully dismissed it and no stale blockers remained
  - follow-up review hardening changed broad zero-match dismissal into a safe no-op success (`exit 0`, empty result set)

## Visual Test Results

- Script: `.claude/automations/test_ghent_dismiss.py`
- Reviewed screenshots:
  - `.claude/screenshots/ghent_dismiss_status_tui_20260330_123411.png`
  - `.claude/screenshots/ghent_dismiss_status_md_20260330_123417.png`
  - `.claude/screenshots/ghent_dismiss_dry_run_md_20260330_123421.png`
  - `.claude/screenshots/ghent_dismiss_dry_run_json_20260330_123425.png`
  - `.claude/screenshots/ghent_dismiss_status_tui_20260330_135026.png`
  - `.claude/screenshots/ghent_dismiss_status_md_20260330_135031.png`
  - `.claude/screenshots/ghent_dismiss_dry_run_md_20260330_135035.png`
  - `.claude/screenshots/ghent_dismiss_dry_run_json_20260330_135039.png`
- Findings:
  - TUI approvals section clearly shows stale count and `(stale)` marker without layout breakage.
  - Markdown status output recommends the exact dismiss command.
  - Dry-run dismiss output renders cleanly in both Markdown and JSON forms.

## Completion Criteria

1. New command implemented and wired into help/docs
2. Status shows stale blockers distinctly
3. Merge-readiness logic remains correct and conservative
4. Skill/docs updated
5. `docs/PROGRESS.md` and `docs/LEARNINGS.md` updated
6. Binary, agent-workflow, and iTerm2 visual verification cover stale-review detection and dry-run dismissal
