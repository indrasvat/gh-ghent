# Vivecaka Large PR Handling & Optimization Patterns

> Learnings extracted from the vivecaka GitHub PR reviewer TUI (`~/code/github.com/indrasvat-vivecaka/`).
> These patterns are directly applicable to ghent's design for handling large PRs, API optimization, and resilient data fetching.
> Date: 2026-02-21

---

## 1. Dual Field Lists for API Optimization

**Problem:** GitHub's `statusCheckRollup` field causes API timeouts when fetching large PR lists (50+ items).

**Solution:** Use two field lists — full for the initial load, lightweight for pagination:

```go
// internal/adapter/ghcli/reader.go:12-17
const (
    // prListFields is for the initial load (single page). Includes statusCheckRollup for CI status.
    prListFields = "number,title,author,state,isDraft,headRefName,baseRefName,labels,statusCheckRollup,reviewDecision,updatedAt,createdAt,url"
    // prListFieldsLight is for pagination (loading more PRs). Excludes statusCheckRollup to avoid API timeouts.
    // CI status will show as "none" for paginated items until detail view is opened.
    prListFieldsLight = "number,title,author,state,isDraft,headRefName,baseRefName,labels,reviewDecision,updatedAt,createdAt,url"
)
```

**Usage in ListPRs:**

```go
// internal/adapter/ghcli/reader.go:117-122
fields := prListFields
if opts.Page > 1 {
    fields = prListFieldsLight
}
args := []string{"pr", "list", "--json", fields}
```

**Ghent applicability:** When `gh ghent comments` or `gh ghent checks` fetches data for a PR, avoid requesting expensive fields (like full review thread bodies or diff hunks) in list views. Fetch them only on-demand in detail views.

---

## 2. Client-Side Pagination

**Problem:** `gh pr list` doesn't support server-side page offsets — it only has `--limit`.

**Solution:** Over-fetch and slice client-side:

```go
// internal/adapter/ghcli/reader.go:138-183
// For page N, we need to fetch N*PerPage items total and skip the first (N-1)*PerPage
page := max(opts.Page, 1)
perPage := opts.PerPage
if perPage <= 0 {
    perPage = 50 // default
}
limit := page * perPage
args = append(args, "--limit", fmt.Sprintf("%d", limit))

// ... fetch and filter ...

// Paginate over the filtered set.
startIdx := (page - 1) * perPage
if startIdx >= len(filtered) {
    return []domain.PR{}, nil
}
pageItems := filtered[startIdx:]
```

**Ghent applicability:** For `gh ghent comments --page 2`, the same over-fetch pattern is needed when using `gh api` with pagination. Consider whether ghent even needs pagination (most PRs have <50 review threads).

---

## 3. Parallel Fetching with Graceful Degradation (errgroup)

**Problem:** Fetching PR details + comments sequentially is slow. But comment fetch failure shouldn't block the core PR data.

**Solution:** `errgroup` with selective error propagation:

```go
// internal/usecase/get_pr.go:24-52
func (uc *GetPRDetail) Execute(ctx context.Context, repo domain.RepoRef, number int) (*domain.PRDetail, error) {
    g, ctx := errgroup.WithContext(ctx)

    var detail *domain.PRDetail
    var comments []domain.CommentThread

    g.Go(func() error {
        var err error
        detail, err = uc.reader.GetPR(ctx, repo, number)
        return err // This is required - no detail means failure.
    })

    g.Go(func() error {
        var err error
        comments, err = uc.reader.GetComments(ctx, repo, number)
        if err != nil {
            comments = nil // Tolerate failure.
        }
        return nil
    })

    if err := g.Wait(); err != nil {
        return nil, err
    }

    detail.Comments = comments
    return detail, nil
}
```

**Key pattern:** Required data returns `err` (which cancels the group via `errgroup.WithContext`). Optional data swallows the error and returns `nil`.

**Ghent applicability:** When fetching review threads + check runs + job logs for a PR, make check runs and job logs optional. If one fails, still show the rest. This is critical for agent use — partial data is better than total failure.

---

## 4. Multi-Repo Concurrent Fetching

**Problem:** When monitoring PRs across multiple repos, sequential fetching is too slow.

**Solution:** `errgroup` + `sync.Mutex` for concurrent collection with per-repo fault tolerance:

```go
// internal/usecase/inbox.go:30-63
func (uc *GetInboxPRs) Execute(ctx context.Context, repos []domain.RepoRef) ([]InboxPR, error) {
    var mu sync.Mutex
    var result []InboxPR

    g, ctx := errgroup.WithContext(ctx)

    for _, repo := range repos {
        g.Go(func() error {
            prs, err := uc.reader.ListPRs(ctx, repo, opts)
            if err != nil {
                return nil // tolerate individual repo failures
            }
            mu.Lock()
            for _, pr := range prs {
                result = append(result, InboxPR{PR: pr, Repo: repo})
            }
            mu.Unlock()
            return nil
        })
    }

    _ = g.Wait()
    return result, nil
}
```

**Ghent applicability:** If ghent ever supports multi-repo monitoring (e.g., `gh ghent checks -R owner/repo1 -R owner/repo2`), this pattern is directly reusable. Even for single-repo use, the parallel comments+checks pattern applies.

---

## 5. Atomic Cache Writes

**Problem:** Cache corruption from concurrent writes or interrupted processes.

**Solution:** Write to a temp file, then `os.Rename` (atomic on most filesystems):

```go
// internal/cache/cache.go:30-54
func Save(repo domain.RepoRef, prs []domain.PR) error {
    path := CachePath(repo)
    if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
        return fmt.Errorf("create cache dir: %w", err)
    }

    // ... marshal data ...

    // Write to temp file then rename for atomicity.
    tmp := path + ".tmp"
    if err := os.WriteFile(tmp, out, 0o644); err != nil {
        return fmt.Errorf("write cache: %w", err)
    }
    return os.Rename(tmp, path)
}
```

**Cache validation:** Verify repo ownership to prevent cross-repo cache pollution:

```go
// internal/cache/cache.go:74-76
if !strings.EqualFold(data.Repo, repo.String()) {
    return nil, time.Time{}, nil
}
```

**TTL-based staleness:**

```go
// internal/cache/cache.go:83-89
func IsStale(repo domain.RepoRef, ttl time.Duration) bool {
    _, updated, err := Load(repo)
    if err != nil || updated.IsZero() {
        return true
    }
    return time.Since(updated) > ttl
}
```

**Ghent applicability:** For `--watch` mode, cache the last known state of check runs and review threads. Show stale data immediately while refreshing in background. This makes ghent feel instant for repeated invocations.

---

## 6. Optimistic UI (Show Cached, Refresh in Background)

**Problem:** Users stare at a loading spinner while fresh data is fetched.

**Solution:** Show cached data immediately, then refresh:

```go
// internal/tui/app.go:318-324
case cachedPRsLoadedMsg:
    if len(msg.PRs) > 0 && a.prList.IsLoading() {
        // Show cached data immediately while fresh load is in progress.
        a.prList.SetPRs(msg.PRs)
        a.header.SetPRCount(a.prList.TotalPRs())
    }
    return a, nil
```

**Ghent applicability:** For `--watch` mode, render the last known check status immediately, then update when fresh data arrives. Add a visual indicator (e.g., "cached 30s ago") so users know data may be stale.

---

## 7. Comment Threading via in_reply_to_id

**Problem:** GitHub REST API returns flat comments, but they need to be grouped into threads for display.

**Solution:** Map `in_reply_to_id` to build thread hierarchies, preserving insertion order:

```go
// internal/adapter/ghcli/reader.go:259-299
func groupCommentsIntoThreads(comments []ghAPIComment) []domain.CommentThread {
    threadMap := make(map[int]*domain.CommentThread)
    var threadOrder []int

    for _, c := range comments {
        comment := domain.Comment{
            ID:     fmt.Sprintf("%d", c.ID),
            Author: c.User.Login,
            Body:   c.Body,
        }

        if c.InReplyToID != nil {
            // Reply to existing thread.
            if thread, ok := threadMap[*c.InReplyToID]; ok {
                thread.Comments = append(thread.Comments, comment)
                continue
            }
        }

        // New thread root.
        thread := domain.CommentThread{
            ID:       fmt.Sprintf("%d", c.ID),
            Path:     c.Path,
            Line:     line,
            Comments: []domain.Comment{comment},
        }
        threadMap[c.ID] = &thread
        threadOrder = append(threadOrder, c.ID)
    }

    threads := make([]domain.CommentThread, 0, len(threadOrder))
    for _, id := range threadOrder {
        threads = append(threads, *threadMap[id])
    }
    return threads
}
```

**Ghent applicability:** ghent uses GraphQL `reviewThreads` which already returns threaded data. However, if we also support REST fallback (more resilient), this threading logic is essential. The `threadOrder` slice preserving insertion order is important for deterministic output.

---

## 8. Streaming Diff Parser (O(n) Memory)

**Problem:** Large PR diffs can be megabytes. Parsing into an AST all at once is memory-intensive.

**Solution:** Line-by-line streaming parser with minimal state:

```go
// internal/adapter/ghcli/parser.go:21-75
func (p *diffParser) parse(raw string) domain.Diff {
    var diff domain.Diff
    var currentFile *domain.FileDiff
    var currentHunk *domain.Hunk

    for _, line := range strings.Split(raw, "\n") {
        switch {
        case strings.HasPrefix(line, "diff --git "):
            // Flush previous file
            if currentFile != nil {
                if currentHunk != nil {
                    currentFile.Hunks = append(currentFile.Hunks, *currentHunk)
                }
                diff.Files = append(diff.Files, *currentFile)
            }
            currentFile = &domain.FileDiff{}
            parseDiffHeader(line, currentFile)

        case strings.HasPrefix(line, "@@ "):
            // New hunk
            if currentFile != nil {
                if currentHunk != nil {
                    currentFile.Hunks = append(currentFile.Hunks, *currentHunk)
                }
                currentHunk = &domain.Hunk{Header: line}
                p.parseHunkHeader(line)
            }

        case currentHunk != nil:
            dl := p.parseDiffLine(line)
            if dl != nil {
                currentHunk.Lines = append(currentHunk.Lines, *dl)
            }
        }
    }
    // ... flush final file ...
}
```

**Key features:**
- Only two pointers in flight (`currentFile`, `currentHunk`)
- Tracks old/new line numbers incrementally during parsing
- Handles renames (old path != new path via `diff --git a/old b/new`)

**Ghent applicability:** If ghent ever needs to parse diffs (e.g., showing comment context), this parser is production-tested. For now, ghent can rely on the `diffHunk` field from GraphQL which is already per-comment.

---

## 9. Large File Threshold for Syntax Highlighting

**Problem:** Chroma tokenization stalls the UI on files with 5000+ lines.

**Solution:** A hard threshold that disables highlighting for large files:

```go
// internal/tui/views/diffview.go:100-102
// maxHighlightLines is the threshold above which syntax highlighting is disabled
// for a single file to prevent Chroma tokenization from stalling the UI.
const maxHighlightLines = 5000
```

**Ghent applicability:** If ghent ever renders diffs with syntax highlighting (unlikely for CLI output, but possible for HTML format), use a similar threshold. More broadly, this is a reminder to set hard limits on expensive operations — e.g., cap the number of review threads displayed by default, or truncate very long CI logs.

---

## 10. Lexer Caching with Double-Check Locking

**Problem:** Creating a new Chroma lexer for every highlighted line is expensive.

**Solution:** Cache lexers by file extension with read-preferring RWMutex:

```go
// internal/tui/views/diffview.go:42-70
func (h *syntaxHighlighter) getLexer(filename string) chroma.Lexer {
    ext := filepath.Ext(filename)
    if ext == "" {
        ext = filename // for files like Makefile, Dockerfile
    }

    h.mu.RLock()
    if lexer, ok := h.lexerCache[ext]; ok {
        h.mu.RUnlock()
        return lexer
    }
    h.mu.RUnlock()

    h.mu.Lock()
    defer h.mu.Unlock()

    // Double-check after acquiring write lock
    if lexer, ok := h.lexerCache[ext]; ok {
        return lexer
    }

    lexer := lexers.Match(filename)
    // ... cache and return ...
}
```

**Ghent applicability:** If ghent processes many files (e.g., grouping comments by file), cache any per-file computation (like mapping file paths to language names for syntax hints).

---

## 11. Auto-Refresh with Countdown

**Problem:** `--watch` mode needs periodic refresh without blocking user interaction.

**Solution:** 1-second tick loop with countdown and pause support:

```go
// internal/tui/app.go:275-296
func (a *App) handleRefreshTick() (tea.Model, tea.Cmd) {
    if a.refreshInterval <= 0 {
        return a, nil
    }
    if a.refreshPaused {
        a.header.SetRefreshCountdown(a.refreshCountdown, true)
        return a, a.refreshTick()
    }
    a.refreshCountdown--
    if a.refreshCountdown <= 0 {
        // Trigger auto-refresh.
        a.refreshCountdown = a.refreshInterval
        a.header.SetRefreshCountdown(a.refreshCountdown, false)
        // ... fire refresh command ...
        return a, tea.Batch(a.refreshTick(), loadPRsCmd(...))
    }
    a.header.SetRefreshCountdown(a.refreshCountdown, false)
    return a, a.refreshTick()
}
```

**Key features:**
- Countdown visible in header ("refreshing in 25s")
- Pausable (e.g., when user is in detail view)
- Fires `tea.Batch` to keep ticks going alongside the actual refresh

**Ghent applicability:** Directly applicable to `gh ghent checks --watch`. Use 1-second ticks for countdown display, configurable refresh interval (default 10-30s), and pause refresh while user is scrolling output.

---

## 12. Configurable Limits and Defaults

**Problem:** Hard-coded values make it impossible to tune for different environments.

**Solution:** Centralized config with sensible defaults and validation:

```go
// internal/config/config.go:58-83
func Default() *Config {
    return &Config{
        General: GeneralConfig{
            RefreshInterval: 30,    // seconds
            PageSize:        50,    // PRs per page
            CacheTTL:        5,     // minutes
            StaleDays:       7,     // days before PR considered stale
        },
        Diff: DiffConfig{
            ContextLines: 3,        // lines of context around changes
        },
    }
}
```

**Validation is comprehensive:**

```go
// internal/config/config.go:93-122
func (c *Config) Validate() error {
    if c.General.RefreshInterval < 0 {
        return fmt.Errorf("general.refresh_interval must be >= 0, got %d", ...)
    }
    if c.General.PageSize <= 0 {
        return fmt.Errorf("general.page_size must be > 0, got %d", ...)
    }
    // ... validate enums against allowed values ...
}
```

**Ghent applicability:** ghent should use similar configurable defaults:
- `--interval` for watch mode (default 10s)
- `--limit` for max threads/checks displayed
- `--context` for lines of diff context around comments
- Config file support for persistent preferences (TOML at `~/.config/gh-ghent/config.toml`)

---

## 13. CI Status Aggregation

**Problem:** A PR can have dozens of check runs. Need a single aggregate status.

**Solution:** Prioritized aggregation — fail > pending > pass:

```go
// internal/adapter/ghcli/reader.go:406-428
func aggregateCI(checks []ghCheck) domain.CIStatus {
    if len(checks) == 0 {
        return domain.CINone
    }
    hasPending := false
    hasFail := false
    for _, c := range checks {
        st := mapCheckStatus(c.Status, c.Conclusion)
        switch st {
        case domain.CIFail:
            hasFail = true
        case domain.CIPending:
            hasPending = true
        }
    }
    if hasFail {
        return domain.CIFail
    }
    if hasPending {
        return domain.CIPending
    }
    return domain.CIPass
}
```

**Check status mapping covers all GitHub states:**

```go
// internal/adapter/ghcli/reader.go:430-448
func mapCheckStatus(status, conclusion string) domain.CIStatus {
    switch status {
    case "COMPLETED":
        switch conclusion {
        case "SUCCESS":                                    return domain.CIPass
        case "FAILURE", "TIMED_OUT", "STARTUP_FAILURE":    return domain.CIFail
        case "SKIPPED", "NEUTRAL":                         return domain.CISkipped
        default:                                           return domain.CIFail
        }
    case "IN_PROGRESS", "QUEUED", "PENDING", "WAITING", "REQUESTED":
        return domain.CIPending
    default:
        return domain.CINone
    }
}
```

**Ghent applicability:** ghent's `checks` command needs the same aggregate + per-check breakdown. The fail-fast behavior for `--watch` maps to: if `hasFail`, show immediately; if only `hasPending`, keep watching.

---

## 14. Performance Targets

From vivecaka's `docs/PRD.md`:

| Metric | Target | Notes |
|--------|--------|-------|
| First render (cached) | <100ms | Show stale data immediately |
| Fresh data load | <500ms | API roundtrip + parsing |
| Keyboard response | <16ms | 60fps equivalent |
| Memory (500 PRs) | <50MB | Streaming parser helps |

**Ghent applicability:** For CLI (non-TUI) usage, targets should be:
- Cached output: <200ms
- Fresh fetch: <2s (GraphQL is slower than REST for complex queries)
- `--watch` poll: configurable, default 10s
- Exit code: immediate (no delay after determination)

---

## 15. Architecture: Clean Layers with Interface Boundaries

Vivecaka uses a clean architecture with clear dependency direction:

```
tui → usecase → domain ← adapter
                  ↑
                plugin
```

**Key interfaces:**

```go
// domain/ports.go (inferred from usage)
type PRReader interface {
    ListPRs(ctx context.Context, repo RepoRef, opts ListOpts) ([]PR, error)
    GetPR(ctx context.Context, repo RepoRef, number int) (*PRDetail, error)
    GetComments(ctx context.Context, repo RepoRef, number int) ([]CommentThread, error)
    GetChecks(ctx context.Context, repo RepoRef, number int) ([]Check, error)
    GetDiff(ctx context.Context, repo RepoRef, number int) (*Diff, error)
}
```

**Ghent applicability:** ghent should define similar interfaces:

```go
// internal/domain/ports.go
type ReviewThreadReader interface {
    GetUnresolvedThreads(ctx context.Context, repo RepoRef, pr int) ([]ReviewThread, error)
    ResolveThread(ctx context.Context, threadID string) error
}

type CheckRunReader interface {
    GetCheckRuns(ctx context.Context, repo RepoRef, pr int) ([]CheckRun, error)
    GetJobLog(ctx context.Context, repo RepoRef, jobID int64) (string, error)
}
```

This allows mocking in tests without an HTTP server, and potential future support for other forges (GitLab, etc.).

---

## Summary: Top 10 Patterns for ghent

| # | Pattern | Source File | Ghent Application |
|---|---------|-------------|-------------------|
| 1 | Dual field lists | `reader.go:12-17` | Avoid expensive fields in list views |
| 2 | errgroup + graceful degradation | `get_pr.go:24-52` | Comments/checks/logs fail independently |
| 3 | Atomic cache writes | `cache.go:48-53` | `--watch` mode cached state |
| 4 | Optimistic UI | `app.go:318-324` | Show stale data while refreshing |
| 5 | Comment threading | `reader.go:259-299` | REST API fallback threading |
| 6 | CI status aggregation | `reader.go:406-448` | `checks` command fail/pending/pass logic |
| 7 | Streaming diff parser | `parser.go:21-75` | Large diff handling if needed |
| 8 | Auto-refresh with countdown | `app.go:275-296` | `--watch` mode implementation |
| 9 | Configurable limits | `config.go:58-83` | Tunable defaults for diverse environments |
| 10 | Interface-based architecture | domain ports | Testable, mockable, future-proof |
