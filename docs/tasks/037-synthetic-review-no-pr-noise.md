# Task 037: Synthetic Review No-PR Skip

| Field | Value |
|-------|-------|
| **Status** | DONE |
| **Phase** | 14 (Stale Review Dismissal follow-up) |
| **Priority** | P1 |
| **Depends on** | 036 |
| **Issue** | follow-up bugfix |

## Objective

Ensure the synthetic stale-review harness does not create failing GitHub Actions noise when a `[synthetic-review]` push happens on a branch without an open pull request.

## Problem

The workflow intentionally uses a `push` trigger so it can dogfood stale-review fixtures from a feature branch, but the current script hard-fails when it cannot find an open PR for the pushed branch. That produces a red run even though "no PR exists" is an expected no-op state for this harness.

## Desired Behavior

- `workflow_dispatch` remains strict and still expects a PR number.
- `[synthetic-review]` push runs should:
  - post the synthetic review when an open PR exists for the branch
  - exit successfully with a clear notice when no open PR exists
- The workflow should not produce a failing run for the no-PR case.

## Files to Modify

- `.github/workflows/synthetic-review.yml`
- `docs/PROGRESS.md`
- `docs/LEARNINGS.md`

## Verification

### L1

```bash
make ci-fast
```

### Workflow sanity

```bash
yq e '.' .github/workflows/synthetic-review.yml >/dev/null
```

### Live verification

1. Push the branch before opening a PR with a commit message containing `[synthetic-review]`.
2. Confirm the workflow run completes successfully with a skip notice instead of failing.

## Verification Results

- **L1:** `make ci-fast` PASS
- **Workflow sanity:** `yq e '.' .github/workflows/synthetic-review.yml >/dev/null` PASS
- **Live verification:** PASS
  - branch: `fix/037-synthetic-review-no-pr-noise`
  - verification push commit: `eac4914` (`test(workflow): verify no-pr skip [synthetic-review]`)
  - Actions run: `Synthetic Review` run `23773130175`
  - result: successful run with notice `No open pull request found for branch fix/037-synthetic-review-no-pr-noise; skipping synthetic review.`
