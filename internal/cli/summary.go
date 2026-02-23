package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"github.com/indrasvat/gh-ghent/internal/domain"
	"github.com/indrasvat/gh-ghent/internal/formatter"
)

func newSummaryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "summary",
		Short: "PR status dashboard",
		RunE: func(cmd *cobra.Command, args []string) error {
			if Flags.PR == 0 {
				return fmt.Errorf("--pr flag is required")
			}

			owner, repo, err := resolveRepo(Flags.Repo)
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			client := GitHubClient()

			// Parallel fetch using errgroup.
			g, ctx := errgroup.WithContext(ctx)

			var threads *domain.CommentsResult
			var checks *domain.ChecksResult
			var reviews []domain.Review

			g.Go(func() error {
				var fetchErr error
				threads, fetchErr = client.FetchThreads(ctx, owner, repo, Flags.PR)
				if fetchErr != nil {
					return fmt.Errorf("fetch threads: %w", fetchErr)
				}
				return nil
			})

			g.Go(func() error {
				var fetchErr error
				checks, fetchErr = client.FetchChecks(ctx, owner, repo, Flags.PR)
				if fetchErr != nil {
					return fmt.Errorf("fetch checks: %w", fetchErr)
				}
				return nil
			})

			g.Go(func() error {
				var fetchErr error
				reviews, fetchErr = client.FetchReviews(ctx, owner, repo, Flags.PR)
				if fetchErr != nil {
					// Tolerate review fetch failure — reviews are optional.
					reviews = nil
				}
				return nil
			})

			if err := g.Wait(); err != nil {
				return err
			}

			// Merge readiness logic.
			mergeReady := IsMergeReady(threads, checks, reviews)

			result := &domain.SummaryResult{
				PRNumber:     Flags.PR,
				Comments:     *threads,
				Checks:       *checks,
				Reviews:      reviews,
				IsMergeReady: mergeReady,
			}

			f, err := formatter.New(Flags.Format)
			if err != nil {
				return err
			}

			if err := f.FormatSummary(os.Stdout, result); err != nil {
				return fmt.Errorf("format output: %w", err)
			}

			// Exit codes: 0=ready, 1=not ready.
			if !mergeReady {
				os.Exit(1)
			}

			return nil
		},
	}

	return cmd
}

// IsMergeReady determines if a PR is ready to merge based on threads, checks, and reviews.
//
// Conditions:
//  1. No unresolved threads
//  2. All checks pass
//  3. At least 1 approval and no CHANGES_REQUESTED reviews
//
// If reviews is nil (fetch failed), the approval requirement is skipped.
func IsMergeReady(threads *domain.CommentsResult, checks *domain.ChecksResult, reviews []domain.Review) bool {
	// Condition 1: No unresolved threads.
	if threads != nil && threads.UnresolvedCount > 0 {
		return false
	}

	// Condition 2: All checks pass.
	if checks != nil && checks.OverallStatus != domain.StatusPass {
		return false
	}

	// Condition 3: At least 1 approval and no changes_requested.
	// If reviews fetch failed (nil), don't block on approvals.
	if reviews != nil {
		hasApproval := false
		for _, r := range reviews {
			if r.State == domain.ReviewApproved {
				hasApproval = true
			}
			if r.State == domain.ReviewChangesRequested {
				return false
			}
		}
		if !hasApproval {
			return false
		}
	}

	return true
}
