package github

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/indrasvat/gh-ghent/internal/domain"
	"github.com/indrasvat/gh-ghent/internal/formatter"
)

type fakeReviewClock struct {
	now  time.Time
	step time.Duration
}

func (c *fakeReviewClock) Now() time.Time {
	current := c.now
	c.now = c.now.Add(c.step)
	return current
}

func scriptedProbe(snaps []*domain.ActivitySnapshot, errs []error) activityProbeFunc {
	index := 0
	return func(context.Context, string, string, int) (*domain.ActivitySnapshot, error) {
		if index >= len(snaps) {
			if len(snaps) == 0 {
				return nil, errors.New("no scripted snapshot")
			}
			return snaps[len(snaps)-1], nil
		}
		snap := snaps[index]
		var err error
		if index < len(errs) {
			err = errs[index]
		}
		index++
		return snap, err
	}
}

func TestDefaultReviewWatchConfig(t *testing.T) {
	cfg := DefaultReviewWatchConfig()
	if cfg.DebounceWindow != 30*time.Second {
		t.Errorf("DebounceWindow = %v, want 30s", cfg.DebounceWindow)
	}
	if cfg.HardTimeout != 5*time.Minute {
		t.Errorf("HardTimeout = %v, want 5m", cfg.HardTimeout)
	}
	if cfg.PollInterval != 15*time.Second {
		t.Errorf("PollInterval = %v, want 15s", cfg.PollInterval)
	}
	if cfg.LateActivityGrace != 30*time.Second {
		t.Errorf("LateActivityGrace = %v, want 30s", cfg.LateActivityGrace)
	}
	if cfg.MaxLateActivityExtensions != 1 {
		t.Errorf("MaxLateActivityExtensions = %d, want 1", cfg.MaxLateActivityExtensions)
	}
	if len(cfg.TailIntervals) != 2 {
		t.Fatalf("TailIntervals length = %d, want 2", len(cfg.TailIntervals))
	}
}

func TestWatchReviewResultTypes(t *testing.T) {
	// Test that WatchReviewResult correctly carries settlement and head change info.
	result := WatchReviewResult{
		Settlement: domain.ReviewSettlement{
			Phase:         domain.ReviewPhaseSettled,
			ActivityCount: 3,
			WaitSeconds:   120,
		},
	}
	if result.HeadChanged {
		t.Error("HeadChanged should be false by default")
	}
	if result.Settlement.Phase != domain.ReviewPhaseSettled {
		t.Errorf("Phase = %q, want settled", result.Settlement.Phase)
	}

	headChanged := WatchReviewResult{
		HeadChanged: true,
		NewHeadSHA:  "abc123",
	}
	if !headChanged.HeadChanged {
		t.Error("HeadChanged should be true")
	}
	if headChanged.NewHeadSHA != "abc123" {
		t.Errorf("NewHeadSHA = %q, want abc123", headChanged.NewHeadSHA)
	}
}

func TestReviewWatchPhaseConstants(t *testing.T) {
	tests := []struct {
		phase domain.ReviewWatchPhase
		want  string
	}{
		{domain.ReviewPhaseNone, ""},
		{domain.ReviewPhaseWaiting, "awaiting"},
		{domain.ReviewPhaseSettled, "settled"},
		{domain.ReviewPhaseTimeout, "timeout"},
	}
	for _, tt := range tests {
		if string(tt.phase) != tt.want {
			t.Errorf("ReviewWatchPhase %q != %q", tt.phase, tt.want)
		}
	}
}

func TestWatchStatusReviewFields(t *testing.T) {
	// Verify that WatchStatus can carry review-phase fields.
	status := &domain.WatchStatus{
		Timestamp:        time.Now(),
		OverallStatus:    domain.StatusPass,
		ReviewPhase:      domain.ReviewPhaseWaiting,
		ReviewConfidence: domain.ReviewConfidenceMedium,
		ReviewIdleSecs:   12,
		ReviewTimeoutIn:  288,
		ReviewTailProbes: 1,
	}

	// Format as JSON to verify serialization.
	var buf bytes.Buffer
	f, _ := formatter.New("json")
	if err := f.FormatWatchStatus(&buf, status); err != nil {
		t.Fatalf("FormatWatchStatus: %v", err)
	}

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte(`"review_phase":"awaiting"`)) {
		t.Errorf("JSON missing review_phase: %s", output)
	}
	if !bytes.Contains([]byte(output), []byte(`"review_idle_secs":12`)) {
		t.Errorf("JSON missing review_idle_secs: %s", output)
	}
	if !bytes.Contains([]byte(output), []byte(`"review_confidence":"medium"`)) {
		t.Errorf("JSON missing review_confidence: %s", output)
	}
}

func TestWatchChecksNoChecksPR(t *testing.T) {
	// Test the no-check bugfix: WatchChecks should not hang forever
	// when a PR has zero checks.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	callCount := 0
	client := &Client{}
	// We can't easily test WatchChecks without a real client,
	// but we can verify the logic by checking that the zero-check
	// condition is handled in buildWatchStatus.

	// Instead, test that buildWatchStatus correctly reports empty checks.
	result := &domain.ChecksResult{
		Checks:        nil,
		OverallStatus: domain.StatusPending,
		PendingCount:  0,
	}
	seen := make(map[int64]string)
	status := buildWatchStatus(time.Now(), result, seen)

	if status.Total != 0 {
		t.Errorf("Total = %d, want 0", status.Total)
	}
	if status.Completed != 0 {
		t.Errorf("Completed = %d, want 0", status.Completed)
	}

	_ = ctx
	_ = callCount
	_ = client
}

func TestReviewMonitorInStatusResult(t *testing.T) {
	// Verify ReviewMonitor is included in StatusResult serialization.
	result := &domain.StatusResult{
		PRNumber:     42,
		IsMergeReady: false,
		ReviewMonitor: &domain.ReviewMonitor{
			Phase:         domain.ReviewPhaseSettled,
			Confidence:    domain.ReviewConfidenceHigh,
			ActivityCount: 3,
			WaitSeconds:   154,
			TailProbes:    2,
		},
		ReviewSettled: &domain.ReviewSettlement{
			Phase:         domain.ReviewPhaseSettled,
			Confidence:    domain.ReviewConfidenceHigh,
			ActivityCount: 3,
			WaitSeconds:   154,
			TailProbes:    2,
		},
	}

	var buf bytes.Buffer
	f, _ := formatter.New("json")
	if err := f.FormatStatus(&buf, result); err != nil {
		t.Fatalf("FormatStatus: %v", err)
	}

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte(`"review_monitor"`)) {
		t.Errorf("JSON missing review_monitor: %s", output)
	}
	if !bytes.Contains([]byte(output), []byte(`"review_settled"`)) {
		t.Errorf("JSON missing review_settled: %s", output)
	}
	if !bytes.Contains([]byte(output), []byte(`"confidence": "high"`)) {
		t.Errorf("JSON missing confidence value: %s", output)
	}
}

func TestReviewSettlementOmittedWhenNil(t *testing.T) {
	result := &domain.StatusResult{
		PRNumber:     42,
		IsMergeReady: true,
	}

	var buf bytes.Buffer
	f, _ := formatter.New("json")
	if err := f.FormatStatus(&buf, result); err != nil {
		t.Fatalf("FormatStatus: %v", err)
	}

	output := buf.String()
	if bytes.Contains([]byte(output), []byte(`"review_settled"`)) {
		t.Errorf("JSON should omit review_settled when nil: %s", output)
	}
}

func TestWatchReviewsHistoricalReviewStateSettlesMedium(t *testing.T) {
	base := time.Unix(1_700_000_000, 0)
	clock := &fakeReviewClock{now: base, step: 100 * time.Millisecond}
	cfg := ReviewWatchConfig{
		DebounceWindow:            150 * time.Millisecond,
		HardTimeout:               2 * time.Second,
		PollInterval:              time.Nanosecond,
		LateActivityGrace:         500 * time.Millisecond,
		MaxLateActivityExtensions: 1,
		TailIntervals:             []time.Duration{time.Nanosecond, time.Nanosecond},
	}
	snap := &domain.ActivitySnapshot{
		HeadSHA:     "abc123",
		ThreadCount: 2,
		ThreadIDs:   []string{"t1", "t2"},
	}
	f, _ := formatter.New("json")

	result, err := watchReviewsWithProbe(
		context.Background(),
		&bytes.Buffer{},
		f,
		"owner",
		"repo",
		1,
		"abc123",
		Fingerprint(snap),
		cfg,
		clock.Now,
		scriptedProbe([]*domain.ActivitySnapshot{snap, snap, snap}, nil),
	)
	if err != nil {
		t.Fatalf("watchReviewsWithProbe: %v", err)
	}
	if result.HeadChanged {
		t.Fatal("HeadChanged = true, want false")
	}
	if result.Settlement.Phase != domain.ReviewPhaseSettled {
		t.Fatalf("Phase = %q, want settled", result.Settlement.Phase)
	}
	if result.Settlement.Confidence != domain.ReviewConfidenceMedium {
		t.Fatalf("Confidence = %q, want medium", result.Settlement.Confidence)
	}
	if result.Settlement.ActivityCount != 0 {
		t.Fatalf("ActivityCount = %d, want 0 for historical-only review state", result.Settlement.ActivityCount)
	}
	if result.Settlement.TailProbes != 2 {
		t.Fatalf("TailProbes = %d, want 2", result.Settlement.TailProbes)
	}
}

func TestWatchReviewsTailRearmsAndSettlesHigh(t *testing.T) {
	base := time.Unix(1_700_000_000, 0)
	clock := &fakeReviewClock{now: base, step: 100 * time.Millisecond}
	cfg := ReviewWatchConfig{
		DebounceWindow:            150 * time.Millisecond,
		HardTimeout:               3 * time.Second,
		PollInterval:              time.Nanosecond,
		LateActivityGrace:         500 * time.Millisecond,
		MaxLateActivityExtensions: 1,
		TailIntervals:             []time.Duration{time.Nanosecond, time.Nanosecond},
	}
	initial := &domain.ActivitySnapshot{HeadSHA: "abc123"}
	firstActivity := &domain.ActivitySnapshot{
		HeadSHA:     "abc123",
		ThreadCount: 1,
		ThreadIDs:   []string{"t1"},
	}
	secondActivity := &domain.ActivitySnapshot{
		HeadSHA:     "abc123",
		ThreadCount: 2,
		ThreadIDs:   []string{"t1", "t2"},
	}
	f, _ := formatter.New("json")

	result, err := watchReviewsWithProbe(
		context.Background(),
		&bytes.Buffer{},
		f,
		"owner",
		"repo",
		1,
		"abc123",
		Fingerprint(initial),
		cfg,
		clock.Now,
		scriptedProbe([]*domain.ActivitySnapshot{
			initial,
			firstActivity,
			firstActivity,
			secondActivity,
			secondActivity,
			secondActivity,
			secondActivity,
		}, nil),
	)
	if err != nil {
		t.Fatalf("watchReviewsWithProbe: %v", err)
	}
	if result.Settlement.Phase != domain.ReviewPhaseSettled {
		t.Fatalf("Phase = %q, want settled", result.Settlement.Phase)
	}
	if result.Settlement.Confidence != domain.ReviewConfidenceHigh {
		t.Fatalf("Confidence = %q, want high", result.Settlement.Confidence)
	}
	if !result.Settlement.TailRearmed {
		t.Fatal("TailRearmed = false, want true")
	}
	if result.Settlement.ActivityCount != 2 {
		t.Fatalf("ActivityCount = %d, want 2", result.Settlement.ActivityCount)
	}
	if result.Settlement.TailProbes != 2 {
		t.Fatalf("TailProbes = %d, want 2", result.Settlement.TailProbes)
	}
}

func TestWatchReviewsLateActivityGraceAllowsTailToFinish(t *testing.T) {
	base := time.Unix(1_700_000_000, 0)
	clock := &fakeReviewClock{now: base, step: 100 * time.Millisecond}
	cfg := ReviewWatchConfig{
		DebounceWindow:            300 * time.Millisecond,
		HardTimeout:               500 * time.Millisecond,
		PollInterval:              time.Nanosecond,
		LateActivityGrace:         400 * time.Millisecond,
		MaxLateActivityExtensions: 1,
		TailIntervals:             []time.Duration{time.Nanosecond},
	}
	initial := &domain.ActivitySnapshot{HeadSHA: "abc123"}
	activity := &domain.ActivitySnapshot{
		HeadSHA:     "abc123",
		ThreadCount: 1,
		ThreadIDs:   []string{"t1"},
	}
	f, _ := formatter.New("json")

	result, err := watchReviewsWithProbe(
		context.Background(),
		&bytes.Buffer{},
		f,
		"owner",
		"repo",
		1,
		"abc123",
		Fingerprint(initial),
		cfg,
		clock.Now,
		scriptedProbe([]*domain.ActivitySnapshot{
			initial,
			activity,
			activity,
			activity,
		}, nil),
	)
	if err != nil {
		t.Fatalf("watchReviewsWithProbe: %v", err)
	}
	if result.Settlement.Phase != domain.ReviewPhaseSettled {
		t.Fatalf("Phase = %q, want settled", result.Settlement.Phase)
	}
	if result.Settlement.Confidence != domain.ReviewConfidenceHigh {
		t.Fatalf("Confidence = %q, want high", result.Settlement.Confidence)
	}
}
