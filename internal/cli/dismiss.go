package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/indrasvat/gh-ghent/internal/domain"
	"github.com/indrasvat/gh-ghent/internal/formatter"
)

func newDismissCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dismiss",
		Short: "Dismiss stale blocking reviews",
		Long: `Dismiss stale CHANGES_REQUESTED pull request reviews.

This command only targets stale blocking reviews whose review commit is
older than the current PR HEAD. It never dismisses current reviews.

Use --review to target one specific stale blocker, or add --author /
--bots-only to narrow the stale set further. Use --dry-run first to
preview exactly what would be dismissed.

Exit codes: 0 = all success, 1 = partial failure, 2 = error.`,
		Example: `  # Preview all stale blockers
  gh ghent dismiss --pr 42 --dry-run

  # Dismiss one specific stale blocker
  gh ghent dismiss --pr 42 --review PRR_kwDO... --message "superseded by current HEAD"

  # Dismiss stale bot blockers only
  gh ghent dismiss --pr 42 --bots-only --message "superseded by current HEAD"

  # Narrow to one author
  gh ghent dismiss --pr 42 --author sonarcloud --dry-run`,
		RunE: runDismiss,
	}

	cmd.Flags().String("review", "", "review node ID or numeric review ID to dismiss")
	cmd.Flags().String("author", "", "dismiss stale blocking reviews from a specific author")
	cmd.Flags().Bool("bots-only", false, "only dismiss stale blocking reviews from bot accounts")
	cmd.Flags().String("message", "", "dismissal message sent to GitHub (required unless --dry-run)")
	cmd.Flags().Bool("dry-run", false, "show what would be dismissed without executing")

	return cmd
}

type dismissClient interface {
	domain.ReviewFetcher
	domain.ReviewDismisser
}

func runDismiss(cmd *cobra.Command, _ []string) error {
	if Flags.PR == 0 {
		return fmt.Errorf("--pr flag is required")
	}

	reviewID, err := cmd.Flags().GetString("review")
	if err != nil {
		return err
	}
	author, err := cmd.Flags().GetString("author")
	if err != nil {
		return err
	}
	botsOnly, err := cmd.Flags().GetBool("bots-only")
	if err != nil {
		return err
	}
	message, err := cmd.Flags().GetString("message")
	if err != nil {
		return err
	}
	dryRun, err := cmd.Flags().GetBool("dry-run")
	if err != nil {
		return err
	}
	if !dryRun && message == "" {
		return fmt.Errorf("--message is required unless --dry-run is set")
	}

	owner, repo, err := resolveRepo(Flags.Repo)
	if err != nil {
		return err
	}

	ctx := cmd.Context()
	client := GitHubClient()
	results, err := buildDismissResults(ctx, client, owner, repo, Flags.PR, reviewID, author, botsOnly, message, dryRun)
	if err != nil {
		return err
	}

	f, err := formatter.New(Flags.Format)
	if err != nil {
		return err
	}
	if err := f.FormatDismissResults(os.Stdout, results); err != nil {
		return fmt.Errorf("format output: %w", err)
	}

	if exitCode := dismissExitCode(results); exitCode != 0 {
		os.Exit(exitCode)
	}

	return nil
}

func buildDismissResults(
	ctx context.Context,
	client dismissClient,
	owner, repo string,
	pr int,
	selector, author string,
	botsOnly bool,
	message string,
	dryRun bool,
) (*domain.DismissResults, error) {
	reviews, err := client.FetchReviews(ctx, owner, repo, pr)
	if err != nil {
		return nil, fmt.Errorf("fetch reviews: %w", err)
	}

	selected, err := selectDismissReviews(reviews, selector, author, botsOnly)
	if err != nil {
		return nil, err
	}

	results := &domain.DismissResults{
		Results: []domain.DismissResult{},
		DryRun:  dryRun,
	}
	for _, review := range selected {
		if dryRun {
			results.Results = append(results.Results, domain.DismissResult{
				ReviewID:    review.ID,
				DatabaseID:  review.DatabaseID,
				Author:      review.Author,
				IsBot:       review.IsBot,
				State:       review.State,
				CommitID:    review.CommitID,
				IsStale:     review.IsStale,
				Dismissed:   false,
				Action:      "would_dismiss",
				SubmittedAt: review.SubmittedAt,
			})
			results.SuccessCount++
			continue
		}

		result, dismissErr := client.DismissReview(ctx, owner, repo, pr, review, message)
		if dismissErr != nil {
			results.FailureCount++
			results.Errors = append(results.Errors, domain.DismissError{
				ReviewID: review.ID,
				Message:  dismissErr.Error(),
			})
			continue
		}

		results.Results = append(results.Results, *result)
		results.SuccessCount++
	}

	return results, nil
}

func dismissExitCode(results *domain.DismissResults) int {
	if results == nil {
		return 0
	}
	if results.FailureCount > 0 && results.SuccessCount > 0 {
		return 1
	}
	if results.FailureCount > 0 {
		return 2
	}
	return 0
}
