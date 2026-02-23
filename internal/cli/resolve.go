package cli

import (
	"context"
	"fmt"
	"os"
	"path"

	"github.com/spf13/cobra"

	"github.com/indrasvat/gh-ghent/internal/domain"
	"github.com/indrasvat/gh-ghent/internal/formatter"
	"github.com/indrasvat/gh-ghent/internal/github"
	"github.com/indrasvat/gh-ghent/internal/tui"
)

func newResolveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resolve",
		Short: "Resolve review threads",
		Long: `Resolve or unresolve PR review threads.

Use --thread to resolve a single thread by ID, or --all to resolve
every unresolved thread in bulk. Use --file and --author to batch
resolve threads matching a glob pattern or author login.
Add --unresolve to reverse the operation.

Use --dry-run to preview what would be resolved without executing.

Requires write permission on the repository. Respects per-thread
viewerCanResolve / viewerCanUnresolve permissions from GitHub.

Exit codes: 0 = all success, 1 = partial failure, 2 = total failure.`,
		Example: `  # Resolve a single thread
  gh ghent resolve --pr 42 --thread PRRT_abc123

  # Resolve all unresolved threads
  gh ghent resolve --pr 42 --all

  # Batch resolve by file pattern
  gh ghent resolve --pr 42 --file "internal/api/*.go"

  # Batch resolve by author
  gh ghent resolve --pr 42 --author reviewer1

  # Combined filters (intersection)
  gh ghent resolve --pr 42 --file "*.go" --author reviewer1

  # Dry run: preview without resolving
  gh ghent resolve --pr 42 --file "*.go" --dry-run

  # Unresolve a thread (reopen for discussion)
  gh ghent resolve --pr 42 --thread PRRT_abc123 --unresolve

  # Agent workflow: resolve all, check result
  gh ghent resolve --pr 42 --all --format json | jq '.success_count'`,
		RunE: runResolve,
	}

	cmd.Flags().String("thread", "", "thread ID to resolve (PRRT_... node ID)")
	cmd.Flags().Bool("all", false, "resolve all unresolved threads in the PR")
	cmd.Flags().Bool("unresolve", false, "unresolve instead of resolve")
	cmd.Flags().String("file", "", "resolve threads in files matching glob (e.g., 'internal/api/*.go')")
	cmd.Flags().String("author", "", "resolve threads started by a specific author")
	cmd.Flags().Bool("dry-run", false, "show what would be resolved without executing")

	return cmd
}

func runResolve(cmd *cobra.Command, _ []string) error {
	if Flags.PR == 0 {
		return fmt.Errorf("--pr flag is required")
	}

	threadID, err := cmd.Flags().GetString("thread")
	if err != nil {
		return err
	}
	all, err := cmd.Flags().GetBool("all")
	if err != nil {
		return err
	}
	unresolve, err := cmd.Flags().GetBool("unresolve")
	if err != nil {
		return err
	}
	fileGlob, err := cmd.Flags().GetString("file")
	if err != nil {
		return err
	}
	author, err := cmd.Flags().GetString("author")
	if err != nil {
		return err
	}
	dryRun, err := cmd.Flags().GetBool("dry-run")
	if err != nil {
		return err
	}

	hasBatchFilter := fileGlob != "" || author != ""

	// Validate flag combinations.
	if threadID != "" && hasBatchFilter {
		return fmt.Errorf("--thread cannot be combined with --file or --author")
	}
	if dryRun && !hasBatchFilter && !all {
		return fmt.Errorf("--dry-run requires --file, --author, or --all")
	}

	// TTY without explicit flags → launch interactive resolve TUI.
	if Flags.IsTTY && threadID == "" && !all && !hasBatchFilter {
		owner, repo, repoErr := resolveRepo(Flags.Repo)
		if repoErr != nil {
			return repoErr
		}
		ctx := cmd.Context()
		client := GitHubClient()
		threads, fetchErr := client.FetchThreads(ctx, owner, repo, Flags.PR)
		if fetchErr != nil {
			return fmt.Errorf("fetch threads: %w", fetchErr)
		}
		repoStr := owner + "/" + repo
		resolverFn := func(threadID string) error {
			_, err := client.ResolveThread(ctx, threadID)
			return err
		}
		return launchTUI(tui.ViewResolve,
			withRepo(repoStr), withPR(Flags.PR),
			withComments(threads),
			withResolver(resolverFn),
		)
	}

	if threadID == "" && !all && !hasBatchFilter {
		return fmt.Errorf("either --thread, --all, --file, or --author is required")
	}
	if threadID != "" && all {
		return fmt.Errorf("--thread and --all are mutually exclusive")
	}

	f, err := formatter.New(Flags.Format)
	if err != nil {
		return err
	}

	ctx := cmd.Context()
	client := GitHubClient()

	var results *domain.ResolveResults
	switch {
	case hasBatchFilter:
		results, err = resolveBatch(ctx, client, fileGlob, author, unresolve, dryRun)
	case all:
		if dryRun {
			results, err = resolveBatchAll(ctx, client, unresolve)
		} else {
			results, err = resolveAll(ctx, client, unresolve)
		}
	default:
		results, err = resolveSingle(ctx, client, threadID, unresolve)
	}
	if err != nil {
		return err
	}

	if err := f.FormatResolveResults(os.Stdout, results); err != nil {
		return fmt.Errorf("format output: %w", err)
	}

	// Exit codes per PRD §6.4: 0=all success, 1=partial failure, 2=error
	if results.FailureCount > 0 && results.SuccessCount > 0 {
		os.Exit(1)
	}
	if results.FailureCount > 0 && results.SuccessCount == 0 {
		os.Exit(2)
	}

	return nil
}

func resolveSingle(ctx context.Context, client *github.Client, threadID string, unresolve bool) (*domain.ResolveResults, error) {
	result, msg := doResolve(ctx, client, threadID, unresolve)
	if msg != "" {
		return &domain.ResolveResults{
			FailureCount: 1,
			Errors: []domain.ResolveError{
				{ThreadID: threadID, Message: msg},
			},
		}, nil
	}

	return &domain.ResolveResults{
		Results:      []domain.ResolveResult{*result},
		SuccessCount: 1,
	}, nil
}

// doResolve executes a single resolve/unresolve mutation, returning the result
// or an error message string.
func doResolve(ctx context.Context, client *github.Client, threadID string, unresolve bool) (*domain.ResolveResult, string) {
	var result *domain.ResolveResult
	var err error

	if unresolve {
		result, err = client.UnresolveThread(ctx, threadID)
	} else {
		result, err = client.ResolveThread(ctx, threadID)
	}
	if err != nil {
		return nil, err.Error()
	}
	return result, ""
}

// matchesFilters returns true if the thread matches all specified filters.
// An empty filter is a no-op (matches everything).
func matchesFilters(t domain.ReviewThread, fileGlob, author string) bool {
	if fileGlob != "" {
		matched, _ := path.Match(fileGlob, t.Path)
		if !matched {
			return false
		}
	}
	if author != "" && len(t.Comments) > 0 {
		if t.Comments[0].Author != author {
			return false
		}
	}
	return true
}

// resolveBatch fetches threads and applies --file/--author filters before
// resolving. Supports --dry-run to preview without executing.
func resolveBatch(ctx context.Context, client *github.Client, fileGlob, author string, unresolve, dryRun bool) (*domain.ResolveResults, error) {
	owner, repo, err := resolveRepo(Flags.Repo)
	if err != nil {
		return nil, err
	}

	var threads *domain.CommentsResult
	if unresolve {
		threads, err = client.FetchResolvedThreads(ctx, owner, repo, Flags.PR)
	} else {
		threads, err = client.FetchThreads(ctx, owner, repo, Flags.PR)
	}
	if err != nil {
		return nil, fmt.Errorf("fetch threads: %w", err)
	}

	action := "resolved"
	wouldAction := "would_resolve"
	if unresolve {
		action = "unresolved"
		wouldAction = "would_unresolve"
	}

	results := &domain.ResolveResults{DryRun: dryRun}

	for _, t := range threads.Threads {
		if !matchesFilters(t, fileGlob, author) {
			continue
		}

		if dryRun {
			results.Results = append(results.Results, domain.ResolveResult{
				ThreadID: t.ID,
				Path:     t.Path,
				Line:     t.Line,
				Action:   wouldAction,
			})
			results.SuccessCount++
			continue
		}

		// Permission check
		if !unresolve && !t.ViewerCanResolve {
			results.SkippedCount++
			results.Errors = append(results.Errors, domain.ResolveError{
				ThreadID: t.ID,
				Message:  fmt.Sprintf("permission denied: cannot resolve thread at %s:%d", t.Path, t.Line),
			})
			continue
		}
		if unresolve && !t.ViewerCanUnresolve {
			results.SkippedCount++
			results.Errors = append(results.Errors, domain.ResolveError{
				ThreadID: t.ID,
				Message:  fmt.Sprintf("permission denied: cannot unresolve thread at %s:%d", t.Path, t.Line),
			})
			continue
		}

		result, msg := doResolve(ctx, client, t.ID, unresolve)
		if msg != "" {
			results.FailureCount++
			results.Errors = append(results.Errors, domain.ResolveError{
				ThreadID: t.ID,
				Message:  msg,
			})
			continue
		}

		result.Action = action
		results.Results = append(results.Results, *result)
		results.SuccessCount++
	}

	return results, nil
}

// resolveBatchAll is --all --dry-run: preview all threads without executing.
func resolveBatchAll(ctx context.Context, client *github.Client, unresolve bool) (*domain.ResolveResults, error) {
	return resolveBatch(ctx, client, "", "", unresolve, true)
}

func resolveAll(ctx context.Context, client *github.Client, unresolve bool) (*domain.ResolveResults, error) {
	owner, repo, err := resolveRepo(Flags.Repo)
	if err != nil {
		return nil, err
	}

	var threads *domain.CommentsResult
	if unresolve {
		threads, err = client.FetchResolvedThreads(ctx, owner, repo, Flags.PR)
	} else {
		threads, err = client.FetchThreads(ctx, owner, repo, Flags.PR)
	}
	if err != nil {
		return nil, fmt.Errorf("fetch threads: %w", err)
	}

	results := &domain.ResolveResults{}

	for _, t := range threads.Threads {
		// Permission check: skip threads where viewer lacks permission
		if !unresolve && !t.ViewerCanResolve {
			results.FailureCount++
			results.Errors = append(results.Errors, domain.ResolveError{
				ThreadID: t.ID,
				Message:  fmt.Sprintf("permission denied: cannot resolve thread at %s:%d", t.Path, t.Line),
			})
			continue
		}
		if unresolve && !t.ViewerCanUnresolve {
			results.FailureCount++
			results.Errors = append(results.Errors, domain.ResolveError{
				ThreadID: t.ID,
				Message:  fmt.Sprintf("permission denied: cannot unresolve thread at %s:%d", t.Path, t.Line),
			})
			continue
		}

		result, msg := doResolve(ctx, client, t.ID, unresolve)
		if msg != "" {
			results.FailureCount++
			results.Errors = append(results.Errors, domain.ResolveError{
				ThreadID: t.ID,
				Message:  msg,
			})
			continue
		}

		results.Results = append(results.Results, *result)
		results.SuccessCount++
	}

	return results, nil
}
