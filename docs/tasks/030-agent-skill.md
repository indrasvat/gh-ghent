# Task 7.1: Agent Skill for gh-ghent

- **Phase:** 7 (Distribution & Ecosystem)
- **Status:** DONE
- **Depends on:** None (all 30 feature tasks complete)
- **Blocks:** None
- **L4 Visual:** Not required (no TUI changes — documentation/skill only)

## Problem

ghent is feature-complete but invisible to the thousands of AI coding agents that would benefit from it. When an agent encounters a PR review cycle — unresolved threads, failing CI, needed approvals — it has no way to discover or learn gh-ghent unless the user manually pastes documentation into context.

The [Agent Skills](https://agentskills.io) open standard solves this. A skill is a `SKILL.md` file (with optional supporting files) that teaches coding agents how to use a tool. It works across 35+ agents: Claude Code, Codex, Cursor, Cline, GitHub Copilot, Amp, and more.

**Goal:** Create a best-in-class skill that makes gh-ghent instantly usable by any AI coding agent, installable with `npx skills add indrasvat/gh-ghent`.

## Research References

- [Agent Skills docs](https://agentskills.io) — open standard specification
- [Claude Code skills docs](https://code.claude.com/docs/en/skills) — Claude-specific features (frontmatter, context: fork, allowed-tools)
- [vercel-labs/skills](https://github.com/vercel-labs/skills) — reference implementation + CLI
- `~/.claude/skills/bubbletea/` — example of a well-structured skill with `references/` subdirectory
- `~/.agents/skills/shadcn-ui/` — example of a skill with `examples/`, `resources/`, `scripts/`

## Skill Design

### Discovery Path

The skills CLI scans these locations for `SKILL.md`:
1. Repository root
2. `skills/` directory (standard convention)
3. Agent-specific directories (`.claude/skills/`, `.agents/skills/`, etc.)

We place the skill in a top-level `skill/` directory for maximum discoverability by `npx skills add indrasvat/gh-ghent`.

### Directory Structure

```
skill/
├── SKILL.md                    # Main skill file (< 500 lines, per best practices)
├── references/
│   ├── command-reference.md    # Complete command + flag reference
│   ├── agent-workflows.md      # Step-by-step agent patterns
│   └── exit-codes.md           # Exit code semantics for branching
└── examples/
    ├── review-cycle.md         # Full review-fix-resolve workflow
    └── ci-monitor.md           # CI watch + error extraction workflow
```

### SKILL.md Content Strategy

The main `SKILL.md` must be concise (< 500 lines) and focus on:

1. **When to use** — triggering conditions (PR review, CI failure, thread resolution)
2. **Installation** — `gh extension install indrasvat/gh-ghent`
3. **Quick start** — the 3 most useful commands for an agent
4. **Dual-mode** — pipe mode (`--no-tui --format json`) is what agents want
5. **Core patterns** — check → fix → resolve → reply cycle
6. **Exit codes** — how to branch logic without parsing output
7. **Incremental monitoring** — `--since` for change-only context
8. **References** — links to supporting files for deep dives

### Frontmatter Design

```yaml
---
name: gh-ghent
description: >
  Monitor and act on GitHub PR reviews with gh-ghent. Use when working with
  pull requests that have review comments, CI checks, or need thread resolution.
  Provides structured JSON/XML/Markdown output optimized for coding agents.
---
```

Key decisions:
- `disable-model-invocation: false` (default) — agents should auto-discover this
- `user-invocable: true` (default) — `/gh-ghent` as manual fallback
- No `allowed-tools` restriction — skill is pure knowledge, no tool access needed
- No `context: fork` — instructions should run inline with agent context

### Supporting Files

**`references/command-reference.md`** — Complete reference for all 5 commands:
- `comments` — flags, output schema, grouping
- `checks` — flags, annotations, log excerpts, watch mode
- `resolve` — single/batch/filter/dry-run/unresolve
- `reply` — body/body-file/stdin, thread validation
- `summary` — parallel fetch, merge readiness, compact mode
- Global flags: `--repo`, `--format`, `--pr`, `--since`, `--no-tui`, `--verbose`, `--debug`

**`references/agent-workflows.md`** — Opinionated, step-by-step patterns:
- "Fix all review comments" workflow
- "Monitor CI until green" workflow
- "Full PR review cycle" workflow (comments → fix → checks → resolve → reply)
- "Incremental delta check" workflow (--since for polling loops)
- Error handling patterns (auth, rate limit, not found)

**`references/exit-codes.md`** — Machine-readable exit code table:
- Per-command exit codes with semantic meaning
- How to use exit codes for branching (`if gh ghent summary ...; then merge; fi`)
- Error exit codes (2 = auth/rate/notfound)

**`examples/review-cycle.md`** — Annotated walkthrough:
- Fetch unresolved threads → parse JSON → fix code → resolve threads → reply
- Real JSON output examples from test repos
- Thread ID extraction patterns

**`examples/ci-monitor.md`** — Annotated walkthrough:
- Check CI status → extract failing annotations → fix → re-check
- Watch mode for agent polling loops
- Log error extraction patterns

## Files to Create

- `skill/SKILL.md` — Main skill file (< 500 lines)
- `skill/references/command-reference.md` — Complete command reference
- `skill/references/agent-workflows.md` — Step-by-step agent patterns
- `skill/references/exit-codes.md` — Exit code semantics
- `skill/examples/review-cycle.md` — Review fix workflow example
- `skill/examples/ci-monitor.md` — CI monitor workflow example

## Files to Modify

- `README.md` — Add "Agent Skill" section with installation instructions

## Execution Steps

### Step 1: Read context
1. Read CLAUDE.md (full project context, all commands, flags, architecture)
2. Read `docs/PRD.md` §5.2 (dual-mode operation), §6 (all commands)
3. Read `internal/cli/root.go` (global flags)
4. Read `internal/cli/comments.go`, `checks.go`, `resolve.go`, `reply.go`, `summary.go` (all command flags)
5. Read `internal/domain/types.go` (output schemas)
6. Read `internal/cli/errors.go` (exit codes)

### Step 2: Create directory structure
```bash
mkdir -p skill/references skill/examples
```

### Step 3: Write SKILL.md
- Frontmatter with name, description
- "When to Use" section with triggering conditions
- "Quick Start" with the 3 most important commands
- "Agent Mode" explaining --no-tui --format json
- "Core Workflow" — the check/fix/resolve/reply cycle
- "Exit Codes" summary table
- "Incremental Monitoring" with --since
- "References" pointing to supporting files
- Keep under 500 lines

### Step 4: Write references/command-reference.md
- Every command with all flags, types, defaults
- Output schema descriptions (JSON field names, types)
- Flag interactions and mutual exclusivity rules
- Per-command examples

### Step 5: Write references/agent-workflows.md
- 5 opinionated workflows with numbered steps
- Each step shows the exact command to run
- JSON output parsing guidance (jq examples)
- Error handling at each step

### Step 6: Write references/exit-codes.md
- Table: command x exit code x meaning
- Bash conditional examples
- Agent branching patterns

### Step 7: Write examples/review-cycle.md
- Full annotated walkthrough
- Real JSON output snippets
- Thread ID extraction
- resolve + reply sequence

### Step 8: Write examples/ci-monitor.md
- CI status check flow
- Annotation extraction
- Watch mode usage
- Log error parsing

### Step 9: Update README.md
- Add "For AI Agents" or "Agent Skill" section
- Installation: `npx skills add indrasvat/gh-ghent`
- Brief description of what the skill teaches

### Step 10: Test skill installation
```bash
# Verify skill is discoverable
npx skills add ./skill --dry-run  # or equivalent local test

# Verify SKILL.md parses correctly
cat skill/SKILL.md | head -20  # Check frontmatter

# Verify all referenced files exist
ls -la skill/references/ skill/examples/
```

## Verification

### Structural
```bash
# All files exist
test -f skill/SKILL.md
test -f skill/references/command-reference.md
test -f skill/references/agent-workflows.md
test -f skill/references/exit-codes.md
test -f skill/examples/review-cycle.md
test -f skill/examples/ci-monitor.md

# SKILL.md has valid frontmatter
head -5 skill/SKILL.md | grep -q "^---"

# SKILL.md under 500 lines
test $(wc -l < skill/SKILL.md) -lt 500
```

### Content Quality
- [ ] SKILL.md frontmatter has `name` and `description`
- [ ] Description includes keywords agents would match: "PR", "review", "comments", "CI", "checks", "resolve"
- [ ] All commands documented with flags
- [ ] All exit codes documented
- [ ] JSON output examples are valid JSON
- [ ] All internal file references (`references/`, `examples/`) resolve
- [ ] No broken links to external URLs
- [ ] Commands match actual implementation (verify against source)

### Agent Simulation
- [ ] An agent reading only SKILL.md can run `gh ghent summary --pr N --format json`
- [ ] An agent reading SKILL.md + agent-workflows.md can complete a full review cycle
- [ ] Exit code table matches actual implementation in `cli/errors.go` and each command's RunE

## Completion Criteria

1. `skill/SKILL.md` exists, < 500 lines, valid frontmatter
2. All 5 supporting files exist with substantive content
3. SKILL.md references all supporting files
4. All documented commands/flags match actual implementation
5. All documented exit codes match actual implementation
6. README.md updated with agent skill section
7. `make ci` still passes (no Go changes)
8. PROGRESS.md updated

## Commit

```
feat(skill): add Agent Skill for gh-ghent

- SKILL.md with agent-optimized usage guide
- Command reference, agent workflows, exit codes
- Review cycle and CI monitor example walkthroughs
- Installable via: npx skills add indrasvat/gh-ghent
```

## Session Protocol

1. Read CLAUDE.md
2. Read this task file
3. **Mark this task as IN PROGRESS**
4. Read source files (cli/*.go, domain/types.go, cli/errors.go) for accuracy
5. Execute steps 1-10
6. Run verification (structural + content quality + agent simulation)
7. **Mark this task complete**
8. Update `docs/PROGRESS.md`
9. Update CLAUDE.md Learnings if needed
10. Commit
