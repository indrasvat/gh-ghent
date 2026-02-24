# Dorikin Project Patterns Research

Research conducted from `/Users/indrasvat/code/github.com/indrasvat-dorikin/` to extract Go CLI project patterns and conventions for use in the ghent project.

---

## 1. Project Overview

Dorikin is a Kubernetes configuration drift detector with a TUI (Terminal User Interface). It compares desired Kubernetes manifests against actual cluster state and surfaces configuration drift. It provides both a CLI for scripting/CI and an interactive TUI for exploration.

- **Language:** Go 1.25
- **CLI Framework:** Cobra v1.9+
- **TUI Framework:** Bubbletea v1.3+ / Lipgloss v1.1+
- **Linter:** golangci-lint v2.7.2
- **Release:** GoReleaser v2
- **Git Hooks:** Lefthook

---

## 2. Directory Structure

```
dorikin/
  cmd/dorikin/              # Single entry point: main.go
  internal/
    cli/                    # Cobra commands (root, scan, ui, version, common)
    config/                 # Configuration loading (.dorikin.yaml)
    drift/                  # Core business logic (comparator, detector, filter, etc.)
    k8s/                    # Kubernetes client wrapper
    loader/                 # Manifest loading interface + implementations (file, helm, kustomize)
    logcapture/             # Log capture/buffering system for TUI
    testinit/               # Test initialization helpers (deterministic styling)
    tui/                    # TUI components (model, view, update, diff, styles)
      styles/               # Lipgloss style definitions
  pkg/api/                  # Public API types (types.go only)
  scripts/                  # Shell scripts (test-track.sh, demo scripts)
  testdata/                 # Test manifests and fixtures
    track-manifests/        # K8s manifest test fixtures
    track-kustomize/        # Kustomize overlay fixtures
    smart-diff/             # Diff algorithm test fixtures
  assets/                   # Logo, demo GIF
  docs/                     # Documentation (drift-detection.md)
  bin/                      # Build output (gitignored)
  dist/                     # Release output (gitignored)
  coverage/                 # Test coverage output (gitignored)
  .github/workflows/        # CI and Release workflows
  .claude/                  # Claude Code settings and automations
```

### Key Structural Patterns

1. **Single binary entry point** at `cmd/<project>/main.go` - extremely minimal, just calls `cli.Execute()`.
2. **`internal/` for all private packages** - no code leaks to the public API except `pkg/api/types.go`.
3. **`pkg/api/` for shared types only** - contains `types.go` with all domain types (DriftStatus, ResourceRef, FieldDiff, ScanResult, ScanOptions, etc.). This is the only public package.
4. **Domain logic separated from CLI** - `internal/drift/` contains the core engine, `internal/cli/` handles Cobra wiring.
5. **Interface-based loader pattern** - `loader.Loader` interface with `FileLoader`, `HelmLoader`, `KustomizeLoader` implementations.
6. **TUI separated into model/view/update** following the Elm Architecture (Bubbletea pattern).
7. **Styles in a dedicated sub-package** - `tui/styles/styles.go` centralizes all Lipgloss styles.

---

## 3. CLAUDE.md Format

The CLAUDE.md file follows this structure:

```
# <Project Name> - Claude Code Instructions

## Project Overview
<One-line description of what the project is>

## Git Conventions
- Conventional commit format with type prefixes (feat:, fix:, refactor:, chore:, docs:, test:, style:)
- Atomic, one-liner commits
- Examples provided

## Build & Development
- Always use relevant Makefile targets
- Key targets listed with brief descriptions
- Domain-specific targets (e.g., "Test Track" for K8s testing)

## Tech Stack
- Versioned list of all major dependencies

## Project Structure
- ASCII tree showing directory layout with comments

## Color Palette
- Table of colors with hex values and usage descriptions

## VHS Tape Format (if applicable)
- Demo recording instructions
```

### Key CLAUDE.md Conventions

- Instructs Claude to "always use relevant Makefile targets" and "add new targets as you build the app and find missing ones."
- Explicitly calls out the Go version and any novel language features in use (e.g., `sync.WaitGroup.Go()` in Go 1.25).
- Lists exact dependency versions in the Tech Stack section.
- Provides concrete commit message examples.

---

## 4. Makefile Patterns

The Makefile is organized with clear visual sections using Unicode box-drawing separators:

```makefile
# ══════════════════════════════════════════════════════════════════════════════
# Section Name
# ══════════════════════════════════════════════════════════════════════════════
```

### Variables

```makefile
BINARY_NAME := dorikin
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GO_VERSION := $(shell go version | cut -d' ' -f3)
```

### LDFLAGS Pattern

```makefile
LDFLAGS := -s -w \
    -X <module>/internal/cli.Version=$(VERSION) \
    -X <module>/internal/cli.Commit=$(COMMIT) \
    -X <module>/internal/cli.BuildDate=$(BUILD_DATE) \
    -X <module>/internal/cli.GoVersion=$(GO_VERSION)
```

### Core Targets

| Target | Description | Pattern |
|--------|-------------|---------|
| `help` (default) | Self-documenting help using `awk` on `## ` comments | `.DEFAULT_GOAL := help` |
| `build` | Build binary to `bin/` dir | `go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME) ./cmd/<name>` |
| `build-all` | Cross-platform builds | `GOOS=<os> GOARCH=<arch> go build ...` for each platform |
| `install` | Install to GOPATH/bin | Depends on `build` |
| `run` | Run the binary (dev mode) | Depends on `build` |
| `test` | Run tests with race detector | `go test -race -shuffle=on ./...` with optional gotestsum |
| `test-v` | Verbose tests (BDD-style) | `gotestsum --format testdox` |
| `test-dots` | Dot-format tests (fast feedback) | `gotestsum --format dots` |
| `test-cover` | Tests with HTML coverage report | `-coverprofile` + `go tool cover -html` |
| `test-short` | Short tests only | `-short` flag |
| `bench` | Benchmarks | `go test -bench=. -benchmem ./...` |
| `lint` | Run golangci-lint | `$(GOLANGCI_LINT) run ./...` |
| `lint-fix` | Lint with auto-fix | `$(GOLANGCI_LINT) run --fix ./...` |
| `fmt` | Format code | `go fmt ./...` + `$(GOLANGCI_LINT) fmt ./...` |
| `vet` | Go vet | `go vet ./...` |
| `tidy` | Tidy go.mod | `go mod tidy` |
| `verify` | Verify deps | `go mod verify` |
| `ci` | Full CI pipeline | `tidy verify vet lint test build` (chained dependencies) |
| `ci-fast` | Fast CI for local dev | `vet lint-fix test-short build` |
| `clean` | Remove build artifacts | `rm -rf` + `go clean -cache -testcache` |
| `deps` | Download dependencies | `go mod download` |
| `tools` | Install dev tools | `go install` for golangci-lint, goreleaser, gotestsum |
| `tools-ci` | Minimal CI tools | golangci-lint + gotestsum only |
| `version` | Show version info | Prints version, commit, date, Go version |
| `release-check` | Validate GoReleaser config | `goreleaser check` |
| `release-snapshot` | Build snapshot (no publish) | `goreleaser release --snapshot --clean` |

### Testing Patterns in Makefile

- **gotestsum is optional** - detected via `$(shell command -v gotestsum 2> /dev/null)`, falls back to plain `go test`.
- **Multiple test output formats** - `pkgname-and-test-fails` (default), `testdox` (verbose BDD), `dots` (fast).
- **Always uses `-race -shuffle=on`** for test runs.
- **Coverage uses atomic mode** - `-covermode=atomic`.

### Color Output

```makefile
COLOR_RESET := \033[0m
COLOR_BOLD := \033[1m
COLOR_GREEN := \033[32m
COLOR_YELLOW := \033[33m
COLOR_BLUE := \033[34m
COLOR_MAGENTA := \033[35m
```

All targets print status messages with colors:
- Blue `▶` for "starting" messages
- Green `✓` for "success" messages

---

## 5. CLI Architecture (Cobra)

### Entry Point Pattern

```go
// cmd/dorikin/main.go
package main

import (
    "os"
    "<module>/internal/cli"
)

func main() {
    if err := cli.Execute(); err != nil {
        os.Exit(1)
    }
}
```

### Root Command Pattern

```go
// internal/cli/root.go
package cli

var (
    Version   = "dev"
    Commit    = "unknown"
    BuildDate = "unknown"
    GoVersion = "unknown"
)

var rootCmd = &cobra.Command{
    Use:   "<binary>",
    Short: "<one-line description>",
    Long:  `<multi-line description>`,
}

func Execute() error {
    return rootCmd.Execute()
}

func init() {
    rootCmd.PersistentFlags().StringP("kubeconfig", "k", "", "...")
    rootCmd.PersistentFlags().StringP("context", "c", "", "...")
    rootCmd.PersistentFlags().Bool("debug", false, "...")
    rootCmd.PersistentFlags().String("log-file", "", "...")
}
```

### Subcommand Pattern

Each subcommand is in its own file (`scan.go`, `ui.go`, `version.go`), registered via `init()`:

```go
func init() {
    rootCmd.AddCommand(scanCmd)
    // register flags...
}
```

### Shared Flags Pattern

Common flags between commands are extracted into a `ScanFlags` struct in `common.go`:

```go
type ScanFlags struct {
    Manifests    []string
    Namespaces   []string
    // ...
}

func RegisterScanFlags(cmd *cobra.Command, f *ScanFlags) {
    cmd.Flags().StringArrayVarP(&f.Manifests, "file", "f", nil, "...")
    // ...
}
```

### ScanContext Builder Pattern

A `BuildScanContext()` function assembles all dependencies from flags:

```go
type ScanContext struct {
    Detector *drift.Detector
    Options  api.ScanOptions
    Config   *config.Config
}

func BuildScanContext(cmd *cobra.Command, args []string, f *ScanFlags) (*ScanContext, error) {
    // 1. Load config file
    // 2. Merge CLI flags with config (CLI overrides config)
    // 3. Create K8s client
    // 4. Select loader based on flags
    // 5. Create detector
    // 6. Build options struct
    return &ScanContext{...}, nil
}
```

### Command RunE Pattern

Commands use `RunE` (not `Run`) to return errors:

```go
var scanCmd = &cobra.Command{
    Use:   "scan [paths...]",
    RunE:  runScan,
}

func runScan(cmd *cobra.Command, args []string) error {
    ctx, err := BuildScanContext(cmd, args, &scanFlags)
    if err != nil {
        return err
    }
    // ... do work ...
    return nil
}
```

### Version Command Pattern

Version info is set via ldflags at build time, with defaults for dev builds:

```go
var (
    Version   = "dev"
    Commit    = "unknown"
    BuildDate = "unknown"
    GoVersion = "unknown"
)
```

---

## 6. Testing Patterns

### Table-Driven Tests

All tests use the standard Go table-driven test pattern with descriptive names:

```go
func TestParseHPAAwareMode(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    api.HPAAwareMode
        wantErr bool
    }{
        {name: "empty defaults to manifests", input: "", want: api.HPAAwareModeManifests},
        {name: "invalid mode", input: "invalid", wantErr: true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := ParseHPAAwareMode(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("ParseHPAAwareMode() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("ParseHPAAwareMode() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Assertion Patterns

- Uses `google/go-cmp` for deep comparisons: `cmp.Diff(want, got)`
- Uses `t.Errorf` / `t.Fatalf` (standard library) - **no third-party assertion library** (no testify).
- Error checking pattern: `if (err != nil) != tt.wantErr`

### Test File Organization

- Test files live alongside source files (e.g., `comparator.go` / `comparator_test.go`)
- Tests are in the same package (not `_test` package) to access unexported functions
- `internal/testinit/init.go` provides a blank import for deterministic TUI test output

### Test Data

- `testdata/` directory at project root for fixture files
- Subdirectories for different test scenarios (`track-manifests/`, `smart-diff/`, `track-kustomize/`)
- YAML manifests used as real-world test inputs

### Test Coverage Areas

Tests cover:
- `internal/cli/common_test.go` - Flag parsing, loader building
- `internal/drift/comparator_test.go` - Deep comparison logic
- `internal/drift/filter_test.go` - Field filtering
- `internal/drift/detector_test.go` - Drift detection
- `internal/drift/hpa_test.go` - HPA-aware comparison
- `internal/drift/quantity_test.go` - K8s quantity normalization
- `internal/drift/report_test.go` - Report formatting
- `internal/drift/arraymatch_test.go` - Array matching algorithms
- `internal/loader/file_test.go` - YAML file loading
- `internal/loader/helm_test.go` - Helm template loading
- `internal/loader/kustomize_test.go` - Kustomize loading
- `internal/config/config_test.go` - Config file loading
- `internal/logcapture/capture_test.go` - Log capture system
- `internal/tui/model_test.go` - TUI model logic

---

## 7. Error Handling Patterns

### Wrapping with Context

All errors are wrapped with `fmt.Errorf("context: %w", err)`:

```go
if err != nil {
    return nil, fmt.Errorf("building config: %w", err)
}
```

### Error Propagation

- Functions return `error` as the last return value
- Cobra commands use `RunE` to propagate errors
- Main function converts errors to exit codes: `os.Exit(1)`

### Validation Errors

Clear, user-friendly error messages:

```go
return fmt.Errorf("invalid --hpa-aware mode: %q (valid: manifests, cluster, disabled)", mode)
return fmt.Errorf("no manifest paths (use -f, positional args, or paths in .dorikin.yaml)")
return fmt.Errorf("--helm and --kustomize are mutually exclusive")
```

### Exit Codes

```
0 = success (all resources in sync)
1 = drift detected
2 = error during scan
```

---

## 8. Dependency Choices

### Direct Dependencies

| Dependency | Purpose |
|-----------|---------|
| `spf13/cobra` v1.10+ | CLI framework |
| `charmbracelet/bubbletea` v1.3+ | TUI framework (Elm architecture) |
| `charmbracelet/bubbles` v0.21+ | TUI components (help, keys) |
| `charmbracelet/lipgloss` v1.1+ | Terminal styling |
| `muesli/termenv` v0.16+ | Terminal environment detection |
| `google/go-cmp` v0.7+ | Deep comparison in tests |
| `pmezard/go-difflib` v1.0+ | Unified diff generation |
| `gopkg.in/yaml.v3` | YAML parsing |
| `sigs.k8s.io/yaml` | Kubernetes-flavored YAML |
| `k8s.io/client-go` v0.34+ | Kubernetes API client |
| `k8s.io/apimachinery` v0.35+ | K8s API types |

### Dev Tools

| Tool | Version | Purpose |
|------|---------|---------|
| `golangci-lint` | v2.7.2 | Linting (pinned version) |
| `goreleaser` | v2 latest | Release builds |
| `gotestsum` | latest | Better test output (optional) |
| `lefthook` | latest | Git hooks |
| `vhs` | - | Terminal demo recording |

### Not Used (Notable)

- **No testify** - standard library `testing` + `go-cmp` only
- **No Viper** - custom config loading with `gopkg.in/yaml.v3`
- **No logrus/zap** - custom `logcapture` package for TUI-compatible logging

---

## 9. Golangci-lint Configuration

File: `.golangci.yml` (version: "2")

### Enabled Linters

- **Code quality:** errcheck, govet, staticcheck, ineffassign, unused
- **Style:** misspell
- **Bugs:** bodyclose, nilerr
- **Performance:** prealloc
- **Complexity:** gocyclo (max 15), gocognit (max 20)
- **Security:** gosec (excludes G104, G304)
- **Modern Go:** noctx, gocritic
- **Error handling:** errorlint

### Disabled

- depguard, exhaustive, funlen, wrapcheck, revive

### Formatters

- gofmt, goimports (with local-prefixes set to the module path)

### Test Exclusions

- Tests excluded from gosec and errcheck

---

## 10. GoReleaser Configuration

File: `.goreleaser.yaml` (version: 2)

- Builds for linux, darwin, windows (amd64, arm64; excluding windows/arm64)
- CGO_ENABLED=0
- Archives as tar.gz with LICENSE and README.md
- SHA256 checksums
- GitHub changelog with conventional commit grouping (Features, Bug Fixes, Documentation, Others)
- Excludes `style:` and `chore(deps):` from changelog
- Release header includes installation instructions and macOS quarantine workaround

---

## 11. CI/CD Workflows

### CI (`.github/workflows/ci.yml`)

- Triggers on push/PR to main
- Sets up Go with caching
- Runs `make tools-ci` then `make ci` (tidy, verify, vet, lint, test, build)
- Uploads coverage to Codecov
- Matrix build for linux/darwin/windows x amd64/arm64
- Uploads build artifacts (7-day retention)

### Release (`.github/workflows/release.yml`)

- Triggers on `v*.*.*` tags
- Uses `goreleaser/goreleaser-action@v6`
- Passes `GITHUB_TOKEN` and `GO_VERSION` env vars

---

## 12. Git Hooks (Lefthook)

File: `lefthook.yml`

```yaml
pre-push:
  commands:
    ci:
      run: make ci
      fail_text: "CI failed! Fix issues before pushing."
```

- Only runs on pre-push (not pre-commit) for speed
- Runs the full `make ci` pipeline

---

## 13. Claude Code Settings

File: `.claude/settings.local.json`

```json
{
  "permissions": {
    "allow": [
      "Bash(wc:*)",
      "Bash(eza:*)",
      "Bash(find:*)",
      "Bash(make:*)",
      "Bash(git add:*)",
      "Bash(git commit:*)",
      "Bash(go build:*)",
      "Bash(uv run:*)",
      "Bash(brew install:*)",
      "Skill(agent-browser)",
      "Bash(npx agent-browser:*)",
      "Bash(agent-browser open:*)",
      "Bash(agent-browser screenshot:*)",
      "Bash(agent-browser close:*)",
      "Bash(colima status:*)",
      "Bash(./scripts/test-track.sh:*)",
      "Bash(gh pr view:*)",
      "Bash(gh api:*)",
      "Bash(gh pr checks:*)"
    ]
  }
}
```

Permissions are whitelisted for: file utilities, make, git, go build, uv (Python), brew, browser agent, colima, project scripts, and GitHub CLI.

---

## 14. Design Patterns Summary

### Functional Options

Used for configurable constructors:

```go
type DetectorOption func(*Detector)

func WithLoader(l loader.Loader) DetectorOption {
    return func(d *Detector) { d.loader = l }
}

func NewDetector(client *k8s.Client, ignorePaths []string, opts ...DetectorOption) *Detector {
    d := &Detector{...defaults...}
    for _, opt := range opts {
        opt(d)
    }
    return d
}
```

Also used in `HelmLoader`:

```go
type HelmOption func(*HelmLoader)
func WithRelease(name string) HelmOption { ... }
func NewHelmLoader(opts ...HelmOption) *HelmLoader { ... }
```

### Interface-Based Abstraction

The `Loader` interface allows swapping manifest sources:

```go
type Loader interface {
    Load(ctx context.Context, paths []string, recursive bool) ([]api.Resource, error)
}
```

Implementations: `FileLoader`, `HelmLoader`, `KustomizeLoader`.

### Elm Architecture (TUI)

The TUI follows Bubbletea's Elm Architecture with separate files:
- `model.go` - State definition
- `update.go` - State transitions (message handling)
- `view.go` - Rendering
- `diff.go` - Specialized diff rendering
- `styles/` - Centralized styles

### Configuration Layering

Priority order (highest wins):
1. CLI flags
2. `.dorikin.yaml` config file
3. Built-in defaults (in `config/defaults.go`)

### Domain Type Organization

All public types in a single `pkg/api/types.go`:
- Status enums as typed strings (`DriftStatus`, `DiffType`, `HPAAwareMode`, `OutputFormat`)
- Render mode as typed int (`RenderMode`)
- Structs with JSON/YAML tags for serialization
- Helper methods on types (`HasDrift()`, `HasIssues()`, `String()`, `Emoji()`)

---

## 15. .gitignore Patterns

```
# IDE
.idea/
.vscode/

# Build output
bin/
dist/
<binary-name>

# Test artifacts
*.test
*.out
coverage.*

# Debug
__debug_bin*

# Local/temp
.local/
*.log
*.env
*.tmp
*~
*.swp

# OS
.DS_Store
Thumbs.db

# Claude
.claude/
```

---

## 16. Key Takeaways for Ghent

1. **Keep `cmd/<name>/main.go` minimal** - just `cli.Execute()` and `os.Exit(1)`.
2. **Put all domain types in `pkg/api/types.go`** - one file, well-documented, with JSON/YAML tags.
3. **Use `internal/` for everything else** - separate by concern (cli, config, core-logic, etc.).
4. **Makefile is the single source of truth** for all build/test/lint/release operations.
5. **Table-driven tests with go-cmp** - no testify, just stdlib + go-cmp.
6. **Functional options for configurable types** - clean API for optional configuration.
7. **Interface-based abstractions** for pluggable components.
8. **golangci-lint v2 with curated linter set** - balance between strictness and noise.
9. **Lefthook for pre-push hooks** running `make ci` - not pre-commit (for speed).
10. **GoReleaser v2 for releases** with conventional commit changelog grouping.
11. **Version info via ldflags** - set at build time, defaulting to "dev" for development.
12. **Errors wrapped with context** - `fmt.Errorf("doing X: %w", err)` throughout.
13. **Config files use YAML** with sensible defaults and CLI override precedence.
14. **CI runs `make ci`** - same command locally and in GitHub Actions.
