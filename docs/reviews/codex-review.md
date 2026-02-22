# Codex Review of ghent Project Documentation

> **Reviewer:** OpenAI Codex v0.104.0 (gpt-5.3-codex, reasoning effort: xhigh)
> **Date:** 2026-02-22
> **Session ID:** 019c849b-ae67-7511-9241-7bf805cc7e56
> **Method:** Three parallel audit agents + one second-pass comprehensive audit
> **Files Reviewed:** All 16 listed files (CLAUDE.md, PRD.md, PROGRESS.md, prd-and-task-conventions.md, 4 task files, testing-strategy.md, tui-mockups.html, 6 research documents)

---

## Summary

The Codex review spawned three parallel sub-agents plus a second-pass audit to cover all 10 review criteria. The review verified dependency versions against the Go module proxy (all versions confirmed to exist: Go 1.26.0, go-gh v2.13.0, Cobra v1.10.x, Bubble Tea v1.3.x, Lipgloss v1.1.0, golangci-lint v2.9.0). It found **3 CRITICAL issues**, **6 IMPORTANT issues**, and **5 MINOR issues** across the documentation suite.

---

## CRITICAL Issues

### C1. Go Module Path vs Repository Name Mismatch

- **SEVERITY:** CRITICAL
- **FILE(S):** `docs/tasks/000-repository-scaffold.md` (lines 31, 53, 63, 82, 88-89), `docs/PRD.md` (line 142)
- **ISSUE:** Task 1.1 instructs the agent to run `go mod init github.com/indrasvat/gh-ghent`, but the actual GitHub remote is `github.com/indrasvat/ghent` (confirmed via `git remote -v`). Initializing the module with a non-existent `gh-ghent` path will break `go mod tidy`, dependency resolution, and all import statements. The PRD directory tree (line 142) also uses `gh-ghent/` as the root directory name.
- **RECOMMENDATION:** Decide on the canonical module path. If the repo stays at `github.com/indrasvat/ghent`, update all `go mod init`, binary output paths, and directory references to use `github.com/indrasvat/ghent`. If the repo should be renamed to `gh-ghent` (matching the `gh` extension naming convention), rename the GitHub repository first. Either way, ensure a single source of truth for the module path across all docs and task files.

### C2. GraphQL Pagination Not Specified in Comments Command

- **SEVERITY:** CRITICAL
- **FILE(S):** `docs/PRD.md` (lines 332-346), `docs/github-api-research.md` (lines 24-148)
- **ISSUE:** The comments command specification only shows `reviewThreads(first: 100)` in the example query. Acceptance criterion FR-COM-09 requires handling PRs with 100+ threads, but the PRD never specifies that the GraphQL client must consume `pageInfo.hasNextPage`/`endCursor` and loop with `after` for pagination. Additionally, `isResolved` cannot be filtered server-side -- client-side filtering is required but not documented. An agent implementing this literally will miss threads beyond the first page and show resolved threads.
- **RECOMMENDATION:** Add explicit pagination requirements to PRD section 6.2: the GraphQL client must paginate using `pageInfo.hasNextPage`/`endCursor` with `after` (as documented in `docs/github-api-research.md` lines 112-147), and must filter `isResolved` client-side. Reference the research doc pagination examples directly.

### C3. Cross-Reference Section Numbers Are Systematically Wrong

- **SEVERITY:** CRITICAL
- **FILE(S):** `docs/prd-and-task-conventions.md` (lines 770-775), `docs/tasks/000-repository-scaffold.md` (lines 22-25, 50, 112), `docs/tasks/001-cobra-cli-skeleton.md` (lines 24, 108), `docs/prd-and-task-conventions.md` (lines 469-470, 694-699)
- **ISSUE:** The "Research docs" reference table and multiple task files point to wrong section numbers in nearly every research document. Agents following these references will read irrelevant content. Specific mismatches:
  - `go-project-patterns-research.md`: Task 000 cites section 3 for Makefile (actually section 5), section 4 for golangci-lint (actually section 9), section 6 for lefthook (actually section 8), section 8 for GoReleaser (actually section 10). Task 001 cites section 7 for Cobra (actually section 4).
  - `gh-extensions-support-research.md`: The conventions table references sections 5-6 for go-gh SDK (actually section 4), section 7 for Auth (actually section 6), section 9 for JSON output (actually section 10), section 13 for testing (section 13 does not exist).
  - `github-api-research.md`: Table references section 3 for review-thread queries (actually section 1), section 4 for resolveReviewThread (actually section 2), section 5 for check runs (actually section 6), section 6 for job logs (actually section 7 or 9).
  - `popular-extensions-research.md`: Table references section 1 for gh-dash (actually section 3), section 3 for Cobra patterns (actually section 14 Cross-Cutting Patterns), section 8 for testing (actually Pattern 8 inside section 14, not section 8 which is gh-workflow-stats).
  - `dorikin-patterns-research.md`: Table references section 3 for BubbleTea patterns (actually section 14) and section 6 for error handling (actually section 7).
- **RECOMMENDATION:** Audit every section reference against the actual Table of Contents in each research document and fix all pointers. The corrected mappings are provided above. Also fix the template examples in the conventions doc (lines 469-470) which cite `github-api-research.md` section 3 (should be section 1) and `gh-extensions-support-research.md` section 9 (should be section 4).

---

## IMPORTANT Issues

### I1. Testing Strategy References Undefined Flags (`--file`, `--dry-run`)

- **SEVERITY:** IMPORTANT
- **FILE(S):** `docs/testing-strategy.md` (lines 239-250), `docs/PRD.md` (lines 405-412)
- **ISSUE:** The `gh ghent resolve` testing checklist exercises `--file src/main.go` and `--dry-run` flags that are nowhere defined in the PRD (section 6.4 only lists `--thread`, `--all`, `--unresolve`), the phase plan, or any task file. Agents or testers following this documentation will attempt to call non-existent flags, causing Phase 2 verification to stall.
- **RECOMMENDATION:** Either extend PRD section 6.4 to define `--file` and `--dry-run` flags (and add corresponding acceptance criteria), or remove these flags from the testing checklist to keep verification aligned with implemented behavior.

### I2. `viewerCanResolve`/`viewerCanUnresolve` Permission Check Missing from PRD

- **SEVERITY:** IMPORTANT
- **FILE(S):** `docs/PRD.md` (lines 405-441), `docs/github-api-research.md` (lines 28-47, 156-167, 372-373)
- **ISSUE:** The resolve command spec describes multi-select, confirmation, and error handling (FR-RES-07 mentions "Requires write permission; clear error if missing"), but it never mentions respecting the `viewerCanResolve`/`viewerCanUnresolve` boolean fields that the GraphQL API returns for each thread. Without this detail, the TUI could offer resolve/unresolve buttons to users who will immediately receive permission errors.
- **RECOMMENDATION:** Document in PRD section 6.4 that the GraphQL thread fetch must capture `viewerCanResolve`/`viewerCanUnresolve` (as documented in `docs/github-api-research.md` lines 372-373) and that the TUI/CLI should either hide/disable action keys or surface a clear permission error before invoking the mutation.

### I3. TUI View Architecture Incomplete (5 views listed, 7 in mockups)

- **SEVERITY:** IMPORTANT
- **FILE(S):** `docs/PRD.md` (lines 226-234), `docs/tui-mockups.html` (7 tabs), `docs/PROGRESS.md` (lines 54-61)
- **ISSUE:** PRD section 5.3 lists only five views (`ViewComments`, `ViewChecks`, `ViewResolve`, `ViewSummary`, `ViewWatch`). The TUI mockups HTML file has seven tabs: comments list, comments expanded, checks, checks log viewer, resolve, summary, and pipe mode. The "expanded thread view" and "pipe mode view" are absent from the architecture section. Phase 5 task planning also describes more views than the architecture accounts for.
- **RECOMMENDATION:** Update PRD section 5.3 to enumerate all views from the mockups: (1) comments list, (2) comments expanded thread, (3) checks list, (4) checks log viewer, (5) resolve multi-select, (6) summary dashboard, (7) pipe mode/non-TTY output. Map each to its file/handler so the architecture is a complete spec.

### I4. Check Run Status Enumeration Includes Invalid Values

- **SEVERITY:** IMPORTANT
- **FILE(S):** `docs/github-api-research.md` (line 543)
- **ISSUE:** Line 543 lists check run status values as `queued`, `in_progress`, `completed`, `waiting`, `requested`, `pending`. According to the GitHub REST API, a Check Run only returns `queued`, `in_progress`, or `completed` (confirmed by the same doc's line 433). The values `waiting`, `requested`, and `pending` exist on check suites or workflow runs, not on check runs. Feeding nonexistent statuses into the status-aggregation logic will lead to unhandled cases and incorrect exit codes.
- **RECOMMENDATION:** Correct line 543 to list only the three valid check run statuses (`queued`, `in_progress`, `completed`). If workflow run statuses are also needed, document them in a separate subsection to avoid confusion.

### I5. Missing `scripts/` Directory Referenced in Testing Strategy

- **SEVERITY:** IMPORTANT
- **FILE(S):** `docs/testing-strategy.md` (lines 1162-1164, 1203), `docs/PROGRESS.md`
- **ISSUE:** The testing strategy's directory layout and Makefile targets reference `scripts/test-binary.sh` (L3 tests) and `scripts/test-agent-workflow.sh` (L5 tests). The repository currently has no `scripts/` directory, so these Makefile targets are unexecutable. CLAUDE.md also references these indirectly through the testing levels.
- **RECOMMENDATION:** Either add stub scripts now (so the Makefile targets work from the start) or explicitly mark these steps as "to be created in Phase 3" in the testing strategy so agents reading the docs understand why the dependency is missing and don't attempt to run them.

### I6. `prd-and-task-conventions.md` Phase 2 Lists Non-Existent Task 2.6

- **SEVERITY:** IMPORTANT
- **FILE(S):** `docs/prd-and-task-conventions.md` (line 651)
- **ISSUE:** The Phase 2 task list in the conventions doc includes "Task 2.6: --watch mode." However, the PRD and PROGRESS.md both place watch mode in Phase 3 (Task 3.1). No Task 2.6 exists in the actual task numbering. This inconsistency could confuse agents about task sequencing.
- **RECOMMENDATION:** Remove "Task 2.6: --watch mode" from the Phase 2 section of the conventions doc, or add a note clarifying this is now Task 3.1 in Phase 3. Ensure the conventions doc's phase breakdown mirrors the PRD exactly.

---

## MINOR Issues

### M1. Lipgloss Version Ceiling

- **SEVERITY:** MINOR
- **FILE(S):** `CLAUDE.md` (line 11), `docs/PRD.md` (line 122), `docs/tasks/000-repository-scaffold.md` (line 17)
- **ISSUE:** The project specifies "Lipgloss v1.1+" as the minimum version. However, `go list -m -versions` shows the latest published Lipgloss version is v1.1.0 (there is no v1.2+). This is technically correct but leaves no room for version pinning above the minimum. If Lipgloss releases v2.0 with breaking changes, "v1.1+" would pull it in.
- **RECOMMENDATION:** Consider pinning to `v1.1.x` or specifying `>= v1.1.0, < v2.0.0` in documentation to avoid future breaking-change surprises.

### M2. Task Template Structural Compliance

- **SEVERITY:** MINOR
- **FILE(S):** `docs/tasks/000-003`
- **ISSUE:** All four Phase 1 task files correctly follow the prescribed template from `docs/prd-and-task-conventions.md` (status, depends/parallel, PRD/research references, files to create/modify, execution steps, verification, completion criteria, commit message, session protocol). No structural deviations were found. However, the template example in the conventions doc (lines 469-470) uses incorrect section numbers in its examples, which could propagate errors to future tasks.
- **RECOMMENDATION:** Fix the template examples in the conventions doc (covered by C3 above). The task files themselves are structurally sound.

### M3. PRD Table of Contents Missing Section 8

- **SEVERITY:** MINOR
- **FILE(S):** `docs/PRD.md` (lines 12-22)
- **ISSUE:** The PRD Table of Contents lists sections 1-7, but section 8 (Implementation Phases) exists at line 576. It is not listed in the ToC.
- **RECOMMENDATION:** Add `8. [Implementation Phases](#8-implementation-phases)` to the Table of Contents.

### M4. Conventions Doc Contains Stale Example Phase Listing

- **SEVERITY:** MINOR
- **FILE(S):** `docs/prd-and-task-conventions.md` (lines 640-655)
- **ISSUE:** The "Core MVP" example in the conventions doc (under "Phase 2: Core MVP") uses different task groupings and numbering than the actual PRD. It includes a "Phase 3: Polish & Harden" section that doesn't match the PRD's Phase 3 structure. This is in the "example" section but could still mislead.
- **RECOMMENDATION:** Update the example phase listing to match the actual PRD phase structure, or add a clear disclaimer that the example uses a different project's phases for illustration.

### M5. `prd-and-task-conventions.md` Line 306 Contradicts PRD

- **SEVERITY:** MINOR
- **FILE(S):** `docs/prd-and-task-conventions.md` (line 306)
- **ISSUE:** The example PRD snippet in the conventions doc says "Not a full TUI (CLI output, not BubbleTea)" under "What ghent Is NOT." However, ghent explicitly IS a full Bubble Tea TUI -- the real PRD section 1.3 says "Not a PR review tool" and "Not a replacement for `gh pr`" but never says it's not a TUI.
- **RECOMMENDATION:** Fix the example snippet to match the actual PRD's "What ghent Is NOT" entries.

---

## Verification Results (No Issues Found)

The following areas were checked and found to be correct:

### Dependency Correctness (Criterion 1)
- Task 1.4 correctly depends on 1.1 + 1.3. Confirmed.
- Tasks 1.2 and 1.3 both depend on 1.1 and are correctly marked as parallelizable. Confirmed.
- Phase 2 dependencies on Phase 1 are correctly structured in both PRD and PROGRESS.md.

### Phase Ordering (Criterion 7)
- CLI-first, TUI-second strategy is correctly reflected across PRD section 8, PROGRESS.md, and the conventions doc.
- Phase 1 (Walking Skeleton) -> Phase 2 (CLI Commands) -> Phase 3 (CLI Polish) -> Phase 4 (TUI Foundation) -> Phase 5 (TUI Views) ordering is consistent.
- No tasks are misordered within phases.

### Technical Accuracy - Dependency Versions (Criterion 6)
- All specified versions verified against `go list -m -versions`:
  - Go 1.26.0: Confirmed (`go version go1.26.0 darwin/arm64`)
  - go-gh v2.13.0: Confirmed (latest on module proxy)
  - Cobra v1.10+: v1.10.0, v1.10.1, v1.10.2 exist. Confirmed.
  - Bubble Tea v1.3+: v1.3.0 through v1.3.10 exist. Confirmed.
  - Lipgloss v1.1+: v1.1.0 exists. Confirmed.
  - golangci-lint v2.9.0: Confirmed (v2.10.0, v2.10.1 also available).

### Task Template Compliance (Criterion 9)
- All four task files follow the prescribed template structure. Confirmed.
- PRD follows the conventions doc's recommended structure (version table, ToC, numbered sections, acceptance criteria checklists, phase roadmap). Minor ToC omission noted above (M3).

### Testing Strategy Alignment (Criterion 10)
- L1 + L3 verification levels for Phase 1 are correctly specified in both testing-strategy.md and individual task verification sections.
- The cardinal rule (binary execution required) is consistently emphasized.
- Testing levels are correctly assigned to phases (L1-L3 for Phase 1-2, L4-L5 for Phase 4-5).

---

## Codex Agent Execution Details

The review was conducted by OpenAI Codex v0.104.0 using model gpt-5.3-codex with three parallel sub-agents:

1. **Agent 1 (Documentation Quality):** Reviewed task dependencies, template compliance, and testing alignment. Completed successfully.
2. **Agent 2 (PRD vs Research/Mockups):** Audited PRD consistency against research documents and TUI mockups. Identified pagination and permission-check gaps. Completed successfully.
3. **Agent 3 (Cross-Reference Integrity):** Verified all section references across all documents. Identified systematic section-number mismatches. Completed with timeout (partial results used).
4. **Second-Pass Agent:** Comprehensive sweep for module path, TUI view drift, and check-run status issues. Completed successfully.

Total unique files read: 16. Total section references verified: 40+. Total dependency versions checked against module proxy: 6.
