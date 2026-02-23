package domain

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestAggregateStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		statuses []OverallStatus
		want     OverallStatus
	}{
		{
			name:     "all pass",
			statuses: []OverallStatus{StatusPass, StatusPass, StatusPass},
			want:     StatusPass,
		},
		{
			name:     "one fail among passes",
			statuses: []OverallStatus{StatusPass, StatusFail, StatusPass},
			want:     StatusFail,
		},
		{
			name:     "one pending among passes",
			statuses: []OverallStatus{StatusPass, StatusPending, StatusPass},
			want:     StatusPending,
		},
		{
			name:     "fail and pending returns fail",
			statuses: []OverallStatus{StatusPending, StatusFail, StatusPass},
			want:     StatusFail,
		},
		{
			name:     "empty slice returns pass",
			statuses: []OverallStatus{},
			want:     StatusPass,
		},
		{
			name:     "nil slice returns pass",
			statuses: nil,
			want:     StatusPass,
		},
		{
			name:     "single pass",
			statuses: []OverallStatus{StatusPass},
			want:     StatusPass,
		},
		{
			name:     "single fail",
			statuses: []OverallStatus{StatusFail},
			want:     StatusFail,
		},
		{
			name:     "single pending",
			statuses: []OverallStatus{StatusPending},
			want:     StatusPending,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := AggregateStatus(tt.statuses)
			if got != tt.want {
				t.Errorf("AggregateStatus(%v) = %q, want %q", tt.statuses, got, tt.want)
			}
		})
	}
}

func TestReviewThread_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 2, 22, 12, 0, 0, 0, time.UTC)
	thread := ReviewThread{
		ID:                 "RT_abc123",
		Path:               "internal/cli/root.go",
		Line:               42,
		StartLine:          40,
		DiffSide:           "RIGHT",
		IsResolved:         false,
		IsOutdated:         true,
		ViewerCanResolve:   true,
		ViewerCanUnresolve: false,
		ViewerCanReply:     true,
		Comments: []Comment{
			{
				ID:         "C_def456",
				DatabaseID: 123456789,
				Author:     "reviewer",
				Body:       "Please fix this.",
				CreatedAt:  now,
				URL:        "https://github.com/org/repo/pull/1#discussion_r123",
				DiffHunk:   "@@ -40,3 +40,3 @@",
				Path:       "internal/cli/root.go",
			},
		},
	}

	data, err := json.Marshal(thread)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	var got ReviewThread
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	if diff := cmp.Diff(thread, got); diff != "" {
		t.Errorf("round-trip mismatch (-want +got):\n%s", diff)
	}
}

func TestCommentsResult_ZeroValue(t *testing.T) {
	t.Parallel()

	var result CommentsResult

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal zero CommentsResult: %v", err)
	}

	var got CommentsResult
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal zero CommentsResult: %v", err)
	}

	if diff := cmp.Diff(result, got); diff != "" {
		t.Errorf("zero-value round-trip mismatch (-want +got):\n%s", diff)
	}
}

func TestComment_DatabaseID_JSONKey(t *testing.T) {
	t.Parallel()

	comment := Comment{
		ID:         "C_abc",
		DatabaseID: 987654321,
		Author:     "user",
		Body:       "test",
		CreatedAt:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		URL:        "https://example.com",
	}

	data, err := json.Marshal(comment)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("json.Unmarshal to map: %v", err)
	}

	if _, ok := raw["database_id"]; !ok {
		t.Errorf("expected JSON key \"database_id\", got keys: %v", keys(raw))
	}

	var dbID int64
	if err := json.Unmarshal(raw["database_id"], &dbID); err != nil {
		t.Fatalf("json.Unmarshal database_id: %v", err)
	}
	if dbID != 987654321 {
		t.Errorf("database_id = %d, want 987654321", dbID)
	}
}

func keys(m map[string]json.RawMessage) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	return ks
}
