package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"github.com/indrasvat/gh-ghent/internal/domain"
	"github.com/indrasvat/gh-ghent/internal/formatter"
	"github.com/indrasvat/gh-ghent/internal/tui"
)

func newSummaryCmd() *cobra.Command {
	var compact bool

	cmd := &cobra.Command{
		Use:   "summary",
		Short: "PR status dashboard",
		Long: `Show a combined status dashboard for a pull request.

Fetches review threads, CI checks, and approvals in parallel, then
displays a unified view with merge-readiness assessment. In TTY mode,
shows KPI cards and section summaries. In pipe mode, outputs all
sections in a single structured response.

Merge-ready when: no unresolved threads + all checks pass + approved.

Exit codes: 0 = merge-ready, 1 = not merge-ready.`,
		Example: `  # Interactive dashboard
  gh ghent summary --pr 42

  # Agent: check merge readiness
  gh ghent summary --pr 42 --format json | jq '.is_merge_ready'

  # Compact one-line-per-thread digest
  gh ghent summary --pr 42 --compact --format json

  # Full status as markdown
  gh ghent summary -R owner/repo --pr 42 --format md`,
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
			var reviewFetchFailed bool

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
					// Tolerate review fetch failure — degrade gracefully, but
					// mark as failed so merge-readiness defaults to not-ready.
					reviews = nil
					reviewFetchFailed = true
				}
				return nil
			})

			if err := g.Wait(); err != nil {
				return err
			}

			// Apply --since filter (no-op if not set).
			FilterThreadsBySince(threads, Flags.Since)
			FilterChecksBySince(checks, Flags.Since)

			// TTY → launch TUI; non-TTY / --no-tui → pipe mode.
			if Flags.IsTTY {
				repoStr := owner + "/" + repo
				return launchTUI(tui.ViewSummary,
					withRepo(repoStr), withPR(Flags.PR),
					withComments(threads), withChecks(checks), withReviews(reviews),
				)
			}

			// Merge readiness logic. If review fetch failed, not merge-ready.
			mergeReady := !reviewFetchFailed && IsMergeReady(threads, checks, reviews)

			now := time.Now()

			result := &domain.SummaryResult{
				PRNumber:     Flags.PR,
				Comments:     *threads,
				Checks:       *checks,
				Reviews:      reviews,
				IsMergeReady: mergeReady,
				PRAge:        computePRAge(threads, reviews, now),
				LastUpdate:   computeLastUpdate(threads, reviews, now),
				ReviewCycles: computeReviewCycles(reviews),
			}

			f, err := formatter.New(Flags.Format)
			if err != nil {
				return err
			}

			if compact {
				if err := f.FormatCompactSummary(os.Stdout, result); err != nil {
					return fmt.Errorf("format output: %w", err)
				}
			} else {
				if err := f.FormatSummary(os.Stdout, result); err != nil {
					return fmt.Errorf("format output: %w", err)
				}
			}

			// Exit codes: 0=ready, 1=not ready.
			if !mergeReady {
				os.Exit(1)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&compact, "compact", false, "one-line-per-thread compact digest (optimized for agents)")

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

// computePRAge derives PR age from the earliest timestamp in threads/reviews.
func computePRAge(threads *domain.CommentsResult, reviews []domain.Review, now time.Time) string {
	earliest := findEarliestTimestamp(threads, reviews)
	if earliest.IsZero() {
		return ""
	}
	return formatRelativeTime(now.Sub(earliest))
}

// computeLastUpdate finds the most recent comment or review timestamp.
func computeLastUpdate(threads *domain.CommentsResult, reviews []domain.Review, now time.Time) string {
	var latest time.Time

	if threads != nil {
		for _, t := range threads.Threads {
			for _, c := range t.Comments {
				if c.CreatedAt.After(latest) {
					latest = c.CreatedAt
				}
			}
		}
	}

	for _, r := range reviews {
		if r.SubmittedAt.After(latest) {
			latest = r.SubmittedAt
		}
	}

	if latest.IsZero() {
		return ""
	}
	return formatRelativeTime(now.Sub(latest))
}

// computeReviewCycles counts distinct review rounds (unique dates of review submissions).
func computeReviewCycles(reviews []domain.Review) int {
	if len(reviews) == 0 {
		return 0
	}

	seen := make(map[string]bool)
	for _, r := range reviews {
		day := r.SubmittedAt.Format("2006-01-02")
		seen[day] = true
	}
	return len(seen)
}

// findEarliestTimestamp returns the oldest timestamp across threads and reviews.
func findEarliestTimestamp(threads *domain.CommentsResult, reviews []domain.Review) time.Time {
	var earliest time.Time

	if threads != nil {
		for _, t := range threads.Threads {
			for _, c := range t.Comments {
				if earliest.IsZero() || c.CreatedAt.Before(earliest) {
					earliest = c.CreatedAt
				}
			}
		}
	}

	for _, r := range reviews {
		if earliest.IsZero() || r.SubmittedAt.Before(earliest) {
			earliest = r.SubmittedAt
		}
	}

	return earliest
}

// formatRelativeTime formats a duration as a human-friendly relative string.
func formatRelativeTime(d time.Duration) string {
	switch {
	case d < time.Minute:
		return "<1m"
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	default:
		return fmt.Sprintf("%dw", int(d.Hours()/(24*7)))
	}
}
