package github

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/indrasvat/gh-ghent/internal/domain"
)

// DefaultPollInterval is the default time between check-status polls.
const DefaultPollInterval = 10 * time.Second

// WatchChecks polls CI check runs until a terminal condition is reached.
// When waitAll is false (default for checks --watch), it exits as soon as
// overall status is pass or fail (fail-fast). When waitAll is true (used by
// summary --watch), it waits until every check has status "completed",
// ensuring the final summary includes all check results and log excerpts.
// On each poll cycle it emits a WatchStatus via the formatter.
// Returns the final OverallStatus or an error.
func (c *Client) WatchChecks(
	ctx context.Context,
	w io.Writer,
	f domain.Formatter,
	owner, repo string,
	pr int,
	interval time.Duration,
	clock func() time.Time,
	waitAll bool,
) (domain.OverallStatus, error) {
	if clock == nil {
		clock = time.Now
	}

	// Track which checks we've already reported as newly completed.
	seen := make(map[int64]string) // checkID → conclusion

	for {
		result, err := c.FetchChecks(ctx, owner, repo, pr)
		if err != nil {
			return "", fmt.Errorf("watch poll: %w", err)
		}

		now := clock()
		status := buildWatchStatus(now, result, seen)

		// Update seen set with newly completed checks.
		for _, ch := range result.Checks {
			if ch.Status == "completed" {
				seen[ch.ID] = ch.Conclusion
			}
		}

		// Determine if this is the final poll.
		var terminal bool
		if waitAll {
			// Wait until every check has completed (no pending checks).
			terminal = result.PendingCount == 0 && len(result.Checks) > 0
		} else {
			// Fail-fast: exit as soon as overall status is pass or fail.
			terminal = result.OverallStatus == domain.StatusPass || result.OverallStatus == domain.StatusFail
		}
		status.Final = terminal

		if err := f.FormatWatchStatus(w, status); err != nil {
			return "", fmt.Errorf("watch format: %w", err)
		}

		if terminal {
			return result.OverallStatus, nil
		}

		// Wait for next poll or context cancellation.
		select {
		case <-ctx.Done():
			return domain.StatusPending, ctx.Err()
		case <-time.After(interval):
		}
	}
}

// buildWatchStatus constructs a WatchStatus from a ChecksResult,
// identifying checks that have completed since the last poll.
func buildWatchStatus(now time.Time, result *domain.ChecksResult, seen map[int64]string) *domain.WatchStatus {
	var events []domain.WatchEvent
	completed := 0

	for _, ch := range result.Checks {
		if ch.Status == "completed" {
			completed++
			// Report as new event only if not previously seen.
			if _, ok := seen[ch.ID]; !ok {
				events = append(events, domain.WatchEvent{
					Name:       ch.Name,
					Status:     ch.Status,
					Conclusion: ch.Conclusion,
					Timestamp:  now,
				})
			}
		}
	}

	return &domain.WatchStatus{
		Timestamp:     now,
		OverallStatus: result.OverallStatus,
		Completed:     completed,
		Total:         len(result.Checks),
		PassCount:     result.PassCount,
		FailCount:     result.FailCount,
		PendingCount:  result.PendingCount,
		Events:        events,
	}
}
