# Popular Go-Based GitHub CLI Extensions: Architecture & Best Practices Research

Research date: 2026-02-21

---

## Table of Contents

1. [Overview of the Ecosystem](#1-overview-of-the-ecosystem)
2. [go-gh v2 SDK (cli/go-gh)](#2-go-gh-v2-sdk-cligo-gh)
3. [gh-dash (dlvhdr/gh-dash)](#3-gh-dash-dlvhdrgh-dash)
4. [gh-poi (seachicken/gh-poi)](#4-gh-poi-seachickengh-poi)
5. [gh-screensaver (vilmibm/gh-screensaver)](#5-gh-screensaver-vilmibmgh-screensaver)
6. [gh-projects (heaths/gh-projects)](#6-gh-projects-heathsgh-projects)
7. [gh-grep (k1LoW/gh-grep)](#7-gh-grep-k1lowgh-grep)
8. [gh-workflow-stats (fchimpan/gh-workflow-stats)](#8-gh-workflow-stats-fchimpangh-workflow-stats)
9. [gh-actions-usage (codiform/gh-actions-usage)](#9-gh-actions-usage-codiformgh-actions-usage)
10. [gh-branch (mislav/gh-branch)](#10-gh-branch-mislavgh-branch)
11. [gh-copilot (github/gh-copilot)](#11-gh-copilot-githubgh-copilot)
12. [Default Scaffold from gh extension create](#12-default-scaffold-from-gh-extension-create)
13. [gh-extension-precompile GitHub Action](#13-gh-extension-precompile-github-action)
14. [Cross-Cutting Patterns & Best Practices](#14-cross-cutting-patterns--best-practices)
15. [Recommended Architecture for New Extensions](#15-recommended-architecture-for-new-extensions)

---

## 1. Overview of the Ecosystem

GitHub CLI extensions are repositories named `gh-*` that add functionality to the `gh` CLI. They can be:

- **Shell scripts** (simplest, e.g., gh-branch): Just a `gh-EXTENSION-NAME` executable script
- **Precompiled Go binaries** (most popular for complex extensions): Cross-compiled and distributed via GitHub Releases
- **Other languages**: Any language that produces an executable

The Go ecosystem is dominant for non-trivial extensions due to:
- The official **go-gh** (github.com/cli/go-gh/v2) SDK providing first-class support
- The **gh-extension-precompile** GitHub Action automating cross-compilation and release
- Access to the same internal APIs and conventions used by `gh` itself

---

## 2. go-gh v2 SDK (cli/go-gh)

**Repository**: https://github.com/cli/go-gh
**Latest release**: v2.13.0 (November 2025)
**License**: MIT
**Stars**: ~409

### Project Structure

```
cli/go-gh/
├── .github/
│   ├── CODE-OF-CONDUCT.md
│   └── CONTRIBUTING.md
├── internal/              # Private implementation packages
├── pkg/                   # Public API packages (the main value)
│   ├── api/               # REST and GraphQL clients
│   ├── auth/              # Token retrieval and host detection
│   ├── browser/           # Open URLs in user's preferred browser
│   ├── config/            # gh configuration file access
│   ├── jq/                # jq expression evaluation on JSON
│   ├── jsonpretty/        # Terminal JSON pretty-printer
│   ├── markdown/          # Terminal markdown renderer
│   ├── prompter/          # Interactive user prompts (select, input, confirm)
│   ├── repository/        # Repository detection and parsing
│   ├── ssh/               # SSH hostname alias resolution
│   ├── tableprinter/      # Column-formatted table output
│   ├── template/          # Go template processing on JSON
│   ├── term/              # Terminal capability detection
│   ├── text/              # Text processing utilities
│   └── x/                 # Experimental packages
│       ├── color/         # Accessible color rendering
│       └── markdown/      # Accessible markdown rendering
├── .golangci.yml
├── example_gh_test.go     # Usage examples
├── gh.go                  # Top-level Exec() function
├── gh_test.go
├── go.mod
├── go.sum
├── LICENSE
└── README.md
```

### Key Packages and Their APIs

#### pkg/api - GitHub API Clients

```go
// REST client (auto-authenticates from gh environment)
client, err := api.DefaultRESTClient()
// or with custom options:
client, err := api.NewRESTClient(api.ClientOptions{
    Host:        "github.com",
    AuthToken:   "token_here",
    EnableCache: true,
    CacheTTL:    24 * time.Hour,
    Headers:     map[string]string{"Accept": "application/vnd.github+json"},
    Log:         os.Stderr,
    Timeout:     30 * time.Second,
})

// REST methods
var response []struct{ Name string }
err := client.Get("repos/cli/cli/tags", &response)
err := client.Post("repos/owner/repo/issues", body, &response)
err := client.Put(path, body, &response)
err := client.Patch(path, body, &response)
err := client.Delete(path, &response)

// Raw request for streaming/binary downloads
resp, err := client.Request("GET", path, nil)
defer resp.Body.Close()

// GraphQL client
gqlClient, err := api.DefaultGraphQLClient()
// or with options:
gqlClient, err := api.NewGraphQLClient(api.ClientOptions{...})

// Struct-based queries
var query struct {
    Repository struct {
        Name  string
        Owner struct{ Login string }
    } `graphql:"repository(owner: $owner, name: $name)"`
}
variables := map[string]interface{}{
    "owner": graphql.String("cli"),
    "name":  graphql.String("cli"),
}
err := gqlClient.Query("GetRepo", &query, variables)

// Mutations
var mutation struct {
    AddStar struct {
        Starrable struct{ ID string }
    } `graphql:"addStar(input: $input)"`
}
err := gqlClient.Mutate("AddStar", &mutation, variables)

// Raw GraphQL
var resp map[string]interface{}
err := gqlClient.Do("query { viewer { login } }", nil, &resp)
```

**Error handling**:
```go
// REST errors
if httpErr, ok := err.(*api.HTTPError); ok {
    fmt.Printf("Status: %d, Message: %s\n", httpErr.StatusCode, httpErr.Message)
    for _, item := range httpErr.Errors {
        fmt.Printf("  Field: %s, Code: %s\n", item.Field, item.Code)
    }
}

// GraphQL errors
if gqlErr, ok := err.(*api.GraphQLError); ok {
    if gqlErr.Match("NOT_FOUND", "repository") {
        // handle specifically
    }
}
```

#### pkg/term - Terminal Detection

```go
t := term.FromEnv()  // Respects GH_FORCE_TTY, NO_COLOR, CLICOLOR, etc.

t.IsTerminalOutput()     // bool: is stdout a TTY?
t.IsColorEnabled()       // bool: safe to emit ANSI colors?
t.Is256ColorSupported()  // bool
t.IsTrueColorSupported() // bool
w, h, err := t.Size()   // terminal dimensions
theme := t.Theme()       // "light", "dark", or ""

t.In()     // io.Reader for stdin
t.Out()    // io.Writer for stdout
t.ErrOut() // io.Writer for stderr
```

#### pkg/tableprinter - Table Output

```go
t := term.FromEnv()
w, _, _ := t.Size()
printer := tableprinter.New(t.Out(), t.IsTerminalOutput(), w)

printer.AddHeader([]string{"NAME", "STATUS", "AGE"})
printer.AddField("my-repo")
printer.AddField("active", tableprinter.WithColor(func(s string) string {
    return "\x1b[32m" + s + "\x1b[m"  // green
}))
printer.AddField("3 days")
printer.EndRow()

if err := printer.Render(); err != nil {
    log.Fatal(err)
}
// TTY output:  NAME     STATUS  AGE
//              my-repo  active  3 days
// Non-TTY:     my-repo\tactive\t3 days
```

#### pkg/repository - Repo Detection

```go
// Detect from current git directory
repo, err := repository.Current()
fmt.Printf("%s/%s/%s\n", repo.Host, repo.Owner, repo.Name)

// Parse from string
repo, err := repository.Parse("cli/cli")
repo, err := repository.Parse("github.com/cli/cli")
repo, err := repository.ParseWithHost("cli/cli", "github.com")
```

#### pkg/auth - Authentication

```go
host, source := auth.DefaultHost()     // "github.com", "default"
token, source := auth.TokenForHost("github.com")  // reads from env/config/keyring
hosts := auth.KnownHosts()
isEnterprise := auth.IsEnterprise("my-ghes.example.com")
```

#### pkg/prompter - Interactive Prompts

```go
t := term.FromEnv()
p := prompter.New(t.In().(*os.File), t.Out().(*os.File), t.ErrOut().(*os.File))

idx, err := p.Select("Choose a repo:", "", []string{"repo-a", "repo-b", "repo-c"})
indices, err := p.MultiSelect("Pick labels:", nil, []string{"bug", "feature", "docs"})
name, err := p.Input("Enter name:", "default-value")
pass, err := p.Password("Enter token:")
ok, err := p.Confirm("Continue?", true)

// For testing - mock prompts:
mock := prompter.NewMock(t)
mock.RegisterConfirm("Continue?", func(prompt string, def bool) (bool, error) {
    return true, nil
})
```

#### pkg/jq - JQ Expression Processing

```go
// Filter JSON with jq expressions
input := strings.NewReader(`[{"name":"a"},{"name":"b"}]`)
err := jq.Evaluate(input, os.Stdout, ".[].name")
// Output: a\nb

// With formatting
err := jq.EvaluateFormatted(input, os.Stdout, ".", "  ", true)
```

#### Top-level gh.Exec()

```go
// Shell out to gh safely (uses the same gh binary)
stdOut, stdErr, err := gh.Exec("issue", "list", "-R", "cli/cli", "--json", "title")
if err != nil {
    log.Fatal(err)
}
fmt.Println(stdOut.String())
```

---

## 3. gh-dash (dlvhdr/gh-dash)

**Repository**: https://github.com/dlvhdr/gh-dash
**Stars**: ~10.2k (the most popular Go-based extension)
**Language**: Go (100%)
**License**: MIT

### Project Structure

```
gh-dash/
├── .github/
│   └── workflows/
│       ├── build-and-test.yaml    # CI: build + test
│       ├── dependabot-sync.yml    # Dependency updates
│       ├── docs.yaml              # Documentation site deploy
│       ├── go-release.yml         # GoReleaser-based release
│       └── lint.yml               # Linting (golangci-lint)
├── cmd/
│   ├── root.go                    # Cobra root command with flags
│   └── sponsors.go                # Sponsors subcommand
├── docs/                          # Documentation site (gh-dash.dev)
├── internal/
│   ├── config/                    # YAML config management
│   ├── data/                      # API data fetching/models
│   ├── git/                       # Git operations
│   ├── tui/                       # Bubble Tea TUI components
│   └── utils/                     # Shared utilities
├── testdata/                      # Test fixtures
├── .gh-dash.yml                   # Default config example
├── .golangci.yml                  # Linting rules
├── .goreleaser.yaml               # Release automation
├── Taskfile.yaml                  # Task runner (alternative to Make)
├── devbox.json                    # Reproducible dev environment
├── gh-dash.go                     # Main entry point (2 lines)
├── go.mod
└── go.sum
```

### How It Uses go-gh

- Uses `github.com/cli/go-gh/v2` for GitHub API access
- Repository detection via internal git package
- Follows gh conventions for authentication automatically

### CLI Flags/Args (via Cobra)

```go
// cmd/root.go
var rootCmd = &cobra.Command{
    Use:     "gh dash",
    Short:   "A beautiful CLI dashboard for GitHub",
    Version: buildVersion(version, commit, date, builtBy),
    Run: func(cmd *cobra.Command, args []string) {
        // Initialize TUI model and start Bubble Tea program
    },
}

func init() {
    rootCmd.Flags().StringVarP(&cfgFile, "config", "c", "", "path to config file")
    rootCmd.Flags().BoolVar(&debug, "debug", false, "enable debug logging")
    rootCmd.Flags().StringVar(&cpuprofile, "cpuprofile", "", "write cpu profile")
}
```

### Entry Point Pattern

```go
// gh-dash.go (main entry point - minimal)
package main

import "github.com/dlvhdr/gh-dash/v4/cmd"

func main() {
    cmd.Execute()
}
```

### Output Format

- Full TUI using Bubble Tea (bubbletea), Lip Gloss (lipgloss), and Glamour (markdown rendering)
- No --json flag (TUI-focused extension)
- Configuration via YAML at `~/.config/gh-dash/config.yml`

### Build/Release Process

- **GoReleaser v2** (.goreleaser.yaml):
  - Cross-compiles for: Linux, macOS, Windows, FreeBSD, Android
  - Architectures: amd64, arm64, arm, 386
  - CGO disabled for portability
  - Version info injected via ldflags
  - Binary naming: `gh-dash_{{ .Tag }}_{{ .Os }}-{{ .Arch }}`
  - Changelog auto-generated with conventional commit categories
  - Announcements to Bluesky and Discord

- **GitHub Actions workflows**:
  - `build-and-test.yaml`: CI on every push/PR
  - `go-release.yml`: Triggered by git tags, runs GoReleaser
  - `lint.yml`: golangci-lint

### Key Dependencies

- `charmbracelet/bubbletea` - TUI framework
- `charmbracelet/lipgloss` - Styling
- `charmbracelet/glamour` - Markdown rendering
- `spf13/cobra` - CLI framework
- `cli/go-gh/v2` - GitHub API access
- `koanf` - Configuration management

### Testing

- Test data fixtures in `testdata/`
- Standard Go testing with `go test`
- golangci-lint for static analysis

---

## 4. gh-poi (seachicken/gh-poi)

**Repository**: https://github.com/seachicken/gh-poi
**Purpose**: Safely clean up local branches that have been merged/closed
**Language**: Go (99.8%)
**License**: MIT

### Project Structure

```
gh-poi/
├── .github/                       # CI/CD workflows
├── cmd/                           # Git/GitHub command execution
├── conn/                          # Connection/API logic layer
├── mocks/                         # Generated test mocks
├── shared/                        # Shared types and utilities
├── .codecov.yml                   # Code coverage config
├── .goreleaser.yml                # Release automation
├── CONTRIBUTING.md
├── LICENSE
├── main.go                        # Entry point with CLI logic
├── main_e2e_test.go               # End-to-end tests
├── go.mod
└── go.sum
```

### How It Uses go-gh

gh-poi does NOT use go-gh directly. Instead, it uses:
- `cli/safeexec` for safe command execution (shelling out to `gh` and `git`)
- Direct subprocess calls to `gh` and `git` commands

This represents an older pattern before go-gh v2 was mature.

### CLI Flags/Args (manual flag parsing)

```go
// main.go - uses Go's standard flag package, not cobra
func main() {
    // Subcommand-style parsing
    // Supports: default (delete branches), lock, unlock, protect (deprecated), unprotect (deprecated)
    // Flags: --state (merged|closed), --dry-run, --debug
}
```

### Output Formatting

- Uses `briandowns/spinner` for progress indication
- Uses `fatih/color` for colored terminal output
- Green checkmark for deleted branches, red for errors
- Branch names with PR metadata displayed beneath
- Worktree awareness and lock status shown

### Error Handling

- Signal handling for graceful interruption (SIGINT)
- Custom error types in shared package
- Continues operation on individual branch errors

### Build/Release

- **GoReleaser** (.goreleaser.yml):
  - Pre-build: `go mod tidy`
  - CGO disabled
  - Targets: Darwin, FreeBSD, Linux, Windows
  - Archive format: binary (no compression)
  - Releases as drafts
  - Snapshot naming: `{{ .Version }}-next`

### Testing Approach

- **E2E tests** (`main_e2e_test.go`):
  - Output capture via pipe redirection of `os.Stdout`
  - CI-only execution guard (`onlyCI()` helper)
  - Tests verify formatted output strings (e.g., checking for specific emoji + text combinations)
  - Tests for dry-run mode, lock/unlock functionality
- **Unit tests** with `golang/mock` for mocking interfaces
- **Code coverage** tracked via Codecov

### Testing Pattern Example

```go
func captureOutput(f func()) string {
    r, w, _ := os.Pipe()
    // Replace os.Stdout temporarily
    os.Stdout = w
    color.Output = w
    f()
    w.Close()
    var buf bytes.Buffer
    io.Copy(&buf, r)
    return buf.String()
}

func TestDeleteBranches(t *testing.T) {
    output := captureOutput(func() {
        runMain(false, "merged") // dry-run=false, state=merged
    })
    assert.Contains(t, output, "Deleting branches...")
}
```

---

## 5. gh-screensaver (vilmibm/gh-screensaver)

**Repository**: https://github.com/vilmibm/gh-screensaver
**Purpose**: Animated terminal screensavers (fun/demo extension)
**Language**: Go (100%)
**License**: GPL-3.0
**Author**: Nate Smith (GitHub staff)

### Project Structure

```
gh-screensaver/
├── .github/workflows/             # CI/CD
├── savers/                        # Individual screensaver implementations
│   ├── fireworks/
│   ├── life/
│   ├── marquee/
│   ├── pipes/
│   ├── pollock/
│   ├── starfield/
│   └── shared/                    # SaverCreator interface
├── .gitignore
├── LICENSE
├── README.md
├── go.mod
├── go.sum
├── main.go                        # Entry point with Cobra CLI
└── util.go                        # Utility functions
```

### How It Uses go-gh

- Uses `cli/safeexec` (predecessor library, not full go-gh v2)
- Simple extension that doesn't need GitHub API access

### CLI Flags/Args (via Cobra)

```go
func rootCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "screensaver",
        Short: "Run a terminal screensaver",
        RunE:  runScreensaver,
    }
    cmd.Flags().StringVarP(&saverName, "saver", "s", "", "which screensaver to run")
    cmd.Flags().StringVarP(&repo, "repo", "R", "", "repository context")
    cmd.Flags().BoolVarP(&list, "list", "l", false, "list available screensavers")
    return cmd
}

func main() {
    if err := rootCmd().Execute(); err != nil {
        os.Exit(1)
    }
}
```

### Architecture Pattern

- **Interface-based screensaver system**: `shared.SaverCreator` interface
- Each screensaver registers itself and provides its own flag definitions
- Main loop: 100ms tick, tcell screen rendering, exit on any keystroke
- Clean separation between CLI framework and screensaver implementations

### Build/Release

- 8 published releases (latest: v2.0.1)
- GitHub Actions for CI
- Installation via `gh extension install vilmibm/gh-screensaver`

### Key Dependencies

- `spf13/cobra` - CLI framework
- `gdamore/tcell/v2` - Terminal rendering
- `lukesampson/figlet` - ASCII art text
- `spf13/pflag` - Flag parsing (via cobra)

---

## 6. gh-projects (heaths/gh-projects)

**Repository**: https://github.com/heaths/gh-projects
**Purpose**: Manage GitHub Projects (V2) from CLI
**Language**: Go
**License**: MIT

### Project Structure

```
gh-projects/
├── .github/workflows/
├── internal/                      # Core implementation
├── .editorconfig
├── .gitignore
├── .golangci.yml
├── main.go                        # Entry point with Cobra setup
├── go.mod
└── go.sum
```

### How It Uses go-gh

- `cli/go-gh` for API clients and repository detection
- GraphQL for GitHub Projects V2 API
- Token scope validation via API error handling

### CLI Flags/Args (via Cobra)

```go
func main() {
    rootCmd := &cobra.Command{Use: "gh-projects"}

    // Persistent flags (inherited by all subcommands)
    rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

    // Pre-run: validates auth, checks token scopes
    rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
        // Check authentication
        // Verify required scopes (project)
        // Resolve repository from flag or current directory
    }

    // Subcommands
    rootCmd.AddCommand(cloneCmd)  // --title, --private, --public
    rootCmd.AddCommand(editCmd)   // --description, --add-issue, -f (field assignments)
    rootCmd.AddCommand(listCmd)   // --search
    rootCmd.AddCommand(viewCmd)
}
```

### Error Handling Pattern

```go
var (
    errNotAuthenticated    = errors.New("not authenticated; run: gh auth login -s project")
    errInsufficientScopes  = errors.New("token lacks required scopes")
)

// In PersistentPreRunE:
// Catches GraphQL scope errors and wraps them meaningfully
```

### Output Formatting

- Uses `charmbracelet/glamour` for markdown rendering
- `muesli/reflow` for text wrapping
- Tabular output for list command

### Testing

- `stretchr/testify` for assertions
- `h2non/gock` for HTTP mocking (mocks GitHub API responses)
- `MakeNowJust/heredoc` for readable test fixtures

### Key Dependencies

- `cli/go-gh` - GitHub SDK
- `spf13/cobra` - CLI framework
- `charmbracelet/glamour` - Markdown rendering
- `stretchr/testify` - Testing assertions
- `h2non/gock` - HTTP mocking
- `heaths/go-console` - Console utilities

---

## 7. gh-grep (k1LoW/gh-grep)

**Repository**: https://github.com/k1LoW/gh-grep
**Purpose**: Grep across GitHub repositories from CLI
**Language**: Go (82.6%), Shell (9.3%), Makefile (5.6%), Dockerfile (2.5%)
**License**: MIT

### Project Structure

```
gh-grep/
├── .github/workflows/
├── cmd/
│   └── root.go                    # Cobra root command with extensive flags
├── gh/                            # GitHub API interaction
├── internal/                      # Private packages
├── scanner/                       # Pattern matching logic
├── version/                       # Version management
├── .golangci.yml
├── .goreleaser.yml
├── Dockerfile                     # Container distribution
├── Makefile
├── main.go
├── go.mod
└── go.sum
```

### CLI Flags/Args (via Cobra - extensive)

```go
// cmd/root.go - very comprehensive flag handling
rootCmd.Flags().StringSliceVarP(&matchPatterns, "regexp", "e", nil, "match pattern")
rootCmd.Flags().StringVar(&owner, "owner", "", "repository owner (required)")
rootCmd.Flags().StringVar(&repo, "repo", "", "repository name")
rootCmd.Flags().StringVar(&branch, "branch", "", "branch name")
rootCmd.Flags().StringVar(&tag, "tag", "", "tag name")
rootCmd.Flags().StringSliceVar(&include, "include", nil, "include file pattern")
rootCmd.Flags().StringSliceVar(&exclude, "exclude", nil, "exclude file pattern")
rootCmd.Flags().BoolVarP(&lineNumber, "line-number", "n", false, "show line numbers")
rootCmd.Flags().BoolVarP(&ignoreCase, "ignore-case", "i", false, "case insensitive")
rootCmd.Flags().BoolVarP(&nameOnly, "name-only", "l", false, "show only filenames")
rootCmd.Flags().BoolVar(&repoOnly, "repo-only", false, "show only repo names")
rootCmd.Flags().BoolVar(&url, "url", false, "show GitHub URLs")
rootCmd.Flags().BoolVarP(&count, "count", "c", false, "show match count")
rootCmd.Flags().BoolVarP(&onlyMatching, "only-matching", "o", false, "show only matches")
```

### Output Formatting

- Uses `mattn/go-colorable` for cross-platform colored output
- Results formatted as `repo:filename:line content`
- `--url` flag produces clickable GitHub links with line anchors
- Piping support (e.g., `| xargs open` on macOS)

### Error Handling

- Gracefully continues on `RepoOnlyError` (keeps scanning other repos)
- Exits with code 1 on fatal errors
- Debug logging to stderr when `DEBUG` env var is set

### Build/Release

- **GoReleaser v2** (.goreleaser.yml):
  - Pre-build: `go mod download && go mod tidy`
  - Three separate build configs (Linux, macOS, Windows)
  - CGO disabled
  - Version metadata via ldflags
  - Linux packages: APK, DEB, RPM
  - Archive includes LICENSE, README, CHANGELOG
  - Checksum generation
- **Docker** distribution via `ghcr.io/k1low/gh-grep`
- **Makefile** for local development tasks

---

## 8. gh-workflow-stats (fchimpan/gh-workflow-stats)

**Repository**: https://github.com/fchimpan/gh-workflow-stats
**Purpose**: Calculate success rate and execution time of GitHub Actions workflows
**Language**: Go
**License**: MIT

### Project Structure

```
gh-workflow-stats/
├── .github/workflows/
├── cmd/
│   └── root.go                    # Cobra root with extensive flags
├── internal/                      # Core logic
├── sample/                        # Example output (text + JSON)
├── main.go
├── go.mod
└── go.sum
```

### CLI Flags/Args (via Cobra)

```go
rootCmd := &cobra.Command{
    Use:   "workflow-stats",
    Short: "Fetch workflow runs stats",
    RunE: func(cmd *cobra.Command, args []string) error {
        // Validate required flags, construct config, run
    },
}

// Organization/repo targeting
rootCmd.Flags().StringVarP(&org, "org", "o", "", "GitHub organization")
rootCmd.Flags().StringVarP(&repo, "repo", "r", "", "repository name")

// Workflow identification
rootCmd.Flags().StringVarP(&file, "file", "f", "", "workflow filename/ID")
rootCmd.Flags().IntVarP(&id, "id", "i", 0, "workflow numeric ID")

// Filters
rootCmd.Flags().StringVarP(&actor, "actor", "a", "", "filter by user")
rootCmd.Flags().StringVarP(&branch, "branch", "b", "", "filter by branch")
rootCmd.Flags().StringVarP(&event, "event", "e", "", "filter by trigger event")
rootCmd.Flags().StringVarP(&status, "status", "s", "", "completion status")
rootCmd.Flags().StringVarP(&created, "created", "c", "", "date range")

// Output control
rootCmd.Flags().BoolVar(&jsonOutput, "json", false, "JSON output")
rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "verbose logging")
rootCmd.Flags().BoolVarP(&debug, "debug", "d", false, "debug logging")
rootCmd.Flags().BoolVarP(&all, "all", "A", false, "fetch all results")
rootCmd.Flags().StringVarP(&host, "host", "H", "", "GitHub Enterprise host")
```

### --json Flag Pattern

This extension implements the `--json` flag as a simple boolean:

```go
// When --json is set, output JSON instead of human-readable tables
if jsonOutput {
    enc := json.NewEncoder(os.Stdout)
    enc.SetIndent("", "  ")
    enc.Encode(stats)
} else {
    // Render human-readable table with success rates, durations, etc.
}
```

### Error Handling

```go
func Execute() {
    err := rootCmd.Execute()
    if err != nil {
        os.Exit(1)
    }
}
```

### Sample Outputs

The `sample/` directory contains both human-readable and JSON format examples for reference.

---

## 9. gh-actions-usage (codiform/gh-actions-usage)

**Repository**: https://github.com/codiform/gh-actions-usage
**Purpose**: Display GitHub Actions usage metrics
**Language**: Go (100%)

### Project Structure

```
gh-actions-usage/
├── .github/workflows/
├── client/                        # GitHub API client abstraction
├── format/                        # Output formatters (interface-based)
├── main.go                        # Entry point with flag parsing
├── go.mod
└── go.sum
```

### Architecture Pattern (Interface-Driven)

```go
// format/formatter.go - output format interface
type Formatter interface {
    Format(data []RepoUsage) error
}

// Implementations: HumanFormatter (table), TSVFormatter
// Selected at runtime based on --output flag
```

### CLI Flags (standard library)

```go
// Uses Go's built-in flag package (no cobra)
flag.BoolVar(&cfg.skip, "skip", false, "skip repos with no workflows")
flag.StringVar(&cfg.output, "output", "human", "output format: human|tsv")
flag.Parse()

// Positional args for targets: owner/repo, @user, @org
```

### Error Handling

```go
// Custom error types for semantic errors
type UnknownRepoError struct{ Name string }
type UnknownUserError struct{ Name string }

func (e *UnknownRepoError) Error() string {
    return fmt.Sprintf("unknown repository: %s", e.Name)
}
```

### Key Pattern: Target Resolution

Flexible target parsing that normalizes:
- `owner/repo` - single repository
- `@username` - all user's repos
- `@orgname` - all org's repos

Uses `repoMap` type for hierarchical organization of results by owner.

---

## 10. gh-branch (mislav/gh-branch)

**Repository**: https://github.com/mislav/gh-branch
**Purpose**: Interactive branch switcher with fuzzy finding
**Language**: Shell (100%)
**License**: Unlicense

### Project Structure

```
gh-branch/
├── gh-branch                      # Single executable shell script
├── readme.md
└── LICENSE
```

### How It Works

- Pure shell script, no compilation needed
- Depends on `fzf` for fuzzy finding
- Lists local branches in relation to PRs in the repo
- Uses `gh` commands internally for GitHub data

### Key Takeaway

This demonstrates that simple extensions can be shell scripts. No build process, no goreleaser, no go.mod. Just a single executable file named `gh-EXTENSION-NAME`.

---

## 11. gh-copilot (github/gh-copilot)

**Repository**: https://github.com/github/gh-copilot
**Status**: ARCHIVED (October 2025), replaced by https://github.com/github/copilot-cli
**Purpose**: AI-powered command suggestions and explanations

### Key Observations

- Closed-source: Repository only contained README and CODE_OF_CONDUCT.md
- Distributed as **precompiled binaries** via GitHub Releases (14 releases)
- Installed via `gh extension install github/gh-copilot --force`
- Required OAuth authentication (not PAT-compatible)
- Cross-platform support (excluded 32-bit Android)

### Takeaway

Even GitHub's official extensions use the same precompiled binary distribution mechanism. Extensions can be closed-source -- only the compiled binaries need to be published.

---

## 12. Default Scaffold from `gh extension create`

When you run `gh extension create --precompiled=go EXTENSION-NAME`, the following is generated:

### Generated Structure

```
gh-EXTENSION-NAME/
├── .github/
│   └── workflows/
│       └── release.yml            # Automated release workflow
├── main.go                        # Starter Go code
├── go.mod                         # Module definition
├── go.sum                         # (after go mod tidy)
└── .gitignore
```

### Generated main.go Template

```go
package main

import (
    "fmt"
    "github.com/cli/go-gh/v2"
    "github.com/cli/go-gh/v2/pkg/api"
)

func main() {
    fmt.Println("hi world, this is the gh-EXTENSION-NAME extension!")

    client, err := api.DefaultRESTClient()
    if err != nil {
        fmt.Println(err)
        return
    }

    response := struct{ Login string }{}
    err = client.Get("user", &response)
    if err != nil {
        fmt.Println(err)
        return
    }

    fmt.Println(response.Login)
}
```

### Generated release.yml Workflow

```yaml
name: release

on:
  push:
    tags:
      - "v*"

permissions:
  contents: write

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: cli/gh-extension-precompile@v2
        with:
          go_version_file: go.mod
```

### What Is NOT Generated

The default scaffold is minimal. It does NOT include:
- Cobra CLI framework setup
- Testing infrastructure
- .goreleaser.yml (uses gh-extension-precompile instead)
- Configuration file handling
- README.md (you create this yourself)
- Makefile or Taskfile

### Note on Cobra

There is an open feature request (cli/cli#7774) to add a Cobra-based template option to `gh extension create`, but as of early 2026 the GitHub team has not implemented it, citing reluctance to "play favorites" among CLI libraries.

---

## 13. gh-extension-precompile GitHub Action

**Repository**: https://github.com/cli/gh-extension-precompile
**Purpose**: Automated cross-compilation and release for Go-based gh extensions

### How It Works

1. Triggers on git tag push (e.g., `v1.0.0`)
2. Cross-compiles Go binaries for all supported platforms
3. Uploads binaries to GitHub Release
4. Follows the naming convention: `gh-NAME-OS-ARCH[.exe]`
5. Tags with hyphens (e.g., `v2.0.0-rc.1`) create prereleases

### Action Inputs

| Input | Description | Default |
|-------|-------------|---------|
| `go_version` | Go version (semver) | auto from go.mod |
| `go_version_file` | Path to go.mod/go.work | - |
| `go_build_options` | Extra build flags | - |
| `build_script_override` | Custom build script | - |
| `draft_release` | Create as draft | false |
| `github_token` | Auth token | `github.token` |
| `release_tag` | Override tag | from git ref |
| `gpg_fingerprint` | GPG signing key | - |
| `generate_attestations` | Build provenance | false |
| `release_android` | Build for Android | false |

### Basic Usage

```yaml
- uses: cli/gh-extension-precompile@v2
  with:
    go_version_file: go.mod
```

### vs. GoReleaser

| Aspect | gh-extension-precompile | GoReleaser |
|--------|------------------------|------------|
| Setup complexity | Minimal (2 lines) | Full YAML config |
| Customization | Limited | Extensive |
| Package formats | Binaries only | DEB, RPM, APK, Docker, Homebrew |
| Changelog | Basic GitHub | Conventional commits, categories |
| Announcements | None | Discord, Slack, Bluesky, etc. |
| Signing | GPG optional | GPG, cosign |
| Best for | Simple extensions | Complex projects (gh-dash) |

---

## 14. Cross-Cutting Patterns & Best Practices

### Pattern 1: Minimal main.go, Logic in cmd/ or internal/

Every well-structured extension follows this pattern:

```go
// main.go
package main

import "github.com/owner/gh-ext/cmd"

func main() {
    cmd.Execute()
}
```

The actual CLI setup, flag parsing, and business logic live in `cmd/root.go` and `internal/` packages.

### Pattern 2: Cobra for Multi-Command Extensions

Most extensions beyond trivial ones use `spf13/cobra`:

```go
// cmd/root.go
var rootCmd = &cobra.Command{
    Use:   "gh ext-name",
    Short: "One-line description",
    RunE: func(cmd *cobra.Command, args []string) error {
        // main logic or show help
    },
}

func init() {
    rootCmd.PersistentFlags().StringVar(&host, "host", "", "GitHub host")
    rootCmd.Flags().BoolVar(&jsonOutput, "json", false, "output as JSON")
}

func Execute() {
    if err := rootCmd.Execute(); err != nil {
        os.Exit(1)
    }
}
```

Simple extensions (single function) may use Go's standard `flag` package instead.

### Pattern 3: Terminal-Aware Output

```go
t := term.FromEnv()
if t.IsTerminalOutput() {
    // Human-readable: tables, colors, progress bars
    w, _, _ := t.Size()
    printer := tableprinter.New(t.Out(), true, w)
    // ... add fields with colors
} else {
    // Machine-readable: TSV, JSON, plain text
    printer := tableprinter.New(t.Out(), false, 0)
    // ... no colors
}
```

### Pattern 4: The --json Flag

The `gh` CLI convention for `--json`:

**Simple boolean approach** (gh-workflow-stats):
```go
if jsonOutput {
    json.NewEncoder(os.Stdout).Encode(data)
} else {
    renderTable(data)
}
```

**Field-selector approach** (gh CLI core):
```go
// gh pr list --json title,url,state
// --json takes comma-separated field names
// Combined with --jq or --template for further processing
```

For extensions, the simple boolean approach is most common.

### Pattern 5: Error Handling

```go
// Pattern A: Cobra RunE (preferred)
RunE: func(cmd *cobra.Command, args []string) error {
    if err := doWork(); err != nil {
        return err  // Cobra handles the error display
    }
    return nil
}

// Pattern B: Named errors for specific conditions
var (
    errNotAuthenticated = errors.New("not authenticated; run: gh auth login")
    errNoRepository     = errors.New("could not determine repository")
)

// Pattern C: GitHub API error handling
if httpErr, ok := err.(*api.HTTPError); ok {
    switch httpErr.StatusCode {
    case 404:
        return fmt.Errorf("repository not found: %s", repoName)
    case 403:
        return fmt.Errorf("insufficient permissions")
    }
}

// Pattern D: Exit code 1 on error
func Execute() {
    if err := rootCmd.Execute(); err != nil {
        os.Exit(1)
    }
}
```

### Pattern 6: Repository Resolution

```go
// From go-gh (automatic)
repo, err := repository.Current()

// From flag (manual override)
if repoFlag != "" {
    repo, err = repository.Parse(repoFlag)
}

// Typical flag:
rootCmd.Flags().StringVarP(&repoFlag, "repo", "R", "", "repository in OWNER/REPO format")
```

### Pattern 7: Build/Release Process

Two main approaches:

**Approach A: gh-extension-precompile** (simpler, recommended for most extensions)
```yaml
# .github/workflows/release.yml
on:
  push:
    tags: ["v*"]
permissions:
  contents: write
jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: cli/gh-extension-precompile@v2
        with:
          go_version_file: go.mod
```

**Approach B: GoReleaser** (more control, for complex projects)
```yaml
# .goreleaser.yaml
version: 2
builds:
  - env: [CGO_ENABLED=0]
    goos: [linux, darwin, windows]
    goarch: [amd64, arm64]
    ldflags:
      - -s -w -X main.version={{.Version}}
archives:
  - format: binary
    name_template: "gh-ext_{{ .Tag }}_{{ .Os }}-{{ .Arch }}"
release:
  draft: true  # Review before publishing
```

### Pattern 8: Testing Approaches

| Extension | Testing Approach | Libraries |
|-----------|-----------------|-----------|
| gh-dash | Standard go test + testdata fixtures | go test |
| gh-poi | E2E tests with stdout capture + mocks | golang/mock, testify |
| gh-projects | HTTP mocking + test assertions | gock, testify, heredoc |
| gh-grep | Unit tests + integration | go test |
| gh-workflow-stats | Sample output validation | go test |

**Common testing patterns**:

```go
// 1. HTTP mocking (for API tests)
func TestListRepos(t *testing.T) {
    defer gock.Off()
    gock.New("https://api.github.com").
        Get("/repos/owner/repo").
        Reply(200).
        JSON(map[string]string{"name": "repo"})

    // run your function that calls the API
}

// 2. Prompter mocking (for interactive tests)
mock := prompter.NewMock(t)
mock.RegisterConfirm("Delete branch?", func(p string, d bool) (bool, error) {
    return true, nil
})

// 3. Output capture (for CLI output tests)
var buf bytes.Buffer
cmd.SetOut(&buf)
cmd.Execute()
assert.Contains(t, buf.String(), "expected output")

// 4. Test fixtures
// Place in testdata/ directory (convention)
data, _ := os.ReadFile("testdata/response.json")
```

### Pattern 9: Configuration

Extensions that need user configuration typically use:

```go
// Option A: gh's config system
cfg, err := config.Read(nil)
val, err := cfg.Get([]string{"extensions", "my-ext", "setting"})

// Option B: Dedicated config file (gh-dash approach)
// ~/.config/gh-dash/config.yml
// Uses koanf, viper, or manual YAML parsing

// Option C: Environment variables
token := os.Getenv("GH_TOKEN")
```

### Pattern 10: Version Information

```go
// Set at build time via ldflags
var (
    version = "dev"
    commit  = "none"
    date    = "unknown"
)

// In goreleaser.yaml:
ldflags:
  - -s -w
  - -X main.version={{.Version}}
  - -X main.commit={{.Commit}}
  - -X main.date={{.Date}}

// Display:
rootCmd.Version = fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date)
```

---

## 15. Recommended Architecture for New Extensions

Based on analyzing all the extensions above, here is the recommended project structure for a new Go-based GitHub CLI extension:

### Minimal Extension (single command)

```
gh-my-ext/
├── .github/
│   └── workflows/
│       └── release.yml            # gh-extension-precompile
├── main.go                        # Entry point -> cmd.Execute()
├── cmd/
│   └── root.go                    # Cobra root command + flags
├── internal/
│   └── ...                        # Business logic packages
├── .gitignore
├── .golangci.yml                  # Optional: linting rules
├── go.mod
├── go.sum
├── LICENSE
└── README.md
```

### Complex Extension (multiple commands)

```
gh-my-ext/
├── .github/
│   └── workflows/
│       ├── ci.yml                 # Build + test + lint
│       └── release.yml            # GoReleaser or precompile
├── main.go                        # Entry point
├── cmd/
│   ├── root.go                    # Root command + global flags
│   ├── list.go                    # Subcommand
│   ├── view.go                    # Subcommand
│   └── create.go                  # Subcommand
├── internal/
│   ├── api/                       # GitHub API wrappers
│   ├── config/                    # Extension configuration
│   ├── output/                    # Formatters (table, JSON)
│   └── ...                        # Domain logic
├── testdata/                      # Test fixtures
├── .gitignore
├── .golangci.yml
├── .goreleaser.yaml               # If using GoReleaser
├── go.mod
├── go.sum
├── LICENSE
└── README.md
```

### Essential go.mod Dependencies

```
module github.com/owner/gh-my-ext

go 1.23

require (
    github.com/cli/go-gh/v2 v2.13.0     // Core SDK (API, auth, term, tableprinter)
    github.com/spf13/cobra v1.8.0         // CLI framework
)

// Optional, as needed:
require (
    github.com/charmbracelet/bubbletea    // TUI framework
    github.com/charmbracelet/lipgloss     // TUI styling
    github.com/stretchr/testify           // Testing assertions
    github.com/h2non/gock                 // HTTP mocking for tests
)
```

### Minimum Viable main.go

```go
package main

import "github.com/owner/gh-my-ext/cmd"

func main() {
    cmd.Execute()
}
```

### Minimum Viable cmd/root.go

```go
package cmd

import (
    "encoding/json"
    "fmt"
    "os"

    "github.com/cli/go-gh/v2/pkg/api"
    "github.com/cli/go-gh/v2/pkg/tableprinter"
    "github.com/cli/go-gh/v2/pkg/term"
    "github.com/spf13/cobra"
)

var (
    version    = "dev"
    jsonOutput bool
    repoFlag   string
)

var rootCmd = &cobra.Command{
    Use:     "gh my-ext",
    Short:   "One-line description of the extension",
    Version: version,
    RunE:    run,
}

func init() {
    rootCmd.Flags().BoolVar(&jsonOutput, "json", false, "output as JSON")
    rootCmd.Flags().StringVarP(&repoFlag, "repo", "R", "", "repository (OWNER/REPO)")
}

func Execute() {
    if err := rootCmd.Execute(); err != nil {
        os.Exit(1)
    }
}

func run(cmd *cobra.Command, args []string) error {
    client, err := api.DefaultRESTClient()
    if err != nil {
        return fmt.Errorf("failed to create API client: %w", err)
    }

    // Fetch data...
    var data []Item
    if err := client.Get("repos/...", &data); err != nil {
        if httpErr, ok := err.(*api.HTTPError); ok {
            return fmt.Errorf("API error %d: %s", httpErr.StatusCode, httpErr.Message)
        }
        return err
    }

    // Output
    if jsonOutput {
        enc := json.NewEncoder(os.Stdout)
        enc.SetIndent("", "  ")
        return enc.Encode(data)
    }

    t := term.FromEnv()
    w, _, _ := t.Size()
    printer := tableprinter.New(t.Out(), t.IsTerminalOutput(), w)
    for _, item := range data {
        printer.AddField(item.Name)
        printer.AddField(item.Status)
        printer.EndRow()
    }
    return printer.Render()
}
```

### Minimum Viable Release Workflow

```yaml
# .github/workflows/release.yml
name: release
on:
  push:
    tags: ["v*"]
permissions:
  contents: write
jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: cli/gh-extension-precompile@v2
        with:
          go_version_file: go.mod
```

---

## Sources

### Repositories Analyzed
- https://github.com/cli/go-gh (official Go SDK, v2.13.0)
- https://github.com/dlvhdr/gh-dash (10.2k stars, TUI dashboard)
- https://github.com/seachicken/gh-poi (branch cleanup)
- https://github.com/vilmibm/gh-screensaver (terminal screensavers)
- https://github.com/heaths/gh-projects (GitHub Projects V2)
- https://github.com/k1LoW/gh-grep (repository grep)
- https://github.com/fchimpan/gh-workflow-stats (workflow statistics)
- https://github.com/codiform/gh-actions-usage (actions usage metrics)
- https://github.com/rsese/gh-actions-status (actions health)
- https://github.com/mislav/gh-branch (interactive branch switcher)
- https://github.com/github/gh-copilot (archived, closed-source)
- https://github.com/cli/gh-extension-precompile (build/release action)

### Documentation
- https://docs.github.com/en/github-cli/github-cli/creating-github-cli-extensions
- https://pkg.go.dev/github.com/cli/go-gh/v2
- https://pkg.go.dev/github.com/cli/go-gh/v2/pkg/api
- https://pkg.go.dev/github.com/cli/go-gh/v2/pkg/tableprinter
- https://pkg.go.dev/github.com/cli/go-gh/v2/pkg/term
- https://pkg.go.dev/github.com/cli/go-gh/v2/pkg/auth
- https://pkg.go.dev/github.com/cli/go-gh/v2/pkg/config
- https://pkg.go.dev/github.com/cli/go-gh/v2/pkg/repository
- https://pkg.go.dev/github.com/cli/go-gh/v2/pkg/prompter
- https://pkg.go.dev/github.com/cli/go-gh/v2/pkg/jq

### Articles
- https://mikeball.info/blog/extending-the-gh-cli-with-go/
- https://meiji163.github.io/post/gh-extension/
- https://www.infoq.com/news/2023/01/GitHub-simplifies-cli-extension/
- https://www.gitkraken.com/blog/8-github-cli-extensions-2024
- https://github.com/cli/cli/issues/7774 (Cobra template discussion)
