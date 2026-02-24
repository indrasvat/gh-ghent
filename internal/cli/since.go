package cli

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/indrasvat/gh-ghent/internal/domain"
)

// ParseSince parses a --since value into a time.Time.
// Accepts ISO 8601 (RFC3339) timestamps or relative durations: 1h, 30m, 2d, 1w.
// Returns zero time if input is empty.
func ParseSince(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}

	// Try RFC3339 first.
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}

	// Try relative duration: number + unit suffix.
	dur, err := parseRelativeDuration(s)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid --since value %q: expected ISO 8601 (e.g. 2026-02-22T00:00:00Z) or relative duration (e.g. 1h, 30m, 2d): %w", s, err)
	}

	return time.Now().Add(-dur), nil
}

// parseRelativeDuration parses strings like "1h", "30m", "2d", "1w" into a time.Duration.
func parseRelativeDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if len(s) < 2 {
		return 0, fmt.Errorf("duration too short: %q", s)
	}

	unit := s[len(s)-1]
	numStr := s[:len(s)-1]

	n, err := strconv.Atoi(numStr)
	if err != nil {
		return 0, fmt.Errorf("invalid number in duration %q: %w", s, err)
	}
	if n <= 0 {
		return 0, fmt.Errorf("duration must be positive: %q", s)
	}

	switch unit {
	case 'm':
		return time.Duration(n) * time.Minute, nil
	case 'h':
		return time.Duration(n) * time.Hour, nil
	case 'd':
		return time.Duration(n) * 24 * time.Hour, nil
	case 'w':
		return time.Duration(n) * 7 * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("unknown unit %q in %q (use m, h, d, or w)", string(unit), s)
	}
}

// FilterThreadsBySince filters threads, keeping only those where the newest comment
// was created at or after the since timestamp. Updates counts on the result.
func FilterThreadsBySince(result *domain.CommentsResult, since time.Time) {
	if since.IsZero() || result == nil {
		return
	}

	result.Since = since.Format(time.RFC3339)

	filtered := result.Threads[:0]
	for _, t := range result.Threads {
		if threadNewestAfter(t, since) {
			filtered = append(filtered, t)
		}
	}
	result.Threads = filtered

	// Recount based on filtered threads.
	unresolved := 0
	resolved := 0
	for _, t := range result.Threads {
		if t.IsResolved {
			resolved++
		} else {
			unresolved++
		}
	}
	result.UnresolvedCount = unresolved
	result.ResolvedCount = resolved
	result.TotalCount = len(result.Threads)
}

// threadNewestAfter returns true if the thread's newest comment is at or after t.
func threadNewestAfter(thread domain.ReviewThread, t time.Time) bool {
	if len(thread.Comments) == 0 {
		return false
	}
	newest := thread.Comments[0].CreatedAt
	for _, c := range thread.Comments[1:] {
		if c.CreatedAt.After(newest) {
			newest = c.CreatedAt
		}
	}
	return !newest.Before(t)
}

// FilterChecksBySince filters check runs, keeping only those where CompletedAt >= since
// (or StartedAt for still-running checks). Updates counts on the result.
func FilterChecksBySince(result *domain.ChecksResult, since time.Time) {
	if since.IsZero() || result == nil {
		return
	}

	result.Since = since.Format(time.RFC3339)

	filtered := result.Checks[:0]
	for _, ch := range result.Checks {
		ts := ch.CompletedAt
		if ts.IsZero() {
			ts = ch.StartedAt
		}
		if !ts.Before(since) {
			filtered = append(filtered, ch)
		}
	}
	result.Checks = filtered

	// Recount.
	pass, fail, pending := 0, 0, 0
	for _, ch := range result.Checks {
		switch {
		case ch.Status != "completed":
			pending++
		case ch.Conclusion == "success" || ch.Conclusion == "neutral" || ch.Conclusion == "skipped":
			pass++
		default:
			fail++
		}
	}
	result.PassCount = pass
	result.FailCount = fail
	result.PendingCount = pending
	result.OverallStatus = domain.AggregateStatus(checksToStatuses(result.Checks))
}

// checksToStatuses converts a slice of check runs to their aggregate statuses.
func checksToStatuses(checks []domain.CheckRun) []domain.OverallStatus {
	statuses := make([]domain.OverallStatus, 0, len(checks))
	for _, ch := range checks {
		switch {
		case ch.Status != "completed":
			statuses = append(statuses, domain.StatusPending)
		case ch.Conclusion == "success" || ch.Conclusion == "neutral" || ch.Conclusion == "skipped":
			statuses = append(statuses, domain.StatusPass)
		default:
			statuses = append(statuses, domain.StatusFail)
		}
	}
	return statuses
}
