package github

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/indrasvat/gh-ghent/internal/domain"
)

// DefaultPollInterval is the default time between check-status polls.
const DefaultPollInterval = 10 * time.Second

// WatchChecks polls CI check runs until a terminal condition is reached.
// When waitAll is false (default for checks --watch), it exits as soon as
// overall status is pass or fail (fail-fast). When waitAll is true (used by
// status --watch), it waits until every check has status "completed",
// ensuring the final status includes all check results and log excerpts.
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
	pollCount := 0

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

		pollCount++

		// Determine if this is the final poll.
		var terminal bool
		if waitAll {
			// Wait until every check has completed (no pending checks).
			terminal = result.PendingCount == 0 && len(result.Checks) > 0

			// No checks configured at all — treat as vacuous pass after first poll.
			if len(result.Checks) == 0 && pollCount > 1 {
				terminal = true
				result.OverallStatus = domain.StatusPass
				status.OverallStatus = domain.StatusPass
			}
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

// ReviewWatchConfig holds configuration for the review-await phase.
type ReviewWatchConfig struct {
	DebounceWindow            time.Duration   // settle after this idle period (default 30s)
	HardTimeout               time.Duration   // max wait after CI completes (default 5m)
	PollInterval              time.Duration   // how often to poll (default 15s)
	LateActivityGrace         time.Duration   // extension when activity arrives near timeout
	MaxLateActivityExtensions int             // cap to keep the wait bounded
	TailIntervals             []time.Duration // sparse confirmation probes after settle
}

// DefaultReviewWatchConfig returns sensible defaults for review watching.
func DefaultReviewWatchConfig() ReviewWatchConfig {
	return ReviewWatchConfig{
		DebounceWindow:            30 * time.Second,
		HardTimeout:               5 * time.Minute,
		PollInterval:              15 * time.Second,
		LateActivityGrace:         30 * time.Second,
		MaxLateActivityExtensions: 1,
		TailIntervals:             []time.Duration{30 * time.Second, 60 * time.Second},
	}
}

// WatchReviewResult carries the outcome of the review-await phase.
type WatchReviewResult struct {
	Settlement  domain.ReviewSettlement
	HeadChanged bool   // true if head SHA changed during review wait
	NewHeadSHA  string // the new SHA if changed
}

// WatchReviews polls review activity until it settles or times out.
// It uses a lightweight activity probe and fingerprint-based change detection.
// If the PR head SHA changes (new push), it returns immediately with HeadChanged=true
// so the caller can restart the CI watch phase.
//
// baselineHash is an optional fingerprint taken before CI watch started. If the
// initial review-phase snapshot differs from the baseline, that means activity
// happened during CI — the debounce is armed immediately instead of waiting for
// the hard timeout. Pass "" to skip baseline comparison.
func (c *Client) WatchReviews(
	ctx context.Context,
	w io.Writer,
	f domain.Formatter,
	owner, repo string,
	pr int,
	initialHeadSHA string,
	baselineHash string,
	cfg ReviewWatchConfig,
	clock func() time.Time,
) (*WatchReviewResult, error) {
	if clock == nil {
		clock = time.Now
	}
	if cfg.PollInterval == 0 {
		cfg.PollInterval = 15 * time.Second
	}
	if cfg.DebounceWindow == 0 {
		cfg.DebounceWindow = 30 * time.Second
	}
	if cfg.HardTimeout == 0 {
		cfg.HardTimeout = 5 * time.Minute
	}
	if cfg.LateActivityGrace == 0 {
		cfg.LateActivityGrace = cfg.DebounceWindow
	}
	if cfg.MaxLateActivityExtensions == 0 {
		cfg.MaxLateActivityExtensions = 1
	}
	if len(cfg.TailIntervals) == 0 {
		cfg.TailIntervals = []time.Duration{30 * time.Second, 60 * time.Second}
	}

	startAt := clock()
	lastActivityAt := startAt
	activityCount := 0
	consecutiveErrors := 0
	currentInterval := cfg.PollInterval
	deadline := startAt.Add(cfg.HardTimeout)
	maxDeadline := deadline.Add(cfg.LateActivityGrace * time.Duration(cfg.MaxLateActivityExtensions))
	lateExtensions := 0
	tailIndex := -1
	tailProbes := 0
	tailRearmed := false

	// Take initial fingerprint.
	snap, err := c.ProbeActivity(ctx, owner, repo, pr)
	if err != nil {
		return nil, fmt.Errorf("review watch initial probe: %w", err)
	}
	prevHash := Fingerprint(snap)

	// Compare against baseline (taken before CI watch started).
	// If different, activity happened during CI — arm the debounce immediately.
	if baselineHash != "" && prevHash != baselineHash {
		activityCount++
		slog.Debug("review activity detected during CI watch",
			"baseline_hash", baselineHash[:12],
			"current_hash", prevHash[:12])
	} else if snap.ThreadCount > 0 {
		activityCount++
		slog.Debug("existing review threads detected at review-watch start",
			"thread_count", snap.ThreadCount)
	}

	for {
		// Cap poll interval to remaining timeout so we don't oversleep.
		remaining := deadline.Sub(clock())
		if remaining <= 0 {
			// Already past timeout — emit and return immediately.
			now := clock()
			status := &domain.WatchStatus{
				Timestamp:        now,
				OverallStatus:    domain.StatusPass,
				ReviewPhase:      domain.ReviewPhaseTimeout,
				ReviewConfidence: domain.ReviewConfidenceLow,
				ReviewTailProbes: tailProbes,
				Final:            true,
			}
			_ = f.FormatWatchStatus(w, status)
			monitor := domain.NewReviewMonitor(
				domain.ReviewPhaseTimeout,
				activityCount,
				int(now.Sub(startAt).Seconds()),
				tailProbes,
				tailRearmed,
			)
			return &WatchReviewResult{
				Settlement: monitor,
			}, nil
		}
		sleepDur := min(currentInterval, remaining)
		if tailIndex >= 0 && tailIndex < len(cfg.TailIntervals) {
			sleepDur = min(cfg.TailIntervals[tailIndex], remaining)
		}

		// Wait for next poll or context cancellation.
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(sleepDur):
		}

		now := clock()
		snap, err = c.ProbeActivity(ctx, owner, repo, pr)
		if err != nil {
			consecutiveErrors++
			slog.Debug("review watch poll error",
				"error", err,
				"consecutive_errors", consecutiveErrors)

			// Exponential backoff on repeated errors.
			if consecutiveErrors >= 3 {
				currentInterval = min(currentInterval*2, 60*time.Second)
			}

			// Emit status with error info but continue.
			totalElapsed := now.Sub(startAt)
			status := &domain.WatchStatus{
				Timestamp:        now,
				OverallStatus:    domain.StatusPass,
				ReviewPhase:      domain.ReviewPhaseWaiting,
				ReviewIdleSecs:   int(now.Sub(lastActivityAt).Seconds()),
				ReviewTimeoutIn:  max(0, int(deadline.Sub(now).Seconds())),
				ReviewTailProbes: tailProbes,
				Final:            false,
			}
			_ = f.FormatWatchStatus(w, status)

			// Hard timeout still applies during errors.
			if !now.Before(deadline) {
				monitor := domain.NewReviewMonitor(
					domain.ReviewPhaseTimeout,
					activityCount,
					int(totalElapsed.Seconds()),
					tailProbes,
					tailRearmed,
				)
				return &WatchReviewResult{
					Settlement: monitor,
				}, nil
			}
			continue
		}

		// Reset error tracking on success.
		consecutiveErrors = 0
		currentInterval = cfg.PollInterval

		// Check for head SHA change (new push).
		if snap.HeadSHA != initialHeadSHA {
			return &WatchReviewResult{
				HeadChanged: true,
				NewHeadSHA:  snap.HeadSHA,
			}, nil
		}

		// Compare fingerprints.
		newHash := Fingerprint(snap)
		sawActivity := false
		if newHash != prevHash {
			lastActivityAt = now
			activityCount++
			prevHash = newHash
			sawActivity = true
			if tailIndex >= 0 {
				tailIndex = -1
				tailRearmed = true
			}
			if deadline.Sub(now) <= cfg.DebounceWindow && lateExtensions < cfg.MaxLateActivityExtensions {
				extended := deadline.Add(cfg.LateActivityGrace)
				if extended.After(maxDeadline) {
					extended = maxDeadline
				}
				if extended.After(deadline) {
					deadline = extended
					lateExtensions++
				}
			}
		}

		idleDuration := now.Sub(lastActivityAt)
		totalElapsed := now.Sub(startAt)

		// Emit review watch status.
		status := &domain.WatchStatus{
			Timestamp:        now,
			OverallStatus:    domain.StatusPass,
			ReviewPhase:      domain.ReviewPhaseWaiting,
			ReviewIdleSecs:   int(idleDuration.Seconds()),
			ReviewTimeoutIn:  max(0, int(deadline.Sub(now).Seconds())),
			ReviewTailProbes: tailProbes,
		}

		if tailIndex >= 0 && !sawActivity {
			tailProbes++
			status.ReviewConfidence = domain.ReviewConfidenceMedium
			status.ReviewTailProbes = tailProbes
			tailIndex++
			if tailIndex >= len(cfg.TailIntervals) {
				monitor := domain.NewReviewMonitor(
					domain.ReviewPhaseSettled,
					activityCount,
					int(totalElapsed.Seconds()),
					tailProbes,
					tailRearmed,
				)
				status.ReviewPhase = domain.ReviewPhaseSettled
				status.ReviewConfidence = monitor.Confidence
				status.Final = true
				_ = f.FormatWatchStatus(w, status)
				return &WatchReviewResult{Settlement: monitor}, nil
			}
			_ = f.FormatWatchStatus(w, status)
			continue
		}

		// Check debounce: settled when idle for the debounce window.
		// Only debounce after at least one activity change — don't settle on
		// nothing, because the reviewer may still be working (e.g., Codex
		// shows 👀 for 2-4 min before posting comments). If no activity is
		// ever detected, the hard timeout below is the safety valve.
		if activityCount > 0 && idleDuration >= cfg.DebounceWindow {
			if len(cfg.TailIntervals) == 0 {
				monitor := domain.NewReviewMonitor(
					domain.ReviewPhaseSettled,
					activityCount,
					int(totalElapsed.Seconds()),
					tailProbes,
					tailRearmed,
				)
				status.ReviewPhase = domain.ReviewPhaseSettled
				status.ReviewConfidence = monitor.Confidence
				status.Final = true
				_ = f.FormatWatchStatus(w, status)
				return &WatchReviewResult{Settlement: monitor}, nil
			}
			tailIndex = 0
			status.ReviewConfidence = domain.ReviewConfidenceMedium
			_ = f.FormatWatchStatus(w, status)
			continue
		}

		// Check hard timeout.
		if !now.Before(deadline) {
			monitor := domain.NewReviewMonitor(
				domain.ReviewPhaseTimeout,
				activityCount,
				int(totalElapsed.Seconds()),
				tailProbes,
				tailRearmed,
			)
			status.ReviewPhase = domain.ReviewPhaseTimeout
			status.ReviewConfidence = monitor.Confidence
			status.Final = true
			_ = f.FormatWatchStatus(w, status)
			return &WatchReviewResult{Settlement: monitor}, nil
		}

		_ = f.FormatWatchStatus(w, status)
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
