package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"github.com/indrasvat/gh-ghent/internal/domain"
	"github.com/indrasvat/gh-ghent/internal/formatter"
	ghub "github.com/indrasvat/gh-ghent/internal/github"
	"github.com/indrasvat/gh-ghent/internal/tui"
)

func newSummaryCmd() *cobra.Command {
	var (
		compact  bool
		withLogs bool
		quiet    bool
		watch    bool
	)

	cmd := &cobra.Command{
		Use:   "summary",
		Short: "PR status dashboard",
		Long: `Show a combined status dashboard for a pull request.

Fetches review threads, CI checks, and approvals in parallel, then
displays a unified view with merge-readiness assessment. In TTY mode,
shows KPI cards and section summaries. In pipe mode, outputs all
sections in a single structured response.

Use --logs to include failing job log excerpts in output.
Use --watch to poll until all checks complete, then output full summary.
Use --quiet for silent exit on merge-ready (exit 0), full output on not-ready (exit 1).

Merge-ready when: no unresolved threads + all checks pass + approved.

Exit codes: 0 = merge-ready, 1 = not merge-ready.`,
		Example: `  # Interactive dashboard
  gh ghent summary --pr 42

  # Agent: check merge readiness
  gh ghent summary --pr 42 --format json | jq '.is_merge_ready'

  # Full status with failure diagnostics
  gh ghent summary --pr 42 --logs --format json

  # Wait for CI, get full report
  gh ghent summary --pr 42 --watch --format json

  # Silent merge-readiness gate
  gh ghent summary --pr 42 --quiet

  # Compact one-line-per-thread digest
  gh ghent summary --pr 42 --compact --format json`,
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

			// Watch mode: poll until CI terminal status, then output full summary.
			if watch {
				// TTY → launch watch TUI (same as checks --watch).
				if Flags.IsTTY {
					repoStr := owner + "/" + repo
					fetchFn := func() (*domain.ChecksResult, error) {
						return client.FetchChecks(ctx, owner, repo, Flags.PR)
					}
					return launchTUI(tui.ViewWatch,
						withRepo(repoStr), withPR(Flags.PR),
						withWatchFetch(fetchFn, ghub.DefaultPollInterval),
					)
				}

				// Non-TTY: watch status → stderr, final summary → stdout.
				f, fErr := formatter.New(Flags.Format)
				if fErr != nil {
					return fErr
				}

				_, watchErr := client.WatchChecks(
					ctx, os.Stderr, f,
					owner, repo, Flags.PR,
					ghub.DefaultPollInterval, nil,
				)
				if watchErr != nil {
					return fmt.Errorf("watch checks: %w", watchErr)
				}
				// Fall through to fetch full summary data below.
			}

			// TTY (non-watch) → launch TUI immediately with async fetch.
			if !watch && Flags.IsTTY {
				repoStr := owner + "/" + repo
				sinceFilter := Flags.Since // capture for closures
				return launchTUI(tui.ViewSummary,
					withRepo(repoStr), withPR(Flags.PR),
					withAsyncFetch(
						func() (*domain.CommentsResult, error) {
							result, err := client.FetchThreads(ctx, owner, repo, Flags.PR)
							if err == nil {
								FilterThreadsBySince(result, sinceFilter)
							}
							return result, err
						},
						func() (*domain.ChecksResult, error) {
							result, err := client.FetchChecks(ctx, owner, repo, Flags.PR)
							if err == nil {
								FilterChecksBySince(result, sinceFilter)
							}
							return result, err
						},
						func() ([]domain.Review, error) {
							return client.FetchReviews(ctx, owner, repo, Flags.PR)
						},
					),
				)
			}

			// Non-TTY / pipe mode: block until all data is fetched.
			cmdCtx := ctx // preserve for post-errgroup log fetching
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

			// Fetch logs for failing checks when --logs is set (or implied by --watch).
			// Use cmdCtx (not ctx) because errgroup's derived context is cancelled
			// after g.Wait() returns.
			if withLogs || watch {
				for i := range checks.Checks {
					ch := &checks.Checks[i]
					if ch.Conclusion != "failure" {
						continue
					}
					logText, logErr := client.FetchJobLog(cmdCtx, owner, repo, ch.ID)
					if logErr != nil {
						continue // graceful degradation
					}
					ch.LogExcerpt = ghub.ExtractErrorLines(logText)
				}
			}

			// Merge readiness logic. If review fetch failed, not merge-ready.
			mergeReady := !reviewFetchFailed && IsMergeReady(threads, checks, reviews)

			// --quiet: silent exit on merge-ready, full output on not-ready.
			if quiet && mergeReady {
				return nil // exit 0, no output
			}

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
	cmd.Flags().BoolVar(&withLogs, "logs", false, "include failing job log excerpts in output")
	cmd.Flags().BoolVar(&quiet, "quiet", false, "silent on merge-ready (exit 0), full output on not-ready (exit 1)")
	cmd.Flags().BoolVar(&watch, "watch", false, "poll until all checks complete, then output full summary")

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
