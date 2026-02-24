package github

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/indrasvat/gh-ghent/internal/domain"
)

func TestBuildWatchStatus(t *testing.T) {
	now := time.Date(2026, 2, 22, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name   string
		result *domain.ChecksResult
		seen   map[int64]string
		want   *domain.WatchStatus
	}{
		{
			name: "all pass",
			result: &domain.ChecksResult{
				OverallStatus: domain.StatusPass,
				Checks: []domain.CheckRun{
					{ID: 1, Name: "lint", Status: "completed", Conclusion: "success"},
					{ID: 2, Name: "test", Status: "completed", Conclusion: "success"},
				},
				PassCount:    2,
				FailCount:    0,
				PendingCount: 0,
			},
			seen: map[int64]string{},
			want: &domain.WatchStatus{
				Timestamp:     now,
				OverallStatus: domain.StatusPass,
				Completed:     2,
				Total:         2,
				PassCount:     2,
				FailCount:     0,
				PendingCount:  0,
				Events: []domain.WatchEvent{
					{Name: "lint", Status: "completed", Conclusion: "success", Timestamp: now},
					{Name: "test", Status: "completed", Conclusion: "success", Timestamp: now},
				},
			},
		},
		{
			name: "one pending",
			result: &domain.ChecksResult{
				OverallStatus: domain.StatusPending,
				Checks: []domain.CheckRun{
					{ID: 1, Name: "lint", Status: "completed", Conclusion: "success"},
					{ID: 2, Name: "test", Status: "in_progress"},
				},
				PassCount:    1,
				FailCount:    0,
				PendingCount: 1,
			},
			seen: map[int64]string{},
			want: &domain.WatchStatus{
				Timestamp:     now,
				OverallStatus: domain.StatusPending,
				Completed:     1,
				Total:         2,
				PassCount:     1,
				FailCount:     0,
				PendingCount:  1,
				Events: []domain.WatchEvent{
					{Name: "lint", Status: "completed", Conclusion: "success", Timestamp: now},
				},
			},
		},
		{
			name: "fail fast",
			result: &domain.ChecksResult{
				OverallStatus: domain.StatusFail,
				Checks: []domain.CheckRun{
					{ID: 1, Name: "lint", Status: "completed", Conclusion: "failure"},
					{ID: 2, Name: "test", Status: "in_progress"},
				},
				PassCount:    0,
				FailCount:    1,
				PendingCount: 1,
			},
			seen: map[int64]string{},
			want: &domain.WatchStatus{
				Timestamp:     now,
				OverallStatus: domain.StatusFail,
				Completed:     1,
				Total:         2,
				PassCount:     0,
				FailCount:     1,
				PendingCount:  1,
				Events: []domain.WatchEvent{
					{Name: "lint", Status: "completed", Conclusion: "failure", Timestamp: now},
				},
			},
		},
		{
			name: "already seen checks not reported",
			result: &domain.ChecksResult{
				OverallStatus: domain.StatusPass,
				Checks: []domain.CheckRun{
					{ID: 1, Name: "lint", Status: "completed", Conclusion: "success"},
					{ID: 2, Name: "test", Status: "completed", Conclusion: "success"},
				},
				PassCount:    2,
				FailCount:    0,
				PendingCount: 0,
			},
			seen: map[int64]string{1: "success"}, // lint already seen
			want: &domain.WatchStatus{
				Timestamp:     now,
				OverallStatus: domain.StatusPass,
				Completed:     2,
				Total:         2,
				PassCount:     2,
				FailCount:     0,
				PendingCount:  0,
				Events: []domain.WatchEvent{
					{Name: "test", Status: "completed", Conclusion: "success", Timestamp: now},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildWatchStatus(now, tt.result, tt.seen)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("buildWatchStatus() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestBuildWatchStatus_NoEvents(t *testing.T) {
	now := time.Date(2026, 2, 22, 12, 0, 0, 0, time.UTC)
	seen := map[int64]string{1: "success", 2: "success"}
	result := &domain.ChecksResult{
		OverallStatus: domain.StatusPass,
		Checks: []domain.CheckRun{
			{ID: 1, Name: "lint", Status: "completed", Conclusion: "success"},
			{ID: 2, Name: "test", Status: "completed", Conclusion: "success"},
		},
		PassCount: 2,
	}

	got := buildWatchStatus(now, result, seen)
	if len(got.Events) != 0 {
		t.Errorf("expected no events when all seen, got %d", len(got.Events))
	}
}

func TestWatchStatusJSONOutput(t *testing.T) {
	now := time.Date(2026, 2, 22, 12, 0, 0, 0, time.UTC)
	status := &domain.WatchStatus{
		Timestamp:     now,
		OverallStatus: domain.StatusPass,
		Completed:     2,
		Total:         2,
		PassCount:     2,
		Final:         true,
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	if err := enc.Encode(status); err != nil {
		t.Fatal(err)
	}

	// Must be a single line (NDJSON).
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 1 {
		t.Errorf("expected 1 line (NDJSON), got %d", len(lines))
	}

	// Must be valid JSON.
	var decoded domain.WatchStatus
	if err := json.Unmarshal([]byte(lines[0]), &decoded); err != nil {
		t.Errorf("invalid JSON: %v", err)
	}
	if decoded.OverallStatus != domain.StatusPass {
		t.Errorf("got status %q, want %q", decoded.OverallStatus, domain.StatusPass)
	}
}

func TestContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	result := &domain.ChecksResult{
		OverallStatus: domain.StatusPending,
		Checks: []domain.CheckRun{
			{ID: 1, Name: "test", Status: "in_progress"},
		},
		PendingCount: 1,
	}

	// buildWatchStatus should still work on cancelled context (it's pure).
	seen := map[int64]string{}
	got := buildWatchStatus(time.Now(), result, seen)
	if got.OverallStatus != domain.StatusPending {
		t.Errorf("got %q, want %q", got.OverallStatus, domain.StatusPending)
	}

	// Verify context.Err reports cancellation.
	if ctx.Err() == nil {
		t.Error("expected context to be cancelled")
	}
}
