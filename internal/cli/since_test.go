package cli

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/indrasvat/gh-ghent/internal/domain"
)

func TestParseSince(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		check   func(t *testing.T, got time.Time)
	}{
		{
			name:  "empty string returns zero",
			input: "",
			check: func(t *testing.T, got time.Time) {
				if !got.IsZero() {
					t.Errorf("expected zero time, got %v", got)
				}
			},
		},
		{
			name:  "ISO 8601 / RFC3339",
			input: "2026-02-22T10:30:00Z",
			check: func(t *testing.T, got time.Time) {
				want := time.Date(2026, 2, 22, 10, 30, 0, 0, time.UTC)
				if !got.Equal(want) {
					t.Errorf("got %v, want %v", got, want)
				}
			},
		},
		{
			name:  "ISO 8601 with timezone offset",
			input: "2026-02-22T10:30:00+05:30",
			check: func(t *testing.T, got time.Time) {
				if got.IsZero() {
					t.Error("expected non-zero time")
				}
			},
		},
		{
			name:  "relative minutes",
			input: "30m",
			check: func(t *testing.T, got time.Time) {
				diff := time.Since(got)
				if diff < 29*time.Minute || diff > 31*time.Minute {
					t.Errorf("expected ~30m ago, got %v ago", diff)
				}
			},
		},
		{
			name:  "relative hours",
			input: "2h",
			check: func(t *testing.T, got time.Time) {
				diff := time.Since(got)
				if diff < 119*time.Minute || diff > 121*time.Minute {
					t.Errorf("expected ~2h ago, got %v ago", diff)
				}
			},
		},
		{
			name:  "relative days",
			input: "7d",
			check: func(t *testing.T, got time.Time) {
				diff := time.Since(got)
				expected := 7 * 24 * time.Hour
				if diff < expected-time.Minute || diff > expected+time.Minute {
					t.Errorf("expected ~7d ago, got %v ago", diff)
				}
			},
		},
		{
			name:  "relative weeks",
			input: "1w",
			check: func(t *testing.T, got time.Time) {
				diff := time.Since(got)
				expected := 7 * 24 * time.Hour
				if diff < expected-time.Minute || diff > expected+time.Minute {
					t.Errorf("expected ~1w ago, got %v ago", diff)
				}
			},
		},
		{
			name:    "invalid format",
			input:   "yesterday",
			wantErr: true,
		},
		{
			name:    "negative duration",
			input:   "-1h",
			wantErr: true,
		},
		{
			name:    "zero duration",
			input:   "0h",
			wantErr: true,
		},
		{
			name:    "unknown unit",
			input:   "5s",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseSince(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseSince(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if tt.check != nil {
				tt.check(t, got)
			}
		})
	}
}

func TestFilterThreadsBySince(t *testing.T) {
	now := time.Now()
	old := now.Add(-48 * time.Hour)
	recent := now.Add(-1 * time.Hour)

	result := &domain.CommentsResult{
		PRNumber: 1,
		Threads: []domain.ReviewThread{
			{
				ID:   "old-thread",
				Path: "old.go",
				Line: 10,
				Comments: []domain.Comment{
					{ID: "c1", CreatedAt: old},
				},
			},
			{
				ID:   "recent-thread",
				Path: "new.go",
				Line: 20,
				Comments: []domain.Comment{
					{ID: "c2", CreatedAt: old},
					{ID: "c3", CreatedAt: recent},
				},
			},
		},
		TotalCount:      2,
		UnresolvedCount: 2,
	}

	since := now.Add(-2 * time.Hour)
	FilterThreadsBySince(result, since)

	if len(result.Threads) != 1 {
		t.Fatalf("expected 1 thread, got %d", len(result.Threads))
	}
	if result.Threads[0].ID != "recent-thread" {
		t.Errorf("expected recent-thread, got %s", result.Threads[0].ID)
	}
	if result.TotalCount != 1 {
		t.Errorf("expected TotalCount=1, got %d", result.TotalCount)
	}
	if result.UnresolvedCount != 1 {
		t.Errorf("expected UnresolvedCount=1, got %d", result.UnresolvedCount)
	}
	if result.Since == "" {
		t.Error("expected Since to be set")
	}
}

func TestFilterThreadsBySince_ZeroTime(t *testing.T) {
	result := &domain.CommentsResult{
		Threads: []domain.ReviewThread{
			{ID: "t1", Comments: []domain.Comment{{ID: "c1", CreatedAt: time.Now()}}},
		},
		TotalCount:      1,
		UnresolvedCount: 1,
	}

	FilterThreadsBySince(result, time.Time{})

	if len(result.Threads) != 1 {
		t.Errorf("zero since should be no-op, got %d threads", len(result.Threads))
	}
	if result.Since != "" {
		t.Error("Since should not be set for zero time")
	}
}

func TestFilterChecksBySince(t *testing.T) {
	now := time.Now()
	old := now.Add(-48 * time.Hour)
	recent := now.Add(-30 * time.Minute)

	result := &domain.ChecksResult{
		PRNumber:      1,
		OverallStatus: domain.StatusFail,
		Checks: []domain.CheckRun{
			{
				ID:          1,
				Name:        "old-check",
				Status:      "completed",
				Conclusion:  "success",
				CompletedAt: old,
			},
			{
				ID:          2,
				Name:        "recent-check",
				Status:      "completed",
				Conclusion:  "failure",
				CompletedAt: recent,
			},
			{
				ID:        3,
				Name:      "running-check",
				Status:    "in_progress",
				StartedAt: recent,
			},
		},
		PassCount:    1,
		FailCount:    1,
		PendingCount: 1,
	}

	since := now.Add(-1 * time.Hour)
	FilterChecksBySince(result, since)

	if len(result.Checks) != 2 {
		t.Fatalf("expected 2 checks, got %d", len(result.Checks))
	}

	want := []string{"recent-check", "running-check"}
	var got []string
	for _, ch := range result.Checks {
		got = append(got, ch.Name)
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("filtered checks mismatch (-want +got):\n%s", diff)
	}

	if result.PassCount != 0 {
		t.Errorf("expected PassCount=0, got %d", result.PassCount)
	}
	if result.FailCount != 1 {
		t.Errorf("expected FailCount=1, got %d", result.FailCount)
	}
	if result.PendingCount != 1 {
		t.Errorf("expected PendingCount=1, got %d", result.PendingCount)
	}
	if result.OverallStatus != domain.StatusFail {
		t.Errorf("expected overall_status=failure, got %s", result.OverallStatus)
	}
	if result.Since == "" {
		t.Error("expected Since to be set")
	}
}

func TestFilterChecksBySince_AllFiltered(t *testing.T) {
	old := time.Now().Add(-48 * time.Hour)

	result := &domain.ChecksResult{
		Checks: []domain.CheckRun{
			{ID: 1, Name: "c1", Status: "completed", Conclusion: "success", CompletedAt: old},
		},
		PassCount:     1,
		OverallStatus: domain.StatusPass,
	}

	FilterChecksBySince(result, time.Now().Add(-1*time.Hour))

	if len(result.Checks) != 0 {
		t.Errorf("expected 0 checks after filter, got %d", len(result.Checks))
	}
	if result.PassCount != 0 {
		t.Errorf("expected PassCount=0, got %d", result.PassCount)
	}
	// With no checks, AggregateStatus returns pass (empty list).
	if result.OverallStatus != domain.StatusPass {
		t.Errorf("expected pass for empty checks, got %s", result.OverallStatus)
	}
}
