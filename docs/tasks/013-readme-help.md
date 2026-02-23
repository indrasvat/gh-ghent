# Task 3.4: README + Help Text

## Status: TODO

## Depends On
- Phase 2 complete (needs all commands finalized for accurate docs)

## Parallelizable With
- Task 3.3: Extension packaging (partially)

## Problem

ghent needs a user-facing README with installation instructions, usage examples for all commands, and polished `--help` text on every subcommand. This is the first thing users and agents see.

## PRD Reference

- §6.1-6.6 — All command signatures and flag descriptions
- §1.3 (What ghent Is NOT) — Anti-goals for README
- §3 (Target Audience) — Developer + agent audience

## Research References

- `docs/popular-extensions-research.md` §14 — Cross-cutting best practices (README patterns)
- `docs/gh-extensions-support-research.md` §2 — Extension installation instructions

## Files to Create

- `.claude/automations/test_ghent_help.py` — iterm2-driver L4 visual test for help output (per `docs/testing-strategy.md` §8)
- `.claude/automations/test_ghent_agent.py` — iterm2-driver L4/L5 visual test for agent workflow (per `docs/testing-strategy.md` §8)

## Files to Modify

- `README.md` — Full content: install, usage, examples, agent integration
- `internal/cli/root.go` — Polish root help with example section
- `internal/cli/comments.go` — Polish help text, add Example field
- `internal/cli/checks.go` — Polish help text, add Example field
- `internal/cli/resolve.go` — Polish help text, add Example field
- `internal/cli/reply.go` — Polish help text, add Example field
- `internal/cli/summary.go` — Polish help text, add Example field

## Execution Steps

### Step 1: Polish Cobra help text
- Each command: Short, Long, Example fields
- Example shows both human and agent usage
- Consistent flag descriptions across all commands

### Step 2: Write README
- Installation: `gh extension install indrasvat/gh-ghent`
- Quick start: 3 most useful commands
- Full command reference (all 5 commands with flags)
- Agent integration section (JSON output, exit codes, piping)
- Format examples (json, xml, md side by side)

### Step 3: Verify help output
- `gh ghent --help` reads well
- Each subcommand `--help` is self-sufficient

## Verification

### L3: Binary Execution
```bash
make build
./bin/gh-ghent --help
./bin/gh-ghent comments --help
./bin/gh-ghent checks --help
./bin/gh-ghent resolve --help
./bin/gh-ghent reply --help
./bin/gh-ghent summary --help
```

### L4: Visual (iterm2-driver)
Create `.claude/automations/test_ghent_help.py` and `.claude/automations/test_ghent_agent.py` following canonical template in `docs/testing-strategy.md` §5:
```bash
uv run .claude/automations/test_ghent_help.py
uv run .claude/automations/test_ghent_agent.py
```
**test_ghent_help.py**: Verify all `--help` outputs are readable, list correct flags, show examples
- Screenshots: `ghent_help_root.png`, `ghent_help_comments.png`, `ghent_help_version.png`
**test_ghent_agent.py**: Verify agent workflow end-to-end (per `docs/testing-strategy.md` §6)
- Valid JSON from `--format json`, meaningful exit codes, no ANSI in piped output, <2s response, actionable error messages

## Completion Criteria

1. README has install, usage, examples, agent section
2. All commands have Short, Long, Example in Cobra
3. Help text is consistent and self-sufficient
4. `make ci` passes
5. PROGRESS.md updated

## Commit

```
docs: add README and polish --help text for all commands

- README with install, usage, agent integration sections
- Cobra Example field on all 5 subcommands
- Consistent flag descriptions
```

## Session Protocol

1. Read CLAUDE.md
2. Read this task file
3. **Change this task's status to `IN PROGRESS`**
4. Read PRD §6.1-6.6
5. Execute steps 1-3
6. Run verification (L3 → L4)
7. **Change this task's status to `DONE`**
8. Update `docs/PROGRESS.md`
9. Update CLAUDE.md Learnings if needed
10. Commit
