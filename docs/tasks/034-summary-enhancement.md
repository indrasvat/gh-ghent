# Task 034: Summary Enhancement (--logs, --watch, --quiet)

| Field | Value |
|-------|-------|
| **Status** | DONE |
| **Phase** | 10 (Summary Enhancement) |
| **Priority** | P1 |
| **Depends on** | 009 (summary), 006 (checks --logs), 010 (watch mode) |

## Objective

Enhance `gh ghent summary` to be a complete single-command entry point for PR monitoring:
- `--logs`: Include failing job log excerpts and annotations in summary output
- `--watch`: Poll CI until complete, then output full summary
- `--quiet`: Silent exit on merge-ready, full output on not-ready
- Enrich all formatters (markdown, json, xml) with failure details

## Implementation

### summary.go
- Added `--logs`, `--watch`, `--quiet` flag variables
- `--watch` TTY path: launches ViewWatch TUI (reuses checks.go pattern)
- `--watch` non-TTY: WatchChecks to stderr, then falls through to full summary fetch
- `--logs` (or `--watch` implies): log-fetch loop for failing checks via FetchJobLog + ExtractErrorLines
- `--quiet`: if merge-ready, return nil (exit 0, no output); else fall through to full output

### Formatter enrichment
- **markdown.go FormatSummary**: Added unresolved thread details (path:line, author, preview) and failed check sections (### FAIL: name, annotations, log excerpt in fenced block)
- **markdown.go FormatCompactSummary**: Added failed check digest line after thread table
- **json.go FormatCompactSummary**: Added `failed_checks` field with name, annotations, log_excerpt
- **xml.go FormatSummary**: Added Threads to xmlSummaryComments, FailedChecks to xmlSummaryChecks
- **xml.go FormatCompactSummary**: Added FailedChecks to xmlCompactSummary

### Documentation
- README.md: Updated summary flags table and agent workflow example
- SKILL.md: Restructured with summary-first approach, new description with trigger phrases
- command-reference.md: Updated summary section with new flags, watch/quiet mode docs, enriched schemas
- agent-workflows.md: Updated CI monitor workflow to use summary --watch --logs
- ci-monitor.md: Updated walkthrough with summary --watch --logs approach
- feature-showcase-hero.html: Replaced checks --verbose with summary --logs, added summary --watch and --quiet rows

## Tests Added

- `TestJSONSummaryWithFailedChecks` — full JSON summary contains annotations + log_excerpt on failing checks
- `TestJSONCompactSummaryFailedChecks` — compact JSON has failed_checks array
- `TestJSONCompactSummaryNoFailedChecksWhenAllPass` — failed_checks absent when all pass
- `TestMarkdownSummaryWithFailedChecks` — FAIL header, annotations, log excerpt in markdown
- `TestMarkdownSummaryNoFailedChecksWhenAllPass` — no FAIL sections when all pass
- `TestMarkdownCompactSummaryWithFailedChecks` — compact markdown includes FAIL: line
- `TestXMLSummaryWithFailedChecks` — XML has FailedChecks + Threads in summary
- `TestXMLSummaryNoFailedChecksWhenAllPass` — no FailedChecks when all pass
- `TestXMLCompactSummaryWithFailedChecks` — compact XML has FailedChecks

## Verification

### L1
```bash
make ci-fast  # 560 tests, all pass, race detector clean
make lint     # 0 issues
```

### L3
```bash
gh ghent summary -R indrasvat/peek-it --pr 2 --logs --format json | jq '.checks.checks[] | select(.conclusion=="failure") | .log_excerpt'
gh ghent summary -R indrasvat/doot --pr 1 --quiet; echo "exit: $?"
gh ghent summary -R indrasvat/tbgs --pr 1 --format json | jq '.comments.unresolved_count'
```
