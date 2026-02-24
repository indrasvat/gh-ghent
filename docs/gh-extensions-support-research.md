# GitHub CLI Extensions: Comprehensive Research Document

> **Last updated:** 2026-02-21
> **Purpose:** Complete reference for building a Go-based GitHub CLI extension
> **Sources:** GitHub Docs, cli/go-gh repository, pkg.go.dev, gh-extension-precompile, popular extension source code analysis, and community best practices

---

## Table of Contents

1. [Extension Naming Conventions](#1-extension-naming-conventions)
2. [Extension Types](#2-extension-types)
3. [Precompiled Binary Distribution & Release Workflow](#3-precompiled-binary-distribution--release-workflow)
4. [The go-gh Library (github.com/cli/go-gh/v2)](#4-the-go-gh-library)
5. [How Extensions Access the GitHub API](#5-how-extensions-access-the-github-api)
6. [Authentication Inheritance](#6-authentication-inheritance)
7. [Extension Distribution and Installation](#7-extension-distribution-and-installation)
8. [Extension Upgrade Mechanism](#8-extension-upgrade-mechanism)
9. [CLI Flags, Arguments, and Output Formatting](#9-cli-flags-arguments-and-output-formatting)
10. [JSON Output for Machine Readability](#10-json-output-for-machine-readability)
11. [Error Handling](#11-error-handling)
12. [GitHub GraphQL API for PR Operations](#12-github-graphql-api-for-pr-operations)
13. [Popular Extensions and Architecture Patterns](#13-popular-extensions-and-architecture-patterns)
14. [Testing Strategies](#14-testing-strategies)
15. [Project Structure and Scaffolding](#15-project-structure-and-scaffolding)
16. [Complete Code Examples](#16-complete-code-examples)

---

## 1. Extension Naming Conventions

### Repository Naming

- **Mandatory prefix:** All extension repositories MUST be named with the `gh-` prefix.
  - Example: `gh-whoami`, `gh-dash`, `gh-poi`
- The repository name directly determines the command name: `gh-whoami` becomes `gh whoami`.
- The executable file at the repository root MUST have the same name as the repository directory.

### Command Naming

- When installed, the `gh-` prefix is stripped: `gh extension install owner/gh-whoami` results in `gh whoami`.
- You cannot have two extensions with identical names installed simultaneously.
- Names should be short, descriptive, and use hyphens for multi-word names (e.g., `gh-markdown-preview`).

### Discoverability

- Add the `gh-extension` topic to your GitHub repository for discovery on `https://github.com/topics/gh-extension`.
- As of 2026, there are 760+ public repositories tagged with this topic.

---

## 2. Extension Types

GitHub CLI supports three types of extensions:

### 2.1 Interpreted Extensions (Shell Scripts)

- Simplest to create; recommended for quick tools.
- Uses bash (or any available interpreter) with a shebang line.
- The executable file at the repository root must be marked executable (`chmod +x`).
- Users must have the interpreter installed.

**Structure:**
```
gh-EXTENSION-NAME/
  gh-EXTENSION-NAME    # Executable script (same name as directory)
```

**Example:**
```bash
#!/usr/bin/env bash
set -e
exec gh api user --jq '"You are @\(.login) (\(.name))"'
```

### 2.2 Precompiled Go Extensions (Recommended for Production)

- Use `gh extension create --precompiled=go EXTENSION-NAME` for scaffolding.
- Leverages the `go-gh` library for GitHub CLI integration.
- Produces cross-platform binaries distributed via GitHub Releases.
- GitHub Actions workflow scaffolding included automatically.
- CGO is disabled by default for maximum portability.

**Advantages:**
- No runtime dependencies for end users.
- Cross-platform support (Linux, macOS, Windows, ARM, AMD64).
- Access to the full go-gh SDK for API, auth, terminal, and formatting.
- The gh CLI itself is written in Go, so extensions feel native.

### 2.3 Precompiled Non-Go Extensions

- Use `gh extension create --precompiled=other EXTENSION-NAME`.
- Requires a `script/build.sh` for automated builds.
- Developer provides their own compilation logic.
- Can use any compiled language (Rust, C#, etc.).

---

## 3. Precompiled Binary Distribution & Release Workflow

### Binary Naming Convention

Binaries attached to GitHub Releases MUST follow this naming pattern:

```
{extension-name}-{os}-{arch}[.exe]
```

**Examples:**
```
gh-whoami-linux-amd64
gh-whoami-linux-arm64
gh-whoami-darwin-amd64
gh-whoami-darwin-arm64
gh-whoami-windows-amd64.exe
gh-whoami-windows-arm64.exe
```

When using `gh-extension-precompile`, the naming includes version:
```
gh-whoami_v1.0.0_linux-amd64
gh-whoami_v1.0.0_darwin-arm64
gh-whoami_v1.0.0_windows-amd64.exe
```

### Manual Release Process

```bash
# Tag the release
git tag v1.0.0
git push origin v1.0.0

# Cross-compile
GOOS=windows GOARCH=amd64 go build -o gh-EXTENSION-NAME-windows-amd64.exe
GOOS=linux   GOARCH=amd64 go build -o gh-EXTENSION-NAME-linux-amd64
GOOS=darwin  GOARCH=amd64 go build -o gh-EXTENSION-NAME-darwin-amd64
GOOS=darwin  GOARCH=arm64 go build -o gh-EXTENSION-NAME-darwin-arm64

# Create release with all binaries
gh release create v1.0.0 ./*amd64* ./*arm64*
```

**Important:** Do NOT commit compiled binaries to version control.

### Automated Release with gh-extension-precompile

The `cli/gh-extension-precompile` GitHub Action automates everything. This is the **recommended approach**.

**Minimal workflow (`.github/workflows/release.yml`):**
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

**Features of gh-extension-precompile:**
- Automatically cross-compiles for all Go-supported platforms.
- CGO disabled by default for portability.
- Supports custom build flags: `go_build_options: '-tags production'`.
- Supports custom build scripts: `build_script_override: "script/build.sh"`.
- Prerelease tags (e.g., `v2.0.0-rc.1`) create prerelease versions.
- Optional GPG signing and checksums.
- Optional artifact attestations for supply chain security.
- Android build support (disabled by default).

**Advanced workflow with GPG signing:**
```yaml
steps:
  - uses: actions/checkout@v4
  - id: import_gpg
    uses: crazy-max/ghaction-import-gpg@v6
    with:
      gpg_private_key: ${{ secrets.GPG_PRIVATE_KEY }}
      passphrase: ${{ secrets.GPG_PASSPHRASE }}
  - uses: cli/gh-extension-precompile@v2
    with:
      gpg_fingerprint: ${{ steps.import_gpg.outputs.fingerprint }}
```

**With attestations:**
```yaml
permissions:
  contents: write
  id-token: write
  attestations: write
steps:
  - uses: cli/gh-extension-precompile@v2
    with:
      generate_attestations: true
```

### Note on manifest.yml

There is **no manifest.yml** file format for gh extensions. Extensions are defined entirely by:
1. Repository naming (`gh-` prefix).
2. Executable presence at root (for script extensions).
3. Binary assets attached to GitHub Releases (for precompiled extensions).

The `gh` CLI auto-detects the correct binary for the user's OS/architecture from release assets.

---

## 4. The go-gh Library

**Module:** `github.com/cli/go-gh/v2`
**Latest version:** v2.13.0 (November 2025)
**License:** MIT
**Documentation:** https://pkg.go.dev/github.com/cli/go-gh/v2

The go-gh library is the official SDK for building Go-based GitHub CLI extensions. It exposes portions of the gh CLI's own code.

### 4.1 Package Overview

| Package | Import Path | Purpose |
|---------|------------|---------|
| `gh` | `github.com/cli/go-gh/v2` | Core: Exec, ExecContext, ExecInteractive, Path |
| `api` | `github.com/cli/go-gh/v2/pkg/api` | REST, GraphQL, and HTTP clients |
| `repository` | `github.com/cli/go-gh/v2/pkg/repository` | Repository detection and parsing |
| `tableprinter` | `github.com/cli/go-gh/v2/pkg/tableprinter` | Terminal table formatting |
| `term` | `github.com/cli/go-gh/v2/pkg/term` | Terminal capability detection |
| `auth` | `github.com/cli/go-gh/v2/pkg/auth` | Authentication token retrieval |
| `config` | `github.com/cli/go-gh/v2/pkg/config` | gh configuration file access |
| `browser` | `github.com/cli/go-gh/v2/pkg/browser` | Open URLs in user's browser |
| `prompter` | `github.com/cli/go-gh/v2/pkg/prompter` | Interactive user input prompts |
| `jq` | `github.com/cli/go-gh/v2/pkg/jq` | jq expression processing |
| `jsonpretty` | `github.com/cli/go-gh/v2/pkg/jsonpretty` | Pretty-print JSON with colors |
| `markdown` | `github.com/cli/go-gh/v2/pkg/markdown` | Render markdown in terminal |
| `template` | `github.com/cli/go-gh/v2/pkg/template` | Go template processing for JSON |
| `text` | `github.com/cli/go-gh/v2/pkg/text` | Text processing utilities |
| `ssh` | `github.com/cli/go-gh/v2/pkg/ssh` | SSH hostname alias resolution |
| `asciisanitizer` | `github.com/cli/go-gh/v2/pkg/asciisanitizer` | UTF-8 control character sanitization |

**Experimental packages:**
| Package | Purpose |
|---------|---------|
| `pkg/x/color` | Experimental accessible color rendering |
| `pkg/x/markdown` | Experimental accessible markdown rendering |

### 4.2 Environment Variables

The go-gh library respects these environment variables (same as gh CLI):

| Variable | Purpose |
|----------|---------|
| `GH_TOKEN` | Override authentication token |
| `GH_HOST` | Override target GitHub host |
| `GH_REPO` | Override current repository detection |
| `GH_FORCE_TTY` | Force terminal mode even when not a TTY |
| `NO_COLOR` | Disable color output |
| `CLICOLOR` | Control color output |
| `GH_DEBUG` | Enable debug logging for API requests |

### 4.3 Core Functions (gh package)

```go
// Shell out to a gh command and capture output
func Exec(args ...string) (stdout, stderr bytes.Buffer, err error)

// Context-aware version with cancellation/timeout
func ExecContext(ctx context.Context, args ...string) (stdout, stderr bytes.Buffer, err error)

// Interactive execution with stdin/stdout/stderr connected to parent
func ExecInteractive(ctx context.Context, args ...string) error

// Find the gh executable path
func Path() (string, error)  // Added in v2.1.0
```

### 4.4 API Package (pkg/api)

#### RESTClient

```go
// Create default REST client (uses gh auth)
func DefaultRESTClient() (*RESTClient, error)

// Create REST client with custom options
func NewRESTClient(opts ClientOptions) (*RESTClient, error)
```

**RESTClient Methods:**

| Method | Signature | Description |
|--------|-----------|-------------|
| `Get` | `(path string, resp interface{}) error` | GET request |
| `Post` | `(path string, body io.Reader, resp interface{}) error` | POST request |
| `Patch` | `(path string, body io.Reader, resp interface{}) error` | PATCH request |
| `Put` | `(path string, body io.Reader, resp interface{}) error` | PUT request |
| `Delete` | `(path string, resp interface{}) error` | DELETE request |
| `Do` | `(method, path string, body io.Reader, resp interface{}) error` | Generic request |
| `DoWithContext` | `(ctx context.Context, method, path string, body io.Reader, resp interface{}) error` | Context-aware generic request |
| `Request` | `(method, path string, body io.Reader) (*http.Response, error)` | Raw HTTP response |
| `RequestWithContext` | `(ctx context.Context, method, path string, body io.Reader) (*http.Response, error)` | Context-aware raw response |

#### GraphQLClient

```go
// Create default GraphQL client (uses gh auth)
func DefaultGraphQLClient() (*GraphQLClient, error)

// Create GraphQL client with custom options
func NewGraphQLClient(opts ClientOptions) (*GraphQLClient, error)
```

**GraphQLClient Methods:**

| Method | Signature | Description |
|--------|-----------|-------------|
| `Do` | `(query string, variables map[string]interface{}, response interface{}) error` | Execute raw GraphQL query |
| `DoWithContext` | `(ctx context.Context, query string, variables map[string]interface{}, response interface{}) error` | Context-aware raw query |
| `Query` | `(name string, q interface{}, variables map[string]interface{}) error` | Execute query from struct tags |
| `QueryWithContext` | `(ctx context.Context, name string, q interface{}, variables map[string]interface{}) error` | Context-aware struct query |
| `Mutate` | `(name string, m interface{}, variables map[string]interface{}) error` | Execute mutation from struct tags |
| `MutateWithContext` | `(ctx context.Context, name string, m interface{}, variables map[string]interface{}) error` | Context-aware struct mutation |

#### ClientOptions

```go
type ClientOptions struct {
    AuthToken          string            // Override auth token
    CacheDir           string            // Cache directory (default: gh cache dir)
    CacheTTL           time.Duration     // Cache TTL (default: 24h)
    EnableCache        bool              // Enable caching (default: false)
    Headers            map[string]string // Custom HTTP headers
    Host               string            // Target API host
    Log                io.Writer         // API request log writer
    LogIgnoreEnv       bool              // Ignore GH_DEBUG env var
    LogColorize        bool              // Colorize log output
    LogVerboseHTTP     bool              // Log HTTP headers and bodies
    SkipDefaultHeaders bool              // Skip Accept, Content-Type, etc.
    Timeout            time.Duration     // Per-request timeout
    Transport          http.RoundTripper // Custom HTTP transport
    UnixDomainSocket   string            // Unix domain socket
}
```

**Default headers** (set unless SkipDefaultHeaders is true):
- `Accept`
- `Content-Type`
- `Time-Zone`
- `User-Agent`

**Security:** Auth tokens are only added to requests when `opts.Host` matches the request host.

#### HTTP Client

```go
func DefaultHTTPClient() (*http.Client, error)
func NewHTTPClient(opts ClientOptions) (*http.Client, error)
```

#### Error Types

```go
// REST API errors
type HTTPError struct {
    Errors     []HTTPErrorItem
    Headers    http.Header
    Message    string
    RequestURL *url.URL
    StatusCode int
}

type HTTPErrorItem struct {
    Code     string
    Field    string
    Message  string
    Resource string
}

// GraphQL API errors
type GraphQLError struct {
    Errors []GraphQLErrorItem
}

type GraphQLErrorItem struct {
    Message    string
    Locations  []struct{ Line, Column int }
    Path       []interface{}
    Extensions map[string]interface{}
    Type       string
}

// Match checks if error is about a specific type on a specific path
func (gr *GraphQLError) Match(expectType, expectPath string) bool

// Parse HTTP response into HTTPError
func HandleHTTPError(resp *http.Response) error
```

### 4.5 Repository Package (pkg/repository)

```go
type Repository struct {
    Host  string
    Name  string
    Owner string
}

// Detect current repository from git remotes (respects GH_REPO)
func Current() (Repository, error)

// Parse "OWNER/REPO", "HOST/OWNER/REPO", or full URL
func Parse(s string) (Repository, error)

// Parse with explicit host fallback
func ParseWithHost(s, host string) (Repository, error)
```

### 4.6 TablePrinter Package (pkg/tableprinter)

```go
type TablePrinter interface {
    AddHeader(columns []string, opts ...fieldOption)
    AddField(text string, opts ...fieldOption)
    EndRow()
    Render() error
}

// Create table printer
// isTTY=true: human-readable columns with color
// isTTY=false: tab-separated values (TSV), no truncation
func New(w io.Writer, isTTY bool, maxWidth int) TablePrinter

// Field options (applied in order: truncate -> pad -> color)
func WithTruncate(fn func(int, string) string) fieldOption  // nil = disable
func WithPadding(fn func(int, string) string) fieldOption    // Added v2.5.0
func WithColor(fn func(string) string) fieldOption           // ANSI escape codes
```

**Usage pattern:**
```go
terminal := term.FromEnv()
termWidth, _, _ := terminal.Size()
isTTY := terminal.IsTerminalOutput()

t := tableprinter.New(terminal.Out(), isTTY, termWidth)
t.AddHeader([]string{"NUMBER", "TITLE", "STATUS"})

red := func(s string) string { return "\x1b[31m" + s + "\x1b[m" }
green := func(s string) string { return "\x1b[32m" + s + "\x1b[m" }

t.AddField("#42", tableprinter.WithTruncate(nil))
t.AddField("Fix the bug")
t.AddField("open", tableprinter.WithColor(green))
t.EndRow()

if err := t.Render(); err != nil {
    log.Fatal(err)
}
```

### 4.7 Term Package (pkg/term)

Used to detect terminal capabilities:

```go
terminal := term.FromEnv()

// Check if output is a terminal (for color/formatting decisions)
isTTY := terminal.IsTerminalOutput()

// Get terminal dimensions
width, height, err := terminal.Size()

// Get output writer
out := terminal.Out()
errOut := terminal.ErrOut()
```

Respects `GH_FORCE_TTY`, `NO_COLOR`, and `CLICOLOR` environment variables.

---

## 5. How Extensions Access the GitHub API

Extensions have three primary methods to access the GitHub API:

### 5.1 Via gh.Exec() (Shell Out)

Shell out to `gh api` subcommand. Simple but creates a subprocess.

```go
// REST API call
args := []string{"api", "repos/owner/repo/pulls", "--jq", ".[].title"}
stdout, stderr, err := gh.Exec(args...)

// GraphQL call
args := []string{"api", "graphql", "-F", "owner=cli", "-F", "repo=cli",
    "-f", "query=query($owner:String!,$repo:String!){repository(owner:$owner,name:$repo){name}}"}
stdout, _, err := gh.Exec(args...)
```

### 5.2 Via REST Client (Direct)

Use the go-gh REST client for direct API calls without subprocess overhead.

```go
client, err := api.DefaultRESTClient()
if err != nil {
    log.Fatal(err)
}

// GET request
var pulls []struct {
    Number int    `json:"number"`
    Title  string `json:"title"`
    State  string `json:"state"`
}
err = client.Get("repos/owner/repo/pulls", &pulls)

// POST request
body := bytes.NewBufferString(`{"title":"New Issue","body":"Description"}`)
var response struct{ Number int }
err = client.Post("repos/owner/repo/issues", body, &response)

// Pagination (manual)
var linkRE = regexp.MustCompile(`<([^>]+)>;\s*rel="([^"]+)"`)
path := "repos/cli/cli/releases"
for {
    resp, err := client.Request(http.MethodGet, path, nil)
    // Process resp.Body...
    link := resp.Header.Get("Link")
    // Parse next page from Link header...
}
```

### 5.3 Via GraphQL Client (Direct)

For complex queries, especially PR operations with nested data.

```go
client, err := api.DefaultGraphQLClient()
if err != nil {
    log.Fatal(err)
}

// Struct-based query (recommended for type safety)
var query struct {
    Repository struct {
        PullRequests struct {
            Nodes []struct {
                Number int
                Title  string
                State  string
                Author struct {
                    Login string
                }
            }
        } `graphql:"pullRequests(first: 100, states: OPEN)"`
    } `graphql:"repository(owner: $owner, name: $repo)"`
}

variables := map[string]interface{}{
    "owner": graphql.String("cli"),
    "repo":  graphql.String("cli"),
}

err = client.Query("RepositoryPRs", &query, variables)

// Raw query string
var result map[string]interface{}
err = client.Do(`query { viewer { login } }`, nil, &result)

// Mutation
var mutation struct {
    AddStar struct {
        Starrable struct {
            StargazerCount int
        }
    } `graphql:"addStar(input: $input)"`
}
variables := map[string]interface{}{
    "input": map[string]interface{}{
        "starrableId": "MDEwOlJlcG9zaXRvcnkxMjM=",
    },
}
err = client.Mutate("AddStar", &mutation, variables)
```

### 5.4 Via gh api Command (from Shell Scripts)

```bash
# REST
gh api repos/{owner}/{repo}/pulls --jq '.[].title'

# GraphQL
gh api graphql -F owner='{owner}' -F repo='{repo}' \
  -f query='query($owner:String!,$repo:String!) {
    repository(owner:$owner,name:$repo) {
      pullRequests(first:10,states:OPEN) {
        nodes { number title }
      }
    }
  }'

# Pagination
gh api graphql --paginate -f query='
  query($endCursor:String) {
    repository(owner:"cli",name:"cli") {
      releases(first:30,after:$endCursor) {
        nodes { tagName }
        pageInfo { hasNextPage endCursor }
      }
    }
  }'
```

---

## 6. Authentication Inheritance

This is one of the biggest advantages of gh extensions: **authentication is automatic**.

### How It Works

1. When users run `gh auth login`, gh stores OAuth tokens in platform-specific credential stores.
2. Extensions using go-gh clients automatically use these stored tokens.
3. No separate credential management is needed in extensions.

### Token Resolution Order

1. `GH_TOKEN` environment variable (highest priority).
2. `GH_HOST` + `GH_TOKEN` for enterprise instances.
3. Stored OAuth token from `gh auth login`.
4. gh configuration files (fallback).

### In Code

```go
// Automatic -- DefaultRESTClient already uses gh auth
client, err := api.DefaultRESTClient()

// Explicit token override
client, err := api.NewRESTClient(api.ClientOptions{
    AuthToken: "ghp_xxxxxxxxxxxx",
})

// Read token directly
import "github.com/cli/go-gh/v2/pkg/auth"
token, host := auth.TokenForHost("github.com")
```

### Security Considerations

- Auth tokens are only added to requests when the configured host matches the request host.
- Extensions operate under the same permissions as the user's gh auth scope.
- Third-party extensions are NOT certified by GitHub -- users should audit source code before installing.

---

## 7. Extension Distribution and Installation

### Publishing

1. Create a public GitHub repository with the `gh-` prefix.
2. For precompiled extensions: attach binaries to a GitHub Release with proper naming.
3. Add the `gh-extension` topic for discoverability.

### Installation

```bash
# By owner/repo
gh extension install owner/gh-extension-name

# By full URL (useful for GitHub Enterprise Server)
gh extension install https://github.com/owner/gh-extension-name

# From local directory (development)
gh extension install .
```

**Behavior:**
- For precompiled extensions, gh automatically detects OS/arch and downloads the correct binary.
- For script extensions, gh clones the repository.
- Extensions are locally installed and scoped to individual users.
- Installation path: `~/.local/share/gh/extensions/` (macOS/Linux).

### Listing Installed Extensions

```bash
gh extension list
# Shows all installed extensions and indicates available updates
```

### Removal

```bash
gh extension remove EXTENSION-NAME
```

---

## 8. Extension Upgrade Mechanism

```bash
# Upgrade a single extension
gh extension upgrade EXTENSION-NAME

# Upgrade all installed extensions
gh extension upgrade --all
```

**How it works:**
- `gh extension list` shows which extensions have available updates.
- For precompiled extensions, gh downloads the latest release binary.
- For script extensions, gh pulls the latest commit from the repository.

---

## 9. CLI Flags, Arguments, and Output Formatting

### Argument Passing

All arguments after `gh EXTENSION-NAME` are passed directly to the extension.

**In bash scripts:**
```bash
# $1, $2, etc. contain the arguments
while [ $# -gt 0 ]; do
  case "$1" in
    --verbose) verbose=1 ;;
    --name)    name_arg="$2"; shift ;;
    -h|--help) echo "Usage: ..."; exit 0 ;;
  esac
  shift
done
```

### Cobra Framework (Recommended for Go Extensions)

The gh CLI itself uses Cobra. Using Cobra in extensions provides consistency.

```go
import "github.com/spf13/cobra"

var rootCmd = &cobra.Command{
    Use:   "my-extension",
    Short: "A brief description of the extension",
    Long:  "A longer description...",
    RunE: func(cmd *cobra.Command, args []string) error {
        // Main logic here
        return nil
    },
}

// Persistent flags (available to all subcommands)
rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
rootCmd.PersistentFlags().StringVar(&repo, "repo", "", "repository in OWNER/REPO format")

// Local flags (only for this command)
rootCmd.Flags().IntVarP(&limit, "limit", "l", 30, "maximum number of results")

func main() {
    if err := rootCmd.Execute(); err != nil {
        os.Exit(1)
    }
}
```

### Subcommands

```go
var listCmd = &cobra.Command{
    Use:   "list",
    Short: "List items",
    RunE:  runList,
}

var createCmd = &cobra.Command{
    Use:   "create",
    Short: "Create an item",
    RunE:  runCreate,
}

func init() {
    rootCmd.AddCommand(listCmd)
    rootCmd.AddCommand(createCmd)
}
```

### Output Formatting Best Practices

1. **TTY detection:** Check if output is a terminal for formatting decisions.
2. **Table output:** Use `tableprinter` for columnar data.
3. **Color:** Use ANSI codes only in TTY mode.
4. **Machine-readable:** Support `--json` flag for programmatic use.

```go
terminal := term.FromEnv()
if terminal.IsTerminalOutput() {
    // Human-readable formatted output with colors
    t := tableprinter.New(terminal.Out(), true, termWidth)
    // ...
} else {
    // Machine-readable TSV output
    t := tableprinter.New(os.Stdout, false, 0)
    // ...
}
```

---

## 10. JSON Output for Machine Readability

### The --json Pattern

The gh CLI implements JSON output via `--json`, `--jq`, and `--template` flags. Extensions should follow this pattern.

**How gh CLI does it internally** (from `pkg/cmdutil/json_flags.go`):
```go
// AddJSONFlags adds --json, --jq, and --template flags
func AddJSONFlags(cmd *cobra.Command, exportTarget *Exporter, fields []string)
```

### Implementing in Your Extension

```go
import "encoding/json"

var jsonOutput bool
var jqFilter string

func init() {
    rootCmd.Flags().BoolVar(&jsonOutput, "json", false, "output as JSON")
    rootCmd.Flags().StringVar(&jqFilter, "jq", "", "filter JSON output with jq expression")
}

func outputResults(results []Result) error {
    if jsonOutput {
        data, err := json.MarshalIndent(results, "", "  ")
        if err != nil {
            return err
        }

        if jqFilter != "" {
            // Use go-gh jq package
            return jq.Evaluate(os.Stdout, bytes.NewReader(data), jqFilter)
        }

        fmt.Println(string(data))
        return nil
    }

    // Human-readable table output
    terminal := term.FromEnv()
    w, _, _ := terminal.Size()
    t := tableprinter.New(terminal.Out(), terminal.IsTerminalOutput(), w)
    for _, r := range results {
        t.AddField(fmt.Sprintf("#%d", r.Number))
        t.AddField(r.Title)
        t.AddField(r.Status)
        t.EndRow()
    }
    return t.Render()
}
```

### Using go-gh jq Package

```go
import "github.com/cli/go-gh/v2/pkg/jq"

jsonData := `[{"name":"foo","count":10},{"name":"bar","count":20}]`
// Evaluate jq expression
err := jq.Evaluate(os.Stdout, strings.NewReader(jsonData), ".[].name")
// Output: "foo"\n"bar"
```

### Using go-gh template Package

```go
import "github.com/cli/go-gh/v2/pkg/template"

// Process JSON with Go templates (same engine as gh --template)
```

---

## 11. Error Handling

### General Principles

1. **Write errors to stderr**, not stdout.
2. **Use non-zero exit codes** for failures.
3. **Wrap errors with context** as they bubble up the call chain.
4. **Match gh CLI error patterns** for consistency.

### In Go Extensions

```go
func main() {
    if err := rootCmd.Execute(); err != nil {
        fmt.Fprintf(os.Stderr, "error: %v\n", err)
        os.Exit(1)
    }
}

// In command handlers
func runList(cmd *cobra.Command, args []string) error {
    client, err := api.DefaultRESTClient()
    if err != nil {
        return fmt.Errorf("failed to create API client: %w", err)
    }

    var pulls []PullRequest
    err = client.Get(path, &pulls)
    if err != nil {
        // Check for specific API errors
        var httpErr *api.HTTPError
        if errors.As(err, &httpErr) {
            switch httpErr.StatusCode {
            case 404:
                return fmt.Errorf("repository not found: %s", repo)
            case 403:
                return fmt.Errorf("insufficient permissions (scope needed: repo)")
            case 422:
                return fmt.Errorf("validation failed: %s", httpErr.Message)
            }
        }
        return fmt.Errorf("API request failed: %w", err)
    }

    return nil
}
```

### Handling GraphQL Errors

```go
var gqlErr *api.GraphQLError
if errors.As(err, &gqlErr) {
    // Check for specific error types
    if gqlErr.Match("NOT_FOUND", "repository") {
        return fmt.Errorf("repository not found")
    }
    if gqlErr.Match("FORBIDDEN", "") {
        return fmt.Errorf("insufficient permissions")
    }
    // Log all errors for debugging
    for _, e := range gqlErr.Errors {
        fmt.Fprintf(os.Stderr, "GraphQL error: %s (type: %s, path: %v)\n",
            e.Message, e.Type, e.Path)
    }
}
```

### In Bash Extensions

```bash
#!/usr/bin/env bash
set -e  # Exit on error

# Check prerequisites
if ! command -v gh &> /dev/null; then
    echo "Error: gh CLI is not installed" >&2
    exit 1
fi

# Check authentication
if ! gh auth status &> /dev/null; then
    echo "Error: not authenticated. Run 'gh auth login' first." >&2
    exit 1
fi

# Handle API errors
result=$(gh api repos/{owner}/{repo}/pulls 2>&1) || {
    echo "Error: failed to fetch pull requests: $result" >&2
    exit 1
}
```

---

## 12. GitHub GraphQL API for PR Operations

### Available PR Mutations

The GitHub GraphQL API provides comprehensive PR mutation support:

| Mutation | Input Type | Purpose |
|----------|-----------|---------|
| `createPullRequest` | `CreatePullRequestInput!` | Create a new PR |
| `mergePullRequest` | `MergePullRequestInput!` | Merge a PR |
| `closePullRequest` | `ClosePullRequestInput!` | Close a PR |
| `reopenPullRequest` | `ReopenPullRequestInput!` | Reopen a closed PR |
| `updatePullRequest` | `UpdatePullRequestInput!` | Update PR title, body, etc. |
| `updatePullRequestBranch` | `UpdatePullRequestBranchInput!` | Update PR branch from base |
| `convertPullRequestToDraft` | `ConvertPullRequestToDraftInput!` | Convert to draft |
| `markPullRequestReadyForReview` | `MarkPullRequestReadyForReviewInput!` | Mark ready for review |
| `enablePullRequestAutoMerge` | `EnablePullRequestAutoMergeInput!` | Enable auto-merge |
| `disablePullRequestAutoMerge` | `DisablePullRequestAutoMergeInput!` | Disable auto-merge |
| `enqueuePullRequest` | `EnqueuePullRequestInput!` | Add to merge queue |
| `dequeuePullRequest` | `DequeuePullRequestInput!` | Remove from merge queue |

### PR Review Mutations

| Mutation | Input Type | Purpose |
|----------|-----------|---------|
| `addPullRequestReview` | `AddPullRequestReviewInput!` | Create a review |
| `submitPullRequestReview` | `SubmitPullRequestReviewInput!` | Submit a pending review |
| `updatePullRequestReview` | `UpdatePullRequestReviewInput!` | Update review body |
| `deletePullRequestReview` | `DeletePullRequestReviewInput!` | Delete a review |
| `dismissPullRequestReview` | `DismissPullRequestReviewInput!` | Dismiss a review |
| `addPullRequestReviewComment` | `AddPullRequestReviewCommentInput!` | Add inline comment |
| `addPullRequestReviewThread` | `AddPullRequestReviewThreadInput!` | Create review thread |
| `addPullRequestReviewThreadReply` | `AddPullRequestReviewThreadReplyInput!` | Reply to thread |
| `deletePullRequestReviewComment` | `DeletePullRequestReviewCommentInput!` | Delete inline comment |

### Related Mutations (Work on PRs)

| Mutation | Purpose |
|----------|---------|
| `addComment` | Add comment to issue or PR |
| `addLabelsToLabelable` | Add labels to PR |
| `removeLabelsFromLabelable` | Remove labels from PR |
| `addReaction` | React to PR or comment |
| `requestReviews` | Request reviews from users/teams |

### Example: Query PRs with GraphQL (Go)

```go
client, err := api.DefaultGraphQLClient()
if err != nil {
    return err
}

var query struct {
    Repository struct {
        PullRequests struct {
            Nodes []struct {
                Number    int
                Title     string
                State     string
                IsDraft   bool
                URL       string
                CreatedAt time.Time
                Author    struct {
                    Login string
                }
                Reviews struct {
                    Nodes []struct {
                        State  string
                        Author struct {
                            Login string
                        }
                    }
                } `graphql:"reviews(first: 10, states: [APPROVED, CHANGES_REQUESTED])"`
                Labels struct {
                    Nodes []struct {
                        Name string
                    }
                } `graphql:"labels(first: 10)"`
                MergeStateStatus string
            }
            PageInfo struct {
                HasNextPage bool
                EndCursor   string
            }
        } `graphql:"pullRequests(first: $first, after: $after, states: $states, orderBy: {field: UPDATED_AT, direction: DESC})"`
    } `graphql:"repository(owner: $owner, name: $repo)"`
}

variables := map[string]interface{}{
    "owner":  graphql.String("cli"),
    "repo":   graphql.String("cli"),
    "first":  graphql.Int(30),
    "after":  (*graphql.String)(nil),
    "states": []graphql.String{"OPEN"},
}

err = client.Query("RepoPRs", &query, variables)
```

### Example: Merge PR with GraphQL (Go)

```go
var mutation struct {
    MergePullRequest struct {
        PullRequest struct {
            Number int
            State  string
            URL    string
        }
    } `graphql:"mergePullRequest(input: $input)"`
}

variables := map[string]interface{}{
    "input": map[string]interface{}{
        "pullRequestId": prNodeID,          // Global node ID
        "mergeMethod":   "SQUASH",          // MERGE, SQUASH, or REBASE
        "commitHeadline": "Merge PR #42",
    },
}

err = client.Mutate("MergePR", &mutation, variables)
```

### Example: Using gh api Command for GraphQL

```bash
# List open PRs
gh api graphql -F owner='{owner}' -F repo='{repo}' -f query='
  query($owner: String!, $repo: String!) {
    repository(owner: $owner, name: $repo) {
      pullRequests(first: 30, states: OPEN, orderBy: {field: UPDATED_AT, direction: DESC}) {
        nodes {
          number
          title
          author { login }
          isDraft
          mergeStateStatus
          reviews(first: 5, states: [APPROVED]) {
            totalCount
          }
        }
      }
    }
  }'

# Paginated query
gh api graphql --paginate -f query='
  query($endCursor: String) {
    repository(owner: "cli", name: "cli") {
      pullRequests(first: 100, after: $endCursor, states: MERGED) {
        nodes {
          number
          title
          mergedAt
          mergedBy { login }
        }
        pageInfo { hasNextPage endCursor }
      }
    }
  }'
```

### gh api Command Reference

**Key flags:**
| Flag | Purpose |
|------|---------|
| `--method`, `-X` | HTTP method (default: GET, or POST if body present) |
| `--field`, `-F` | Typed field with magic conversion (booleans, ints, file reads) |
| `--raw-field`, `-f` | Raw string field |
| `--jq`, `-q` | jq expression to filter output |
| `--template`, `-t` | Go template for formatting |
| `--paginate` | Fetch all pages |
| `--slurp` | Combine paginated results into array |
| `--cache` | Cache duration (e.g., "1h") |
| `--header`, `-H` | Custom HTTP header |
| `--include`, `-i` | Include HTTP headers in output |
| `--silent` | Suppress response body |
| `--hostname` | Target GitHub instance |

**Placeholder variables** in endpoint paths:
- `{owner}` -- replaced with repository owner
- `{repo}` -- replaced with repository name
- `{branch}` -- replaced with current branch

---

## 13. Popular Extensions and Architecture Patterns

### Top Extensions by Stars (2026)

| Extension | Stars | Language | Description |
|-----------|-------|----------|-------------|
| [gh-dash](https://github.com/dlvhdr/gh-dash) | 10.2k | Go | Rich terminal UI dashboard for GitHub |
| [gh-aw](https://github.com/some/gh-aw) | 3.4k | Go | Agentic Workflows tool |
| [gh-skyline](https://github.com/some/gh-skyline) | 1.2k | Go | 3D contribution visualization |
| [gh-copilot](https://github.com/some/gh-copilot) | 1.1k | Go | AI assistance in terminal |
| [gh-poi](https://github.com/seachicken/gh-poi) | 892 | Go | Safely clean up local branches |
| [gh-markdown-preview](https://github.com/some/gh-markdown-preview) | 780 | Go | Markdown preview with GitHub styling |
| [gh-eco](https://github.com/some/gh-eco) | 461 | Go | Explore the ecosystem |
| [gh-s](https://github.com/some/gh-s) | 385 | Go | Interactive repository search |
| [gh-token](https://github.com/some/gh-token) | 370 | Go | Manage GitHub app installation tokens |
| [gh-notify](https://github.com/some/gh-notify) | 325 | Shell | Terminal GitHub notifications |
| [gh-actions-cache](https://github.com/some/gh-actions-cache) | 323 | Go | Manage Actions cache |

**Key observation:** The majority of popular extensions are written in Go.

### Architecture Pattern: gh-dash (TUI Application)

gh-dash is the most popular Go-based gh extension and uses a clean architecture:

**Directory structure:**
```
gh-dash/
  cmd/          # CLI entry points
  ui/           # TUI rendering (Bubble Tea components)
  data/         # GitHub GraphQL API queries
  config/       # User config.yml parsing
  utils/        # Helper utilities
  internal/     # Private packages
  testdata/     # Test fixtures
  docs/         # Documentation (Hugo)
```

**Technology stack:**
- **TUI:** Bubble Tea (charmbracelet/bubbletea) -- Elm Architecture
- **API:** GitHub GraphQL API via go-gh
- **Markdown:** charmbracelet/glamour for rendering
- **Config:** YAML-based user configuration

**Elm Architecture pattern (Model-Update-View):**
```go
// Model holds application state
type Model struct {
    prs      []PullRequest
    cursor   int
    loading  bool
}

// Init returns initial command
func (m Model) Init() tea.Cmd { return fetchPRs }

// Update handles events and updates state
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        // Handle keyboard input
    case prsLoadedMsg:
        m.prs = msg.prs
        m.loading = false
    }
    return m, nil
}

// View renders the UI
func (m Model) View() string {
    // Render based on current state
}
```

### Architecture Pattern: Simple CLI Extension

Most extensions follow a simpler pattern:

```
gh-my-extension/
  main.go                          # Entry point
  cmd/
    root.go                        # Root Cobra command
    list.go                        # list subcommand
    create.go                      # create subcommand
  internal/
    api/
      client.go                    # API wrapper
      types.go                     # Response types
    output/
      table.go                     # Table formatting
      json.go                      # JSON output
  .github/
    workflows/
      release.yml                  # gh-extension-precompile
  go.mod
  go.sum
```

### Architecture Pattern: gh-poi (Branch Cleanup)

gh-poi demonstrates a focused single-purpose extension:
- Uses go-gh for API access and auth
- Detects merged/deleted remote branches
- Safely removes local tracking branches
- Simple CLI without subcommands

### Architecture Pattern: Shell Script Extensions

For simpler tools, shell scripts with `gh api` are sufficient:

```
gh-my-script/
  gh-my-script                     # Executable bash script
```

---

## 14. Testing Strategies

### Local Testing During Development

```bash
# Install from local directory
gh extension install .

# Test execution
gh my-extension --help
gh my-extension list --json

# After changes, reinstall
gh extension remove my-extension
gh extension install .
```

### Unit Testing Go Extensions

#### Testing API Calls with httptest

```go
import (
    "net/http"
    "net/http/httptest"
    "testing"
    "github.com/cli/go-gh/v2/pkg/api"
)

func TestListPRs(t *testing.T) {
    // Create mock server
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path == "/repos/owner/repo/pulls" {
            w.WriteHeader(200)
            w.Write([]byte(`[{"number":1,"title":"Test PR"}]`))
            return
        }
        w.WriteHeader(404)
    }))
    defer server.Close()

    // Create client pointing to mock server
    client, err := api.NewRESTClient(api.ClientOptions{
        Host:      server.URL,
        AuthToken: "test-token",
    })
    if err != nil {
        t.Fatal(err)
    }

    var pulls []struct {
        Number int    `json:"number"`
        Title  string `json:"title"`
    }
    err = client.Get("repos/owner/repo/pulls", &pulls)
    if err != nil {
        t.Fatal(err)
    }

    if len(pulls) != 1 {
        t.Errorf("expected 1 PR, got %d", len(pulls))
    }
    if pulls[0].Title != "Test PR" {
        t.Errorf("expected 'Test PR', got '%s'", pulls[0].Title)
    }
}
```

#### Interface-Based Mocking

```go
// Define interface for your API interactions
type GitHubClient interface {
    ListPullRequests(owner, repo string) ([]PullRequest, error)
    MergePullRequest(owner, repo string, number int) error
}

// Real implementation
type realClient struct {
    rest *api.RESTClient
}

// Mock implementation for tests
type mockClient struct {
    prs []PullRequest
    err error
}

func (m *mockClient) ListPullRequests(owner, repo string) ([]PullRequest, error) {
    return m.prs, m.err
}

func (m *mockClient) MergePullRequest(owner, repo string, number int) error {
    return m.err
}

// Test with mock
func TestRunList(t *testing.T) {
    mock := &mockClient{
        prs: []PullRequest{{Number: 1, Title: "Test"}},
    }
    err := runListWithClient(mock)
    if err != nil {
        t.Fatal(err)
    }
}
```

#### Using httpmock Library

```go
import "github.com/jarcoal/httpmock"

func TestAPICall(t *testing.T) {
    httpmock.Activate()
    defer httpmock.DeactivateAndReset()

    httpmock.RegisterResponder("GET", "https://api.github.com/repos/owner/repo/pulls",
        httpmock.NewJsonResponderOrPanic(200, []map[string]interface{}{
            {"number": 1, "title": "Test PR"},
        }))

    // Run your function that makes API calls
    result, err := listPRs("owner", "repo")
    // Assert results...
}
```

#### Testing Cobra Commands

```go
func TestRootCommand(t *testing.T) {
    cmd := NewRootCmd()
    buf := new(bytes.Buffer)
    cmd.SetOut(buf)
    cmd.SetErr(buf)
    cmd.SetArgs([]string{"list", "--limit", "5"})

    err := cmd.Execute()
    if err != nil {
        t.Fatal(err)
    }

    output := buf.String()
    if !strings.Contains(output, "expected content") {
        t.Errorf("unexpected output: %s", output)
    }
}
```

#### Testing Table Output

```go
func TestTableOutput(t *testing.T) {
    buf := new(bytes.Buffer)
    // isTTY=false produces TSV, easy to parse in tests
    tp := tableprinter.New(buf, false, 80)
    tp.AddField("hello")
    tp.AddField("world")
    tp.EndRow()
    tp.Render()

    expected := "hello\tworld\n"
    if buf.String() != expected {
        t.Errorf("got %q, want %q", buf.String(), expected)
    }
}
```

### Integration Testing

```bash
# In CI, test the full binary
go build -o gh-my-extension .
./gh-my-extension --help
./gh-my-extension list --json 2>&1 | jq .
```

---

## 15. Project Structure and Scaffolding

### Creating a New Go Extension

```bash
# Interactive wizard
gh extension create

# Direct creation
gh extension create --precompiled=go my-extension
```

This generates:
```
gh-my-extension/
  .github/
    workflows/
      release.yml              # gh-extension-precompile workflow
  main.go                      # Entry point with go-gh example
  go.mod                       # Module definition
  go.sum                       # Dependency checksums
```

### Recommended Expanded Structure

For production extensions, expand to:

```
gh-my-extension/
  .github/
    workflows/
      release.yml              # Automated release
      ci.yml                   # CI/CD (tests, lint)
  cmd/
    root.go                    # Root command with persistent flags
    list.go                    # Subcommands
    merge.go
    version.go
  internal/
    api/
      client.go                # GitHub API wrapper
      graphql.go               # GraphQL queries/mutations
      types.go                 # API response types
    config/
      config.go                # Extension configuration
    output/
      table.go                 # Table rendering
      json.go                  # JSON output
  main.go                      # Entry point: cmd.Execute()
  go.mod
  go.sum
  LICENSE
  README.md
```

### Essential go.mod

```go
module github.com/YOUR-USERNAME/gh-my-extension

go 1.22

require (
    github.com/cli/go-gh/v2 v2.13.0
    github.com/spf13/cobra v1.8.0
)
```

### Minimal main.go

```go
package main

import (
    "fmt"
    "os"

    "github.com/YOUR-USERNAME/gh-my-extension/cmd"
)

func main() {
    if err := cmd.Execute(); err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
}
```

---

## 16. Complete Code Examples

### Example 1: Simple API Query Extension

```go
package main

import (
    "fmt"
    "log"
    "os"

    gh "github.com/cli/go-gh/v2"
    "github.com/cli/go-gh/v2/pkg/api"
    "github.com/cli/go-gh/v2/pkg/repository"
    "github.com/cli/go-gh/v2/pkg/tableprinter"
    "github.com/cli/go-gh/v2/pkg/term"
)

func main() {
    // Detect current repository
    repo, err := repository.Current()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: not in a git repository: %v\n", err)
        os.Exit(1)
    }

    // Create authenticated REST client
    client, err := api.DefaultRESTClient()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }

    // Fetch pull requests
    var pulls []struct {
        Number int    `json:"number"`
        Title  string `json:"title"`
        State  string `json:"state"`
        User   struct {
            Login string `json:"login"`
        } `json:"user"`
    }

    path := fmt.Sprintf("repos/%s/%s/pulls?state=open&per_page=30", repo.Owner, repo.Name)
    err = client.Get(path, &pulls)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error fetching PRs: %v\n", err)
        os.Exit(1)
    }

    // Render as table
    terminal := term.FromEnv()
    w, _, _ := terminal.Size()
    t := tableprinter.New(terminal.Out(), terminal.IsTerminalOutput(), w)

    green := func(s string) string { return "\x1b[32m" + s + "\x1b[m" }

    for _, pr := range pulls {
        t.AddField(fmt.Sprintf("#%d", pr.Number), tableprinter.WithColor(green))
        t.AddField(pr.Title)
        t.AddField(pr.User.Login)
        t.EndRow()
    }

    if err := t.Render(); err != nil {
        log.Fatal(err)
    }
}
```

### Example 2: GraphQL Extension with Cobra

```go
// cmd/root.go
package cmd

import (
    "fmt"
    "os"

    "github.com/spf13/cobra"
)

var (
    repoFlag string
    jsonFlag bool
    limitFlag int
)

var rootCmd = &cobra.Command{
    Use:   "my-ext",
    Short: "A GitHub CLI extension",
    Long:  "Does amazing things with GitHub.",
}

func Execute() error {
    return rootCmd.Execute()
}

func init() {
    rootCmd.PersistentFlags().StringVarP(&repoFlag, "repo", "R", "", "repository (OWNER/REPO)")
    rootCmd.PersistentFlags().BoolVar(&jsonFlag, "json", false, "output as JSON")
}

// cmd/list.go
package cmd

import (
    "encoding/json"
    "fmt"
    "os"

    "github.com/cli/go-gh/v2/pkg/api"
    "github.com/cli/go-gh/v2/pkg/repository"
    "github.com/cli/go-gh/v2/pkg/tableprinter"
    "github.com/cli/go-gh/v2/pkg/term"
    "github.com/shurcooL/graphql"
    "github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
    Use:   "list",
    Short: "List open pull requests",
    RunE:  runList,
}

func init() {
    listCmd.Flags().IntVarP(&limitFlag, "limit", "l", 30, "max results")
    rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
    // Determine repository
    var repo repository.Repository
    var err error
    if repoFlag != "" {
        repo, err = repository.Parse(repoFlag)
    } else {
        repo, err = repository.Current()
    }
    if err != nil {
        return fmt.Errorf("could not determine repository: %w", err)
    }

    // GraphQL query
    client, err := api.DefaultGraphQLClient()
    if err != nil {
        return err
    }

    var query struct {
        Repository struct {
            PullRequests struct {
                Nodes []struct {
                    Number    int
                    Title     string
                    Author    struct{ Login string }
                    IsDraft   bool
                    CreatedAt string
                }
            } `graphql:"pullRequests(first: $limit, states: OPEN, orderBy: {field: UPDATED_AT, direction: DESC})"`
        } `graphql:"repository(owner: $owner, name: $repo)"`
    }

    variables := map[string]interface{}{
        "owner": graphql.String(repo.Owner),
        "repo":  graphql.String(repo.Name),
        "limit": graphql.Int(limitFlag),
    }

    err = client.Query("ListPRs", &query, variables)
    if err != nil {
        return fmt.Errorf("GraphQL query failed: %w", err)
    }

    prs := query.Repository.PullRequests.Nodes

    // Output
    if jsonFlag {
        data, _ := json.MarshalIndent(prs, "", "  ")
        fmt.Println(string(data))
        return nil
    }

    terminal := term.FromEnv()
    w, _, _ := terminal.Size()
    t := tableprinter.New(terminal.Out(), terminal.IsTerminalOutput(), w)

    for _, pr := range prs {
        t.AddField(fmt.Sprintf("#%d", pr.Number))
        t.AddField(pr.Title)
        t.AddField(pr.Author.Login)
        t.EndRow()
    }

    return t.Render()
}
```

### Example 3: Shelling Out to gh

```go
package main

import (
    "fmt"
    "os"

    gh "github.com/cli/go-gh/v2"
)

func main() {
    // Execute gh command and capture output
    args := []string{"api", "user", "--jq", `"You are @\(.login) (\(.name))"`}
    stdout, stderr, err := gh.Exec(args...)
    if err != nil {
        fmt.Fprintf(os.Stderr, "gh error: %s\n%s\n", err, stderr.String())
        os.Exit(1)
    }
    fmt.Println(stdout.String())
}
```

### Example 4: REST Client with Custom Options and Pagination

```go
package main

import (
    "fmt"
    "net/http"
    "regexp"
    "time"

    "github.com/cli/go-gh/v2/pkg/api"
)

func main() {
    opts := api.ClientOptions{
        EnableCache: true,
        CacheTTL:    5 * time.Minute,
        Timeout:     10 * time.Second,
    }
    client, err := api.NewRESTClient(opts)
    if err != nil {
        panic(err)
    }

    // Manual pagination
    var linkRE = regexp.MustCompile(`<([^>]+)>;\s*rel="([^"]+)"`)
    path := "repos/cli/cli/releases"

    for path != "" {
        resp, err := client.Request(http.MethodGet, path, nil)
        if err != nil {
            panic(err)
        }
        // Process resp.Body...
        resp.Body.Close()

        // Parse Link header for next page
        path = ""
        for _, m := range linkRE.FindAllStringSubmatch(resp.Header.Get("Link"), -1) {
            if m[2] == "next" {
                path = m[1]
                break
            }
        }
    }
}
```

### Example 5: GraphQL Pagination with Cursors

```go
client, _ := api.DefaultGraphQLClient()

var allReleases []Release
var endCursor *graphql.String

for {
    var query struct {
        Repository struct {
            Releases struct {
                Nodes []struct {
                    Name    string
                    TagName string
                }
                PageInfo struct {
                    HasNextPage bool
                    EndCursor   string
                }
            } `graphql:"releases(first: 30, after: $after, orderBy: {field: CREATED_AT, direction: DESC})"`
        } `graphql:"repository(owner: $owner, name: $repo)"`
    }

    variables := map[string]interface{}{
        "owner": graphql.String("cli"),
        "repo":  graphql.String("cli"),
        "after": endCursor,
    }

    err := client.Query("Releases", &query, variables)
    if err != nil {
        break
    }

    // Collect results...

    if !query.Repository.Releases.PageInfo.HasNextPage {
        break
    }
    cursor := graphql.String(query.Repository.Releases.PageInfo.EndCursor)
    endCursor = &cursor
}
```

### Example 6: Downloading Release Assets

```go
opts := api.ClientOptions{
    Headers: map[string]string{
        "Accept": "application/octet-stream",
    },
}
client, _ := api.NewRESTClient(opts)

assetURL := "repos/cli/cli/releases/assets/12345"
response, err := client.Request(http.MethodGet, assetURL, nil)
if err != nil {
    panic(err)
}
defer response.Body.Close()

// Write response.Body to file...
```

---

## Summary of Best Practices

1. **Use Go with go-gh v2** for production extensions. It provides the best developer experience and user experience.
2. **Use Cobra** for CLI argument parsing to match gh CLI conventions.
3. **Use `gh-extension-precompile`** GitHub Action for automated cross-platform releases.
4. **Never commit binaries** to version control.
5. **Support `--json` output** for machine readability.
6. **Use `tableprinter`** for human-readable table output that adapts to terminal width.
7. **Detect TTY** to decide between human-readable and machine-readable output.
8. **Write errors to stderr** and use non-zero exit codes for failures.
9. **Wrap errors with context** as they propagate up the call chain.
10. **Use GraphQL for complex queries** (e.g., PRs with reviews, labels, status checks).
11. **Use REST for simple CRUD** operations.
12. **Let go-gh handle auth** -- never ask users for tokens.
13. **Add `gh-extension` topic** to your repository for discoverability.
14. **Test with interfaces** and `httptest` for unit testing API calls.
15. **Follow the naming convention** strictly: `gh-` prefix for repos, proper binary naming for releases.
16. **Support `--repo` / `-R` flag** to allow users to specify a repository (matching gh CLI conventions).
17. **Use `repository.Current()`** as the default when no repo is specified.

---

## References and Sources

- [Creating GitHub CLI Extensions (GitHub Docs)](https://docs.github.com/en/github-cli/github-cli/creating-github-cli-extensions)
- [Using GitHub CLI Extensions (GitHub Docs)](https://docs.github.com/en/github-cli/github-cli/using-github-cli-extensions)
- [go-gh Library (GitHub)](https://github.com/cli/go-gh)
- [go-gh v2 API Reference (pkg.go.dev)](https://pkg.go.dev/github.com/cli/go-gh/v2)
- [go-gh v2 API Package (pkg.go.dev)](https://pkg.go.dev/github.com/cli/go-gh/v2/pkg/api)
- [go-gh v2 TablePrinter Package (pkg.go.dev)](https://pkg.go.dev/github.com/cli/go-gh/v2/pkg/tableprinter)
- [go-gh v2 Repository Package (pkg.go.dev)](https://pkg.go.dev/github.com/cli/go-gh/v2/pkg/repository)
- [gh-extension-precompile Action (GitHub)](https://github.com/cli/gh-extension-precompile)
- [gh api Command Reference](https://cli.github.com/manual/gh_api)
- [GitHub GraphQL Mutations Reference](https://docs.github.com/en/graphql/reference/mutations)
- [Extending the gh CLI with Go (Mike Ball)](https://mikeball.info/blog/extending-the-gh-cli-with-go/)
- [Fun With GitHub CLI Extensions (meiji163)](https://meiji163.github.io/post/gh-extension/)
- [go-gh InfoQ Article](https://www.infoq.com/news/2023/01/GitHub-simplifies-cli-extension/)
- [GitHub GraphQL PR Queries Gist](https://gist.github.com/MichaelCurrin/f8a7a11451ce4ec055d41000c915b595)
- [gh-dash Contributing Guide](https://github.com/dlvhdr/gh-dash/blob/main/CONTRIBUTING.md)
- [Cobra CLI Framework](https://github.com/spf13/cobra)
- [gh CLI Source (json_flags.go)](https://fossies.org/linux/gh-cli/pkg/cmdutil/json_flags.go)
- [gh Extensions Topic Page](https://github.com/topics/gh-extension)
