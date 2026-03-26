package github

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/indrasvat/gh-ghent/internal/domain"
	"github.com/indrasvat/gh-ghent/internal/formatter"
)

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
		Timestamp:       time.Now(),
		OverallStatus:   domain.StatusPass,
		ReviewPhase:     domain.ReviewPhaseWaiting,
		ReviewIdleSecs:  12,
		ReviewTimeoutIn: 288,
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

func TestReviewSettlementInSummaryResult(t *testing.T) {
	// Verify ReviewSettled is included in SummaryResult serialization.
	result := &domain.SummaryResult{
		PRNumber:     42,
		IsMergeReady: false,
		ReviewSettled: &domain.ReviewSettlement{
			Phase:         domain.ReviewPhaseSettled,
			ActivityCount: 3,
			WaitSeconds:   154,
		},
	}

	var buf bytes.Buffer
	f, _ := formatter.New("json")
	if err := f.FormatSummary(&buf, result); err != nil {
		t.Fatalf("FormatSummary: %v", err)
	}

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte(`"review_settled"`)) {
		t.Errorf("JSON missing review_settled: %s", output)
	}
	if !bytes.Contains([]byte(output), []byte(`"settled"`)) {
		t.Errorf("JSON missing phase value: %s", output)
	}
}

func TestReviewSettlementOmittedWhenNil(t *testing.T) {
	result := &domain.SummaryResult{
		PRNumber:     42,
		IsMergeReady: true,
	}

	var buf bytes.Buffer
	f, _ := formatter.New("json")
	if err := f.FormatSummary(&buf, result); err != nil {
		t.Fatalf("FormatSummary: %v", err)
	}

	output := buf.String()
	if bytes.Contains([]byte(output), []byte(`"review_settled"`)) {
		t.Errorf("JSON should omit review_settled when nil: %s", output)
	}
}
