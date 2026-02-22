# Gemini Review of ghent Project Documentation

> **Reviewer:** Google Gemini CLI (gemini --yolo mode)
> **Date:** 2026-02-22
> **Method:** Sequential file-by-file review with cross-referencing against Codex review
> **Files Reviewed:** All 17 listed files (CLAUDE.md, PRD.md, PROGRESS.md, prd-and-task-conventions.md, 4 task files, testing-strategy.md, tui-mockups.html, 6 research documents, codex-review.md)

---

## Summary

Gemini conducted a thorough review of all 17 project files, cross-referencing the PRD against research documents, TUI mockups, task files, and the conventions guide. The review also verified whether issues identified in the previous Codex review had been addressed. The review found **1 CRITICAL issue**, **2 IMPORTANT issues**, and **3 MINOR issues**. Most of the Codex-identified issues have been successfully fixed, with two pending items carried forward.

---

## Codex Review Verification

Gemini verified each issue from `docs/reviews/codex-review.md` against the current state of the files:

| Codex Issue | Status | Verification Note |
| :--- | :--- | :--- |
| **C1: Module Path Mismatch** | **FIXED** | `docs/tasks/000-repository-scaffold.md` and `PRD.md` now use `github.com/indrasvat/ghent`. |
| **C2: GraphQL Pagination** | **FIXED** | PRD section 6.2 explicitly requires `pageInfo.hasNextPage` and client-side filtering. |
| **C3: Cross-Ref Section Errors** | **FIXED** | All section pointers in Tasks 000-003 and the conventions doc have been audited and corrected. |
| **I1: Undefined Flags (`--file`, `--dry-run`)** | **FIXED** | `testing-strategy.md` no longer references `--file` or `--dry-run` for the resolve command. |
| **I2: Permission Checks Missing** | **FIXED** | PRD section 6.4 now includes `viewerCanResolve`/`viewerCanUnresolve` requirements. |
| **I3: TUI View Drift (5 vs 7)** | **FIXED** | PRD section 5.3 now enumerates all 7 views present in `tui-mockups.html`. |
| **I4: Check Run Status Values** | **FIXED** | `docs/github-api-research.md` now correctly differentiates check run vs suite statuses. |
| **I5: Missing `scripts/` Directory** | **PENDING** | `scripts/` is referenced in PRD directory tree but not created in any Phase 1 task. See Issue 3 below. |
| **I6: Task 2.6 Ghost Reference** | **FIXED** | Removed from `docs/prd-and-task-conventions.md`. |
| **M1: Lipgloss Version Ceiling** | **PENDING** | PRD still uses `v1.1+` which could pull in breaking v2.0 changes. See Issue 6 below. |
| **M2: Template Compliance** | **FIXED** | Template examples corrected alongside C3 fixes. |
| **M3: PRD ToC Missing Section 8** | **FIXED** | PRD Table of Contents is now complete with all 12 sections. |
| **M4: Stale Example Phases** | **FIXED** | Phase 3 in conventions doc now matches PROGRESS.md. |
| **M5: "What ghent Is NOT" Contradiction** | **FIXED** | Corrected to match actual PRD entries. |

**Summary: 11 of 14 Codex issues confirmed FIXED. 2 remain PENDING (I5, M1). 1 issue (M2) confirmed no structural deviations in task files.**

---

## New Issues (Fresh Perspective)

### Issue 1: Extension Repository Naming Convention

- **SEVERITY:** CRITICAL
- **FILE(S):** `docs/tasks/000-repository-scaffold.md`, `README.md`, `docs/PRD.md`
- **ISSUE:** The current repository is named `ghent`, but GitHub CLI extension naming conventions (and the project's own research in `docs/gh-extensions-support-research.md` section 1) state repositories MUST have the `gh-` prefix (e.g., `gh-ghent`). While `gh extension install indrasvat/ghent` will technically work, this diverges from the ecosystem standard and may cause issues with automated discovery or future `gh` CLI changes. The module path was already corrected to `github.com/indrasvat/ghent` (fixing Codex C1), but this does not address the extension naming convention itself.
- **RECOMMENDATION:** Either rename the GitHub repository to `gh-ghent` (preferred for ecosystem compatibility), or explicitly document in PRD section 1 why the `gh-` prefix convention is being intentionally ignored and the tradeoffs of doing so. Ensure the module path remains consistent with whichever name is chosen.

### Issue 2: Release Workflow Redundancy (GoReleaser vs gh-extension-precompile)

- **SEVERITY:** IMPORTANT
- **FILE(S):** `docs/tasks/000-repository-scaffold.md`, `docs/PRD.md` sections 4 and 8
- **ISSUE:** Phase 1 (Task 1.1) specifies creating both `.goreleaser.yml` and a release workflow using `gh-extension-precompile`. These tools perform overlapping functions -- both handle cross-compilation of Go binaries. `gh-extension-precompile` is specifically designed for `gh` extensions and follows their unique binary naming schema automatically (e.g., `gh-ghent-darwin-arm64`), while GoReleaser offers more general-purpose control. Using both without a clear separation of responsibilities (e.g., GoReleaser for changelogs only, precompile for binary distribution) will cause CI conflicts or produce redundant artifacts.
- **RECOMMENDATION:** Standardize on one tool for binary distribution. Given this is a `gh` extension, `gh-extension-precompile` is the preferred choice for binary distribution. Use GoReleaser only if its specific changelog grouping or non-binary artifact features are needed, and if so, disable its binary upload step to avoid clashing with the precompile action. Document the decision rationale in the task file.

### Issue 3: Missing `scripts/` and `testdata/` Creation in Phase 1

- **SEVERITY:** IMPORTANT
- **FILE(S):** `docs/tasks/000-repository-scaffold.md`, `docs/tasks/002-domain-types.md`, `docs/testing-strategy.md`
- **ISSUE:** The testing strategy and PRD reference `scripts/test-binary.sh` (L3 tests) and `testdata/*.json` fixtures. Phase 2 tasks depend on these for verification, but none of the Phase 1 tasks (000-003) schedule their creation. An agent starting Phase 2 will find the L3/L5 verification steps unexecutable. This is a carryover from Codex I5 which remains unresolved.
- **RECOMMENDATION:** Add a step to Task 1.1 (000-repository-scaffold.md) to create the `scripts/` directory with a minimal stub `test-binary.sh` that runs `ghent --version` and validates exit code 0. Add a step to Task 1.3 (002-domain-types.md) to create initial JSON fixtures in `testdata/` (e.g., `review_threads_response.json`, `check_runs_response.json`) since the domain types should be modeled after real API response shapes.

### Issue 4: Multi-View `WindowSizeMsg` Propagation Pitfall

- **SEVERITY:** MINOR
- **FILE(S):** `docs/PRD.md` section 5.3, `CLAUDE.md` (TUI Pitfalls section)
- **ISSUE:** The TUI architecture defines 7 distinct views. A common Bubble Tea pitfall -- not currently listed in CLAUDE.md or testing-strategy.md -- is failing to propagate `tea.WindowSizeMsg` to ALL sub-models, not just the active one. If an inactive view is initialized while backgrounded and then switched to, its layout will be garbled until the next terminal resize event. This is especially problematic for the expanded thread view and log viewer which have viewport components with fixed dimensions.
- **RECOMMENDATION:** Add a bullet point to the "TUI Pitfalls" section in `CLAUDE.md`: "Always propagate `WindowSizeMsg` to all sub-models (active and inactive) to ensure layout integrity on view switch." Also note this requirement in `testing-strategy.md` under TUI-related test guidance.

### Issue 5: Task Filename vs Task ID Numbering Discrepancy

- **SEVERITY:** MINOR
- **FILE(S):** `docs/tasks/000-repository-scaffold.md`, `docs/tasks/001-cobra-cli-skeleton.md`, `docs/tasks/002-domain-types.md`, `docs/tasks/003-github-api-client.md`
- **ISSUE:** There is a mismatch between task filenames (zero-indexed: 000, 001, 002, 003) and task IDs (phase-indexed: 1.1, 1.2, 1.3, 1.4). While internally consistent, this creates a small mental burden when cross-referencing. For example, `000-repository-scaffold.md` contains "Task 1.1" -- the filename prefix `000` does not correspond to either the task number `1` or the phase-task pair `1.1`.
- **RECOMMENDATION:** Consider aligning filenames with task IDs (e.g., `011-repository-scaffold.md` for Task 1.1, `012-cobra-cli-skeleton.md` for Task 1.2) or strictly use only the zero-indexed `NNN` scheme in all cross-references to avoid confusion. If keeping the current scheme, add a note to the conventions doc mapping file indices to task IDs.

### Issue 6: Lipgloss Version Pinning (Carryover from Codex M1)

- **SEVERITY:** MINOR
- **FILE(S):** `CLAUDE.md`, `docs/PRD.md` section 4
- **ISSUE:** The project specifies "Lipgloss v1.1+" as the minimum version. As noted by Codex (M1), v1.1.0 is the latest published version. The open-ended `v1.1+` constraint leaves the project vulnerable to breaking changes if Lipgloss releases v2.0.
- **RECOMMENDATION:** Update version specification across all docs to `Lipgloss >= v1.1.0, < v2.0.0` (or equivalently `v1.1.x`) to allow minor updates while preventing breaking major version changes.

---

## Areas Verified Without Issues

The following areas were checked and found to be correct:

### Dependency Correctness (Criterion 1)
- Task 1.4 correctly depends on Tasks 1.1 and 1.3. Confirmed.
- Tasks 1.2 and 1.3 both depend on Task 1.1 and are correctly marked as parallelizable. Confirmed.
- Phase 2 dependencies on Phase 1 completion are correctly structured in both PRD and PROGRESS.md. Confirmed.

### Phase Ordering (Criterion 7)
- CLI-first, TUI-second strategy is correctly reflected across PRD section 8, PROGRESS.md, and the conventions doc.
- Phase 1 (Walking Skeleton) -> Phase 2 (CLI Commands) -> Phase 3 (CLI Polish) -> Phase 4 (TUI Foundation) -> Phase 5 (TUI Views) ordering is consistent throughout.
- No tasks are misordered within any phase.

### PRD vs Mockups Consistency (Criterion 4)
- PRD section 5.3 now enumerates all 7 views: (1) comments list, (2) comments expanded thread, (3) checks list, (4) checks log viewer, (5) resolve multi-select, (6) summary dashboard, (7) pipe mode/non-TTY output. Confirmed consistent with `tui-mockups.html` tabs.
- Component choices (bubbles/list, bubbles/viewport, etc.) are consistent between PRD and mockups. Confirmed.

### PRD vs Research Consistency (Criterion 3)
- GraphQL pagination requirements now match research findings. Confirmed.
- API endpoints and methods are accurately reflected. Confirmed.
- Permission fields (`viewerCanResolve`/`viewerCanUnresolve`) are now documented. Confirmed.
- Check run status enumerations are now correctly differentiated from suite statuses. Confirmed.

### Task Template Compliance (Criterion 9)
- All four Phase 1 task files follow the prescribed template structure from `prd-and-task-conventions.md`. Confirmed.
- PRD follows the conventions doc's recommended structure (version table, ToC, numbered sections, acceptance criteria checklists, phase roadmap). Confirmed.

### Testing Strategy Alignment (Criterion 10)
- L1 + L3 verification levels for Phase 1 are correctly specified in both testing-strategy.md and individual task verification sections. Confirmed.
- The cardinal rule (binary execution required) is consistently emphasized. Confirmed.
- Testing levels are correctly assigned to phases (L1-L3 for Phase 1-2, L4-L5 for Phase 4-5). Confirmed.

### Cross-Reference Integrity (Criterion 8)
- All section references in task files now point to sections that actually exist in the corresponding research docs. Confirmed (all C3 fixes verified).

---

## Comparison: Gemini vs Codex Findings

| Area | Codex Found | Gemini Found | Overlap |
| :--- | :--- | :--- | :--- |
| Module path mismatch | C1 | Confirmed FIXED | Full overlap |
| GraphQL pagination | C2 | Confirmed FIXED | Full overlap |
| Cross-ref errors | C3 | Confirmed FIXED | Full overlap |
| Extension naming convention (`gh-` prefix) | Not found | **Issue 1 (CRITICAL)** | New finding |
| Release workflow redundancy | Not found | **Issue 2 (IMPORTANT)** | New finding |
| Missing `scripts/` directory | I5 | **Issue 3 (IMPORTANT)** -- still pending | Carryover |
| `WindowSizeMsg` propagation | Not found | **Issue 4 (MINOR)** | New finding |
| Task filename vs ID numbering | Not found | **Issue 5 (MINOR)** | New finding |
| Lipgloss version pinning | M1 | **Issue 6 (MINOR)** -- still pending | Carryover |
| Undefined flags | I1 | Confirmed FIXED | Full overlap |
| Permission checks | I2 | Confirmed FIXED | Full overlap |
| TUI view drift | I3 | Confirmed FIXED | Full overlap |
| Check run statuses | I4 | Confirmed FIXED | Full overlap |
| Task 2.6 ghost | I6 | Confirmed FIXED | Full overlap |
| PRD ToC, stale examples, "What ghent Is NOT" | M3-M5 | Confirmed FIXED | Full overlap |

**Net new findings by Gemini: 4 (Issues 1, 2, 4, 5)**
**Carried forward from Codex: 2 (Issues 3, 6)**

---

## Gemini Review Execution Details

The review was conducted by Google Gemini CLI in YOLO mode with sequential file reads and cross-referencing. The agent:

1. Read the Codex review first to establish a baseline of known issues
2. Read all core documentation (CLAUDE.md, PRD.md, testing-strategy.md)
3. Read all four Phase 1 task files and verified against the conventions template
4. Read all six research documents and cross-referenced section pointers
5. Read the TUI mockups HTML and verified view enumeration against PRD
6. Verified the PROGRESS.md phase structure against conventions doc examples
7. Checked PRD Table of Contents completeness
8. Produced a final assessment with severity-ranked findings

Total unique files read: 17. All Codex issues verified. 4 new issues identified.
