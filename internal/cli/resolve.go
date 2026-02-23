package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/indrasvat/ghent/internal/domain"
	"github.com/indrasvat/ghent/internal/formatter"
	"github.com/indrasvat/ghent/internal/github"
)

func newResolveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resolve",
		Short: "Resolve review threads",
		Long:  "Resolve or unresolve PR review threads. Use --thread for a single thread or --all for bulk resolution.",
		RunE:  runResolve,
	}

	cmd.Flags().String("thread", "", "thread ID to resolve (PRRT_... format)")
	cmd.Flags().Bool("all", false, "resolve all unresolved threads")
	cmd.Flags().Bool("unresolve", false, "unresolve instead of resolve")

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

	if threadID == "" && !all {
		return fmt.Errorf("either --thread or --all is required")
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
	if all {
		results, err = resolveAll(ctx, client, unresolve)
	} else {
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
