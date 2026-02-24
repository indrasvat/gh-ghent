# Task 8.1: Styled Help & Version Output

- **Phase:** 8 (Polish & DX)
- **Status:** TODO
- **Depends on:** None (all feature tasks complete)
- **Blocks:** None
- **L4 Visual:** Required (styled terminal output — verify with iterm2-driver screenshots)

## Problem

ghent's `--version` and `--help` output uses Cobra's unadorned defaults — plain monochrome text indistinguishable from any other CLI tool. For a tool whose TUI is a meticulously styled Tokyo Night experience, the first thing users see (`gh ghent --help`) is a letdown. The help text is the storefront; right now it's a blank wall.

**Current state (from screenshot):**
- `--version` → `ghent version v0.1.0-1-g4e50809 (commit: 4e50809, built: 2026-02-24T04:53:19Z)` — flat, no color
- `--help` → Cobra default template: monochrome section headers, no visual hierarchy, no brand identity
- Subcommand `--help` → same Cobra defaults

**Desired state:**
- Version and help output that *feels like ghent* — Tokyo Night colors, clear visual hierarchy, brand presence
- Graceful degradation: full color in TTY, plain text when piped (identical to dual-mode pattern used everywhere else)
- Consistent with the aesthetic established in TUI mockups (`docs/tui-mockups.html`)

## Design

### Version Output (TTY)

```
  ghent v0.1.0  ·  commit 4e50809  ·  built 2026-02-24
  ^^^^^^         ^^^^^^^^^^^^^^^^^    ^^^^^^^^^^^^^^^^^^
  blue+bold      purple (short)       dim

  Agentic PR monitoring for GitHub
  ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
  dim italic
```

- App name in Blue, bold
- Version number in Green
- Commit hash truncated to 7 chars, in Purple
- Build date in Dim (date only, no time)
- Tagline in Dim
- Separator dots (`·`) in Dim

### Version Output (Pipe/non-TTY)

```
ghent v0.1.0 (commit: 4e50809, built: 2026-02-24)
```

Plain, machine-parseable, single line. No ANSI codes.

### Root Help (TTY)

```
  ghent — Agentic PR monitoring for GitHub
  ^^^^^   ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
  blue    dim

  Interactive Bubble Tea TUI for humans, structured output
  (json/md/xml) for AI agents. Works wherever gh is
  authenticated — zero config.

    TTY detected  → launches interactive TUI
    Piped / no-tui → outputs structured data (default: json)

  Commands:
    comments     Show unresolved review threads
    checks       Show CI check status
    resolve      Resolve review threads
    reply        Reply to a review thread
    summary      PR status dashboard

  Global Flags:
    --pr int            pull request number (required by subcommands)
    -R, --repo string   repository in OWNER/REPO format (default: current repo)
    -f, --format string output format: json, md, xml (pipe mode) [default: json]
    --since string      filter by timestamp (ISO 8601 or relative: 1h, 30m, 2d)
    --no-tui            force pipe mode even in TTY (for agents)
    --verbose           show additional context (diff hunks, debug info)
    --debug             enable debug logging to stderr
    -v, --version       version for ghent
    -h, --help          help for ghent

  Examples:
    # Interactive TUI for PR #42
    gh ghent comments --pr 42

    # Agent: get unresolved threads as JSON
    gh ghent comments --pr 42 --format json --no-tui

    # Quick merge-readiness check
    gh ghent summary --pr 42 --format json | jq '.is_merge_ready'

    # Watch CI until done, fail-fast on failure
    gh ghent checks --pr 42 --watch

  Use "ghent [command] --help" for more information about a command.
```

**Color mapping:**
- Title: app name in Blue bold, em dash + description in Dim
- Section headers (`Commands:`, `Global Flags:`, `Examples:`) in Blue, bold
- Command names in Cyan
- Command descriptions in default Text
- Flag names (long) in Cyan
- Flag shorthand (`-R`, `-f`) in Cyan
- Flag type hints (`string`, `int`) in Yellow
- Default values in Orange, brackets
- Comments in examples (`#`) in Dim
- Command text in examples in Text
- Flag values in examples in Orange
- Mode arrows (`→`) in Green
- Footer hint in Dim

### Root Help (Pipe/non-TTY)

Standard Cobra defaults — no ANSI codes, plain text. Same content, just no styling.

### Subcommand Help (TTY)

Same color scheme applied to subcommand templates:
- Subcommand name in Cyan bold
- Section headers in Blue bold
- Flags styled identically to root
- Usage line with command path colored

### Subcommand Help (Pipe/non-TTY)

Plain Cobra defaults.

## Implementation Approach

### Custom Cobra Templates

Cobra supports `SetUsageTemplate()`, `SetHelpTemplate()`, and `SetVersionTemplate()` on any command. The strategy:

1. **Create `internal/cli/help.go`** — custom template functions and templates
2. **Detect TTY** at template render time (not at PersistentPreRunE, since `--help` runs before PreRunE for the root command)
3. **Register lipgloss-styled templates** for TTY, let Cobra defaults handle non-TTY
4. **Custom template functions** via `cobra.AddTemplateFunc()` for color helpers

### Template Function Approach

Register template functions that apply lipgloss styles:

```go
cobra.AddTemplateFunc("blue", func(s string) string { ... })
cobra.AddTemplateFunc("cyan", func(s string) string { ... })
cobra.AddTemplateFunc("dim", func(s string) string { ... })
// etc.
```

Then use them in templates:
```
{{blue "Commands:"}}
  {{cyan .Name}}  {{.Short}}
```

### TTY Detection for Templates

Cobra renders templates before `PersistentPreRunE` runs (for `--help`/`--version`). So TTY detection must happen inside the template functions themselves, or be set up in a `cobra.OnInitialize()` callback.

Strategy: use `term.FromEnv().IsTerminalOutput()` inside template funcs — if non-TTY, return the string unstyled.

### Version Template

Override `cmd.SetVersionTemplate()` with a custom template that uses the color helpers.

### No lipgloss.Background()

Per TUI pitfalls: never use `lipgloss.Background()`. Only use `lipgloss.Foreground()` for coloring text in help output. This is safe since help text is rendered to stdout directly, not inside a Bubble Tea program.

## Files to Create

- `internal/cli/help.go` — custom help/version templates, template functions, TTY-aware styling

## Files to Modify

- `internal/cli/root.go` — register custom templates on root command, call setup from `NewRootCmd()`
- `internal/version/version.go` — optionally add `ShortCommit()`, `ShortDate()` helpers

## Execution Steps

### Step 1: Read context
1. Read `internal/cli/root.go` (current Cobra setup)
2. Read `internal/version/version.go` (current version format)
3. Read `internal/tui/styles/theme.go` (color constants)
4. Read `internal/tui/styles/styles.go` (existing lipgloss style patterns)
5. Read `internal/cli/flags.go` (GlobalFlags struct)
6. Read Cobra template documentation for `SetHelpTemplate()`, `SetVersionTemplate()`, `SetUsageTemplate()`

### Step 2: Create help.go with template functions
1. Create `internal/cli/help.go`
2. Add TTY-detection helper (cached, using `term.FromEnv().IsTerminalOutput()`)
3. Add template funcs: `blue`, `cyan`, `dim`, `green`, `purple`, `orange`, `yellow`, `bold`, `faint`
4. Each func checks TTY — if non-TTY, returns string unchanged
5. Register funcs via `cobra.AddTemplateFunc()` in an `init()` or explicit setup function

### Step 3: Create custom version template
1. Define version template string using template funcs
2. TTY version: styled with colors, truncated commit, date-only build time, tagline
3. Non-TTY version: clean single line (automatically handled by template funcs returning plain text)
4. Register via `cmd.SetVersionTemplate()`

### Step 4: Create custom root help template
1. Define root help template with sections: title, description, commands, flags, examples, footer
2. Apply color template funcs to each section
3. Reorder sections for better flow: description → commands → examples → flags → footer
4. Register via `cmd.SetHelpTemplate()`

### Step 5: Create custom subcommand help template
1. Define subcommand template (similar to root but with usage line, no command list)
2. Apply same color scheme
3. Register on each subcommand, or set on root and let inheritance work

### Step 6: Add version helpers
1. Add `ShortCommit() string` to version package (first 7 chars)
2. Add `ShortDate() string` to version package (date only, no time)
3. Update `String()` to use these (or keep for backward compat, add `StyledString()`)

### Step 7: Wire into root command
1. In `NewRootCmd()`, call the template setup function
2. Ensure templates are registered before any `cmd.Execute()` call

### Step 8: Test
1. `make build && make install`
2. `gh ghent --version` — verify styled output in terminal
3. `gh ghent --help` — verify styled help in terminal
4. `gh ghent comments --help` — verify styled subcommand help
5. `gh ghent --version 2>&1 | cat` — verify plain output when piped
6. `gh ghent --help 2>&1 | cat` — verify plain output when piped
7. `gh ghent --version | grep -o 'ghent'` — verify parseable without ANSI
8. `make lint` — verify no lint issues
9. `make test` — verify no regressions
10. L4: iterm2-driver screenshots of version and help output

## Testing

### Unit Tests
- Template func returns styled string when TTY=true
- Template func returns plain string when TTY=false
- `ShortCommit()` returns 7-char prefix (or "unknown" for "unknown")
- `ShortDate()` returns date portion only

### L3 Real Testing
```bash
# TTY mode (run in terminal, visually inspect)
gh ghent --version
gh ghent --help
gh ghent comments --help
gh ghent checks --help
gh ghent resolve --help
gh ghent reply --help
gh ghent summary --help

# Pipe mode (verify no ANSI codes)
gh ghent --version | cat -v     # no escape sequences
gh ghent --help | cat -v        # no escape sequences

# Parseable version
gh ghent --version | grep -oP 'v[\d.]+'  # extracts version number
```

### L4 Visual Tests
- Screenshot `gh ghent --version` — verify Tokyo Night colors
- Screenshot `gh ghent --help` — verify visual hierarchy, color mapping
- Screenshot `gh ghent comments --help` — verify subcommand styling

## Verification

### Structural
```bash
# New file exists
test -f internal/cli/help.go

# Builds cleanly
make build

# Lint passes
make lint

# Tests pass
make test
```

### Visual
- [ ] Version output shows app name in blue, version in green, commit in purple, date in dim
- [ ] Root help shows section headers in blue bold
- [ ] Command names in cyan
- [ ] Flag names in cyan, types in yellow, defaults in orange
- [ ] Examples have dimmed comments, colored flags
- [ ] Subcommand help follows same color scheme
- [ ] Piped output has zero ANSI escape sequences
- [ ] Piped version is single-line and parseable

### Behavioral
- [ ] `gh ghent --version` works (exit 0)
- [ ] `gh ghent --help` works (exit 0)
- [ ] `gh ghent comments --help` works (exit 0)
- [ ] `gh ghent --version | cat` produces clean text
- [ ] `gh ghent --help | cat` produces clean text
- [ ] No regressions in existing commands
- [ ] `make ci-fast` passes

## Completion Criteria

1. `internal/cli/help.go` exists with template functions and custom templates
2. Version output is styled in TTY, plain in pipe
3. Root help output is styled in TTY, plain in pipe
4. All subcommand help output is styled in TTY, plain in pipe
5. Tokyo Night color scheme applied consistently
6. Zero ANSI codes in piped output (verified with `cat -v`)
7. `make ci-fast` passes
8. L4 iterm2-driver screenshots captured
9. PROGRESS.md updated

## Commit

```
feat(cli): add styled help and version output with Tokyo Night theme

Custom Cobra templates with lipgloss-styled help text and version
display. TTY-aware: full Tokyo Night colors in terminal, clean plain
text when piped. Consistent with TUI aesthetic.
```

## Session Protocol

1. Read CLAUDE.md
2. Read this task file
3. Query `cm context "styled help version output cobra templates lipgloss"` for relevant rules
4. **Mark this task as IN PROGRESS**
5. Execute steps 1-8
6. Run verification (structural + visual + behavioral)
7. **Mark this task complete**
8. Update `docs/PROGRESS.md`
9. Update CLAUDE.md Learnings if needed
10. Commit
