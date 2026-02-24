# Go CLI Project Patterns Research

> Research extracted from 5 Go projects and 1 Rust project across the indrasvat codebase.
> Date: 2026-02-21

## Projects Analyzed

| Project | Language | Description | Go Version |
|---------|----------|-------------|------------|
| **dorikin** | Go | Kubernetes configuration drift detector TUI | 1.25.0 |
| **yukti** | Go | Google Apps Script TUI manager | 1.25.5 |
| **gp-vpn** | Go | GlobalProtect VPN CLI for macOS | 1.25.5 |
| **vivecaka** | Go | Plugin-based GitHub PR TUI | 1.25.7 |
| **nidhi** | Go | Git stash TUI (BubbleTea v2) | 1.26.0 |
| **shux** | Rust | Terminal multiplexer (patterns only) | N/A |

---

## 1. Directory Structure Convention

All Go projects follow a consistent layout based on the standard Go project layout:

```
project-name/
├── cmd/
│   └── project-name/
│       └── main.go              # Minimal entrypoint, delegates to internal/cli
├── internal/
│   ├── cli/                     # Cobra commands (root.go, subcommands)
│   ├── config/                  # Configuration loading
│   ├── domain/                  # Domain entities and interfaces (clean arch)
│   ├── tui/                     # BubbleTea TUI components
│   │   ├── components/          # Reusable UI components
│   │   ├── views/               # View models
│   │   └── core/                # Shared TUI types (theme, styles, keymap)
│   ├── usecase/                 # Application use cases (vivecaka)
│   ├── adapter/                 # External service adapters (vivecaka)
│   ├── buildinfo/               # Version info package (yukti)
│   └── version/                 # Version info package (gp-vpn)
├── pkg/                         # Public packages (optional, only dorikin uses pkg/api/)
├── docs/
│   ├── PRD.md                   # Product requirements document
│   ├── PROGRESS.md              # Implementation progress tracker
│   └── tasks/                   # Individual task specification files
├── scripts/                     # Helper scripts
├── testdata/                    # Test fixtures
├── bin/                         # Build output (gitignored)
├── dist/                        # Release output (gitignored)
├── coverage/                    # Coverage reports (gitignored)
├── .claude/                     # Claude Code settings (gitignored)
│   ├── settings.json            # Hooks configuration
│   ├── automations/             # iTerm2 automation scripts
│   └── screenshots/             # Visual test screenshots
├── CLAUDE.md                    # AI agent instructions (source of truth)
├── AGENTS.md                    # Redirect to CLAUDE.md
├── Makefile                     # Build system (command interface)
├── lefthook.yml                 # Git hooks configuration
├── .goreleaser.yaml             # Release configuration
├── .golangci.yml                # Linter configuration
├── .gitignore
├── go.mod
├── go.sum
└── README.md
```

### Key Observations

- **`cmd/project-name/main.go` is always minimal** -- typically 5-15 lines, just calls `cli.Execute()`.
- **All business logic lives in `internal/`** -- never in `cmd/`.
- **`pkg/` is rarely used** -- only dorikin has `pkg/api/` for public API types. Most projects keep everything in `internal/`.
- **`internal/cli/` contains Cobra commands** -- root.go defines the root command, each subcommand is a separate file.
- **TUI projects have `internal/tui/`** with `components/`, `views/`, and sometimes `core/` subdirectories.
- **Clean architecture** (vivecaka) uses `domain/`, `usecase/`, `adapter/` separation.

---

## 2. main.go Pattern

Every project follows the same minimal main.go pattern:

```go
// Package main is the entry point for project-name.
package main

import (
    "os"

    "github.com/indrasvat/project-name/internal/cli"
)

func main() {
    if err := cli.Execute(); err != nil {
        os.Exit(1)
    }
}
```

Some projects (gp-vpn) add `fmt.Fprintf(os.Stderr, "error: %v\n", err)` before the exit.

---

## 3. Go Module Naming Convention

```
module github.com/indrasvat/project-name
```

Consistent pattern: `github.com/indrasvat/<project-name>`

**Exception:** yukti uses `module yukti` (short module path, without the GitHub prefix). This appears to be a deviation -- the other projects consistently use the full GitHub path.

---

## 4. Cobra CLI Pattern

### root.go Structure

```go
// Package cli provides the command-line interface for project-name.
package cli

import (
    "github.com/spf13/cobra"
)

// Version information set by ldflags.
var (
    Version   = "dev"
    Commit    = "unknown"
    BuildDate = "unknown"
    GoVersion = "unknown"
)

var rootCmd = &cobra.Command{
    Use:   "project-name",
    Short: "One-line description",
    Long:  `Multi-line description with details.`,
}

// Execute runs the root command.
func Execute() error {
    return rootCmd.Execute()
}

func init() {
    rootCmd.PersistentFlags().StringP("flag-name", "f", "", "description")
    rootCmd.PersistentFlags().Bool("debug", false, "enable debug logging")
}
```

### Subcommand Pattern

Each subcommand is a separate file in `internal/cli/`:
- `version.go` - version command
- `scan.go` - scan subcommand (dorikin)
- `ui.go` - TUI launcher
- `status.go` - status display

### Common CLI Flags

- `--debug` / `--verbose` - debug/verbose output
- `--format` / `-o` - output format (json, text, plain)
- `-k` / `--kubeconfig` - K8s config path (dorikin)
- `--token-file` - auth token storage (yukti)

---

## 5. Makefile Pattern

All projects use the same structured Makefile approach. The Makefile is considered the **command interface** -- all operations go through `make`.

### Standard Variables Block

```makefile
# Variables
BINARY_NAME := project-name
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GO_VERSION := $(shell go version | cut -d' ' -f3)

# Build flags
LDFLAGS := -s -w \
    -X github.com/indrasvat/project-name/internal/cli.Version=$(VERSION) \
    -X github.com/indrasvat/project-name/internal/cli.Commit=$(COMMIT) \
    -X github.com/indrasvat/project-name/internal/cli.BuildDate=$(BUILD_DATE) \
    -X github.com/indrasvat/project-name/internal/cli.GoVersion=$(GO_VERSION)

# Directories
BIN_DIR := bin
DIST_DIR := dist
COVERAGE_DIR := coverage

# Tools
GOLANGCI_LINT := golangci-lint
GORELEASER := goreleaser
GOTESTSUM := $(shell command -v gotestsum 2> /dev/null)

# Colors for output
COLOR_RESET := \033[0m
COLOR_BOLD := \033[1m
COLOR_GREEN := \033[32m
COLOR_YELLOW := \033[33m
COLOR_BLUE := \033[34m
COLOR_MAGENTA := \033[35m
```

### Standard Targets (in order)

| Target | Description | Present In |
|--------|-------------|------------|
| `help` | Show help message (`.DEFAULT_GOAL`) | All |
| `build` | Build binary to `bin/` | All |
| `build-all` | Cross-platform builds | dorikin, yukti |
| `install` | Install to GOPATH/bin or ~/.local/bin | All |
| `run` | Run the app (development) | All |
| `test` | Run tests (gotestsum with fallback to go test) | All |
| `test-v` | Verbose tests (BDD/testdox style) | dorikin, yukti |
| `test-cover` | Tests with coverage report | All |
| `test-short` | Short tests only | dorikin, yukti, gp-vpn |
| `bench` | Run benchmarks | All |
| `lint` | Run golangci-lint | All |
| `lint-fix` | Auto-fix lint issues | dorikin, yukti, vivecaka |
| `fmt` | Format code (gofmt + golangci-lint fmt) | All |
| `fmt-check` | Check formatting without changes | gp-vpn, shux |
| `vet` | Run go vet | dorikin, yukti, gp-vpn, vivecaka |
| `tidy` | Run go mod tidy | dorikin, yukti, vivecaka |
| `verify` | Run go mod verify | dorikin, yukti, vivecaka |
| `ci` | Full CI pipeline (tidy + verify + vet + lint + test + build) | All |
| `ci-fast` | Fast CI for local dev | dorikin, yukti |
| `clean` | Remove build artifacts | All |
| `deps` | Download dependencies | dorikin, yukti, gp-vpn, vivecaka |
| `tools` | Install development tools | All |
| `tools-ci` | Install CI tools (minimal set) | dorikin, yukti, vivecaka |
| `hooks` | Install lefthook git hooks | All |
| `hooks-uninstall` | Remove git hooks | yukti, vivecaka |
| `hooks-run` | Run hooks manually | yukti, dorikin |
| `version` | Show version info | All |
| `info` | Show project info | dorikin, yukti |
| `release-check` | Validate goreleaser config | dorikin, yukti |
| `release-snapshot` | Build snapshot release | dorikin, yukti, vivecaka |
| `release` | Create release (requires GITHUB_TOKEN) | All |

### CI Target Composition

The standard CI pipeline is:

```makefile
ci: tidy verify vet lint test build
```

Some projects also include `fmt` and `govulncheck`:

```makefile
# vivecaka (most comprehensive)
ci: tidy verify fmt vet lint govulncheck test build
```

### Test Target with gotestsum Fallback

```makefile
test:
    @echo "$(COLOR_BLUE)Running tests...$(COLOR_RESET)"
ifdef GOTESTSUM
    gotestsum --format pkgname-and-test-fails --format-icons hivis -- -race -shuffle=on ./...
else
    go test -race -shuffle=on ./...
endif
    @echo "$(COLOR_GREEN)Tests passed$(COLOR_RESET)"
```

### Build Target Pattern

```makefile
build:
    @echo "$(COLOR_BLUE)Building $(BINARY_NAME)...$(COLOR_RESET)"
    @mkdir -p $(BIN_DIR)
    go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME) ./cmd/$(BINARY_NAME)
    @echo "$(COLOR_GREEN)Built $(BIN_DIR)/$(BINARY_NAME)$(COLOR_RESET)"
```

### Output Style

All Makefiles use colored terminal output with a consistent prefix pattern:
- Blue `>>` for action start
- Green check mark for success
- Yellow for warnings
- Red for errors

---

## 6. CLAUDE.md Format and Content Pattern

CLAUDE.md is the **single source of truth** for all AI coding agents. The standard format has these sections:

### Standard Header

```markdown
# CLAUDE.md -- project-name AI Agent Instructions

> **This file is the source of truth for all AI coding agents working on project-name.**
> AGENTS.md points here. Do not duplicate instructions elsewhere.
```

### Standard Sections (in order)

1. **Project Overview** - What the project is, tech stack summary
2. **Build & Test Commands** - Code block listing all `make` targets
3. **Architecture** - ASCII diagram of module/crate/package structure
4. **Code Conventions** - Formatting, linting, errors, imports, testing rules
5. **Git Workflow** - Branch naming, commit conventions, hooks
6. **Session Protocol** (shux, nidhi) - Rules for starting/completing tasks, updating PROGRESS.md
7. **Key Decisions** - Table of architectural decisions with rationale and date
8. **Important API Notes** - Specific library versions and gotchas
9. **Learnings** - Dated entries of concrete, actionable insights (STRICT RULE: updated every session)

### Key Rules Always Present

- "Always use Makefile targets, never raw commands"
- "Conventional commits (feat:, fix:, refactor:, test:, docs:, chore:)"
- "Run `make ci` before committing/pushing"
- Errors: return errors, don't panic; use `fmt.Errorf("context: %w", err)` for wrapping
- Testing: table-driven tests, `-race` flag, test file naming `foo_test.go`

### Learnings Section Pattern

```markdown
## Learnings

> **STRICT RULE:** This section MUST be updated at the end of every coding session.
> Each entry should be a concrete, actionable insight. Delete entries that become obsolete.

- **2026-02-18 (task 001):** Specific finding...
- **2026-02-19 (task 012):** Another finding...
```

---

## 7. AGENTS.md Pattern

AGENTS.md is always a simple redirect:

```markdown
# AGENTS.md

All project conventions, architecture decisions, coding standards, and operational
guidance are maintained in a single source of truth:

**[CLAUDE.md](CLAUDE.md)**

Agents (Claude Code, Codex, Gemini, etc.) MUST read CLAUDE.md before making any
changes to this codebase.
```

---

## 8. lefthook.yml Pattern

### Minimal Pattern (most common across Go projects)

```yaml
# lefthook.yml - Git hooks configuration
# Runs full CI on pre-push only (not pre-commit for speed)

pre-push:
  commands:
    ci:
      run: make ci
      fail_text: "CI failed! Fix issues before pushing."
```

This is the most common pattern (dorikin, yukti, vivecaka all use this exact structure).

### Extended Pattern (nidhi)

```yaml
pre-commit:
  parallel: true
  commands:
    lint:
      run: make lint
      stage_fixed: true
    test-unit:
      run: go test -short -count=1 ./...

pre-push:
  commands:
    test-full:
      run: make test
```

### Extended Pattern with Glob Filtering (shux/Rust)

```yaml
pre-commit:
  parallel: true
  commands:
    fmt-check:
      glob: "*.rs"
      run: make fmt-check
    clippy:
      glob: "*.rs"
      run: make clippy

pre-push:
  commands:
    progress-check:
      run: make check-progress
    test:
      run: make test
    deny:
      run: make deny-soft
```

### Smart Skip Pattern (gp-vpn)

```yaml
pre-push:
  commands:
    ci-check:
      run: |
        CHANGED_FILES=$(git diff --name-only @{push}..HEAD 2>/dev/null || git diff --name-only HEAD~1..HEAD)
        CODE_CHANGES=$(echo "$CHANGED_FILES" | grep -E '\.(go|mod|sum)$|^Makefile$' || true)
        if [ -z "$CODE_CHANGES" ]; then
          echo "No code changes detected, skipping CI..."
          exit 0
        fi
        make ci
```

### Common Pattern Summary

- **pre-push with `make ci`** is the universal minimum
- **pre-commit is optional** -- only used in nidhi and shux for faster feedback
- All hooks invoke **Makefile targets**, never raw commands
- `parallel: true` used for independent pre-commit checks

---

## 9. .golangci.yml Pattern

All Go projects use golangci-lint v2 configuration:

```yaml
version: "2"

run:
  timeout: 5m
  modules-download-mode: readonly   # optional

formatters:
  enable:
    - gofmt
    - goimports
  settings:
    goimports:
      local-prefixes:
        - github.com/indrasvat/project-name

linters:
  enable:
    # Core quality
    - errcheck
    - govet
    - staticcheck
    - ineffassign
    - unused
    # Style
    - misspell
    # Bugs
    - bodyclose
    - nilerr
    # Modern Go
    - noctx
    - gocritic
    # Error handling
    - errorlint
    # Complexity (optional)
    - gocyclo       # min-complexity: 15-80
    - gocognit      # min-complexity: 20-70
    # Security
    - gosec

  settings:
    errcheck:
      exclude-functions:
        - fmt.Fprintf
        - fmt.Fprintln
        - fmt.Printf
        - (io.Closer).Close
    gosec:
      excludes:
        - G104   # Unhandled errors (covered by errcheck)
        - G304   # File path from variable

  exclusions:
    rules:
      - path: _test\.go
        linters:
          - gosec
          - errcheck
```

### Common Enabled Linters

Present in most/all projects: `errcheck`, `govet`, `staticcheck`, `ineffassign`, `unused`, `misspell`

Present in most projects: `bodyclose`, `nilerr`, `gocritic`, `errorlint`, `gosec`, `noctx`

Optional: `gocyclo`, `gocognit`, `unconvert`, `unparam`, `revive`, `prealloc`

---

## 10. GoReleaser Pattern

All Go projects use goreleaser v2:

```yaml
version: 2

project_name: project-name

before:
  hooks:
    - go mod tidy
    - go mod verify

builds:
  - id: project-name
    main: ./cmd/project-name
    binary: project-name
    env:
      - CGO_ENABLED=0
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64
    ignore:
      - goos: windows
        goarch: arm64
    ldflags:
      - -s -w
      - -X github.com/indrasvat/project-name/internal/cli.Version={{.Version}}
      - -X github.com/indrasvat/project-name/internal/cli.Commit={{.ShortCommit}}
      - -X github.com/indrasvat/project-name/internal/cli.BuildDate={{.Date}}
      - -X github.com/indrasvat/project-name/internal/cli.GoVersion={{.Env.GO_VERSION}}

archives:
  - formats:
      - tar.gz
    format_overrides:             # optional, nidhi uses this
      - goos: windows
        formats:
          - zip
    name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"
    files:
      - LICENSE
      - README.md

checksum:
  name_template: "checksums.txt"
  algorithm: sha256

changelog:
  use: github
  sort: asc
  groups:
    - title: Features
      regexp: '^.*?feat(\([[:word:]]+\))??!?:.+$'
      order: 0
    - title: Bug Fixes
      regexp: '^.*?fix(\([[:word:]]+\))??!?:.+$'
      order: 1
    - title: Documentation
      regexp: '^.*?docs(\([[:word:]]+\))??!?:.+$'
      order: 2
    - title: Others
      order: 999
  filters:
    exclude:
      - "^style:"
      - "^chore\\(deps\\):"

release:
  github:
    owner: indrasvat
    name: project-name
  draft: false
  prerelease: auto
  name_template: "{{ .Tag }}"
  header: |
    ## Project Name {{ .Tag }}

    Description.

    **macOS users:** After downloading, you may need to remove the quarantine attribute:
    ```bash
    xattr -d com.apple.quarantine project-name
    ```
  footer: |
    ---
    **Full Changelog**: https://github.com/indrasvat/project-name/compare/{{ .PreviousTag }}...{{ .Tag }}
```

### Key Patterns

- `CGO_ENABLED=0` for static binaries
- Cross-compilation: linux/darwin (amd64+arm64), windows (amd64 only)
- Windows arm64 is always excluded
- `sha256` checksums
- Conventional commit changelog grouping
- macOS quarantine warning in release header
- nidhi additionally has a `brews:` section for Homebrew tap

---

## 11. .gitignore Pattern

```gitignore
# IDE
.idea/
.vscode/

# Build output
bin/
dist/
project-name

# Test artifacts
*.test
*.out
coverage.*
coverage/

# Debug
__debug_bin*
*.log

# Local/temp
.local/
*.env
*.tmp
*~
*.swp

# OS
.DS_Store
Thumbs.db

# Claude Code
.claude/
screenshots/
```

---

## 12. Testing Patterns

### Test Flags

All projects use:
- `-race` - race detector (always enabled)
- `-shuffle=on` - randomize test order (dorikin, yukti, vivecaka)
- `-coverprofile=coverage.out` - coverage output
- `-covermode=atomic` - atomic coverage mode

### gotestsum Usage

gotestsum is the preferred test runner with fallback to `go test`:
- `--format pkgname-and-test-fails` - default format (compact)
- `--format testdox` - BDD-style verbose output
- `--format dots` - minimal dots output
- `--format-icons hivis` - high-visibility icons

### Test Organization

- Unit tests: `foo_test.go` alongside source files
- Integration tests: `internal/e2e/` (nidhi) or `internal/tui/integration_test.go` (vivecaka)
- Test fixtures: `testdata/` directories at package level
- Performance tests: `internal/perf/` (nidhi)

### Testing Libraries

- `github.com/stretchr/testify` - assertions (vivecaka, used after migration)
- `github.com/google/go-cmp` - deep comparison (dorikin)
- Table-driven tests are the standard pattern

---

## 13. Version Info / Build Info Pattern

### Version Variables via ldflags

Two patterns are used:

**Pattern A: Variables in cli package** (dorikin, gp-vpn)
```go
// internal/cli/root.go
var (
    Version   = "dev"
    Commit    = "unknown"
    BuildDate = "unknown"
    GoVersion = "unknown"
)
```

**Pattern B: Separate buildinfo package** (yukti)
```go
// internal/buildinfo/buildinfo.go
package buildinfo
var (
    Version   = "dev"
    Commit    = "unknown"
    BuildDate = "unknown"
    GoVersion = "unknown"
)
```

**Pattern C: Variables in main** (vivecaka, nidhi)
```go
// cmd/project-name/main.go
var (
    version   = "dev"
    commit    = "none"
    date      = "unknown"
    goVersion = "unknown"
)
```

Pattern B (separate buildinfo package) avoids import cycles when multiple packages need version info. Pattern A is simpler for projects where only the CLI package needs version info.

---

## 14. Error Handling Patterns

Consistent across all projects:
- Return errors, never panic (except in tests)
- `fmt.Errorf("context: %w", err)` for error wrapping
- `errors.Is()` for error comparison, not `==`
- Handle error return values from deferred `Close()` calls: `defer func() { _ = f.Close() }()`
- Pre-allocate slices when length is known: `make([]T, 0, len)`

---

## 15. Common Dependencies

### CLI Framework
- `github.com/spf13/cobra` -- used in dorikin, gp-vpn, yukti, vivecaka (not nidhi)

### TUI Framework
- `github.com/charmbracelet/bubbletea` -- all TUI projects
- `github.com/charmbracelet/lipgloss` -- all TUI projects
- `github.com/charmbracelet/bubbles` -- all TUI projects
- `github.com/muesli/termenv` -- terminal environment detection

### Configuration
- `github.com/pelletier/go-toml/v2` -- TOML config (nidhi, vivecaka, gp-vpn)

### Testing
- `github.com/stretchr/testify` -- assertions (vivecaka, gp-vpn)
- `github.com/google/go-cmp` -- deep comparison (dorikin)

### Logging
- `github.com/rs/zerolog` -- structured logging (gp-vpn)
- `log/slog` -- standard library logging (vivecaka)

---

## 16. PROGRESS.md Pattern

### Standard Header

```markdown
# project-name -- Implementation Progress

> **STRICT RULE:** This file MUST be updated at the end of every coding session.
```

### Structure

1. **Current Phase** -- which phase/milestone is active
2. **Status** -- milestone targets with checkboxes
3. **How to Resume** -- numbered steps for new sessions
4. **Phase Task Tables** -- numbered tasks with status

### Task Table Format

```markdown
| # | Task File | Description | Status | Depends On |
|---|-----------|-------------|--------|------------|
| 001 | `docs/tasks/001-task-name.md` | Brief description | DONE | -- |
| 002 | `docs/tasks/002-task-name.md` | Brief description | TODO | 001 |
```

### Session Log Format

```markdown
## Session Log

### 2026-02-19
- Task 016: Description -- Done
  - Bullet point of what was implemented
  - Another bullet point
  - Tests: 546 passing
  - Learning: key insight discovered
```

---

## 17. .claude/ Directory Pattern

The `.claude/` directory is gitignored and contains Claude Code settings:

### settings.json

```json
{
  "hooks": {
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "make -C \"$CLAUDE_PROJECT_DIR\" check-progress-active",
            "timeout": 10,
            "statusMessage": "Checking progress tracking..."
          }
        ]
      }
    ],
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "...",
            "timeout": 10,
            "statusMessage": "Verifying progress before push..."
          }
        ]
      }
    ]
  }
}
```

### Automation Scripts

- `.claude/automations/` -- iTerm2 driver scripts for visual TUI testing
- `.claude/screenshots/` -- captured screenshots (gitignored)

---

## 18. Git Workflow Conventions

Consistent across all projects:

### Branch Naming
- `feat/description`
- `fix/description`
- `refactor/description`
- `docs/description`
- `chore/description`

### Commit Format
Conventional commits: `type(scope): message`
- Types: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`, `style`
- Keep commits atomic and focused on a single change
- One-liner messages, frequent commits

### Rules
- Never `git add -A` or `git add .` -- explicitly enumerate files
- Run `make ci` before pushing
- lefthook enforces CI on pre-push

---

## 19. Development Tools

### Standard Tool Versions (as of Feb 2026)

| Tool | Version | Install Command |
|------|---------|----------------|
| golangci-lint | v2.7.2 - v2.8.0 | `go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.8.0` |
| goreleaser | v2 (latest) | `go install github.com/goreleaser/goreleaser/v2@latest` |
| gotestsum | latest | `go install gotest.tools/gotestsum@latest` |
| govulncheck | latest | `go install golang.org/x/vuln/cmd/govulncheck@latest` |
| lefthook | latest | `go install github.com/evilmartians/lefthook@latest` or `brew install lefthook` |
| air | latest | `go install github.com/air-verse/air@latest` (optional, live reload) |

---

## 20. Summary of Key Conventions

1. **Makefile is the command interface** -- all operations go through `make`, including those called by hooks
2. **CLAUDE.md is the single source of truth** for AI agents; AGENTS.md redirects to it
3. **Minimal main.go** -- delegates immediately to `internal/cli.Execute()`
4. **`internal/` for everything** -- `pkg/` only for genuinely public API types
5. **Cobra for CLI** with persistent flags on root, subcommands in separate files
6. **golangci-lint v2** with gofmt + goimports formatters and a standard set of linters
7. **lefthook pre-push runs `make ci`** -- the minimum git hook for all projects
8. **goreleaser v2** for releases with `CGO_ENABLED=0`, cross-compilation, and conventional commit changelogs
9. **Version info via ldflags** -- injected at build time into a cli or buildinfo package
10. **Tests with `-race -shuffle=on`** and gotestsum for output formatting
11. **Conventional commits** enforced by convention and documented in CLAUDE.md
12. **PROGRESS.md** tracks implementation with numbered tasks, phase tables, and session logs
13. **`.claude/` is gitignored** -- settings.json for hooks, automations/ for visual tests
14. **Colored Makefile output** with consistent blue/green/yellow/red styling
15. **`.DEFAULT_GOAL := help`** -- running bare `make` shows available targets
