package cli

import (
	"testing"
	"time"

	"github.com/indrasvat/gh-ghent/internal/domain"
)

func TestIsMergeReady(t *testing.T) {
	threadsClean := &domain.CommentsResult{UnresolvedCount: 0}
	threadsDirty := &domain.CommentsResult{UnresolvedCount: 2}

	checksPass := &domain.ChecksResult{OverallStatus: domain.StatusPass}
	checksFail := &domain.ChecksResult{OverallStatus: domain.StatusFail}
	checksPending := &domain.ChecksResult{OverallStatus: domain.StatusPending}

	reviewsApproved := []domain.Review{
		{Author: "alice", State: domain.ReviewApproved},
	}
	reviewsChangesRequested := []domain.Review{
		{Author: "alice", State: domain.ReviewChangesRequested},
	}
	reviewsCommentedOnly := []domain.Review{
		{Author: "alice", State: domain.ReviewCommented},
	}
	reviewsMixed := []domain.Review{
		{Author: "alice", State: domain.ReviewApproved},
		{Author: "bob", State: domain.ReviewChangesRequested},
	}
	reviewsApprovedAndCommented := []domain.Review{
		{Author: "alice", State: domain.ReviewApproved},
		{Author: "bob", State: domain.ReviewCommented},
	}
	reviewsEmpty := []domain.Review{}

	tests := []struct {
		name    string
		threads *domain.CommentsResult
		checks  *domain.ChecksResult
		reviews []domain.Review
		want    bool
	}{
		{
			name:    "all clear",
			threads: threadsClean,
			checks:  checksPass,
			reviews: reviewsApproved,
			want:    true,
		},
		{
			name:    "unresolved threads",
			threads: threadsDirty,
			checks:  checksPass,
			reviews: reviewsApproved,
			want:    false,
		},
		{
			name:    "failing checks",
			threads: threadsClean,
			checks:  checksFail,
			reviews: reviewsApproved,
			want:    false,
		},
		{
			name:    "pending checks",
			threads: threadsClean,
			checks:  checksPending,
			reviews: reviewsApproved,
			want:    false,
		},
		{
			name:    "changes requested",
			threads: threadsClean,
			checks:  checksPass,
			reviews: reviewsChangesRequested,
			want:    false,
		},
		{
			name:    "no approvals only comments",
			threads: threadsClean,
			checks:  checksPass,
			reviews: reviewsCommentedOnly,
			want:    false,
		},
		{
			name:    "empty reviews list",
			threads: threadsClean,
			checks:  checksPass,
			reviews: reviewsEmpty,
			want:    false,
		},
		{
			name:    "mixed approval and changes requested",
			threads: threadsClean,
			checks:  checksPass,
			reviews: reviewsMixed,
			want:    false,
		},
		{
			name:    "approval plus comment",
			threads: threadsClean,
			checks:  checksPass,
			reviews: reviewsApprovedAndCommented,
			want:    true,
		},
		{
			name:    "nil reviews (fetch failed) skips approval check",
			threads: threadsClean,
			checks:  checksPass,
			reviews: nil,
			want:    true,
		},
		{
			name:    "nil threads treated as clean",
			threads: nil,
			checks:  checksPass,
			reviews: reviewsApproved,
			want:    true,
		},
		{
			name:    "nil checks treated as passing",
			threads: threadsClean,
			checks:  nil,
			reviews: reviewsApproved,
			want:    true,
		},
		{
			name:    "everything nil",
			threads: nil,
			checks:  nil,
			reviews: nil,
			want:    true,
		},
		{
			name:    "all failing",
			threads: threadsDirty,
			checks:  checksFail,
			reviews: reviewsChangesRequested,
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsMergeReady(tt.threads, tt.checks, tt.reviews)
			if got != tt.want {
				t.Errorf("IsMergeReady() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatRelativeTime(t *testing.T) {
	tests := []struct {
		name string
		d    time.Duration
		want string
	}{
		{name: "seconds", d: 30 * time.Second, want: "<1m"},
		{name: "minutes", d: 45 * time.Minute, want: "45m"},
		{name: "hours", d: 5 * time.Hour, want: "5h"},
		{name: "days", d: 3 * 24 * time.Hour, want: "3d"},
		{name: "weeks", d: 14 * 24 * time.Hour, want: "2w"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatRelativeTime(tt.d)
			if got != tt.want {
				t.Errorf("formatRelativeTime(%v) = %q, want %q", tt.d, got, tt.want)
			}
		})
	}
}

func TestComputeReviewCycles(t *testing.T) {
	tests := []struct {
		name    string
		reviews []domain.Review
		want    int
	}{
		{
			name:    "no reviews",
			reviews: nil,
			want:    0,
		},
		{
			name: "single day",
			reviews: []domain.Review{
				{SubmittedAt: time.Date(2026, 2, 20, 10, 0, 0, 0, time.UTC)},
				{SubmittedAt: time.Date(2026, 2, 20, 14, 0, 0, 0, time.UTC)},
			},
			want: 1,
		},
		{
			name: "two days",
			reviews: []domain.Review{
				{SubmittedAt: time.Date(2026, 2, 20, 10, 0, 0, 0, time.UTC)},
				{SubmittedAt: time.Date(2026, 2, 21, 9, 0, 0, 0, time.UTC)},
			},
			want: 2,
		},
		{
			name: "three distinct days",
			reviews: []domain.Review{
				{SubmittedAt: time.Date(2026, 2, 18, 10, 0, 0, 0, time.UTC)},
				{SubmittedAt: time.Date(2026, 2, 20, 14, 0, 0, 0, time.UTC)},
				{SubmittedAt: time.Date(2026, 2, 22, 9, 0, 0, 0, time.UTC)},
			},
			want: 3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeReviewCycles(tt.reviews)
			if got != tt.want {
				t.Errorf("computeReviewCycles() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestComputePRAge(t *testing.T) {
	now := time.Date(2026, 2, 23, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		threads *domain.CommentsResult
		reviews []domain.Review
		want    string
	}{
		{
			name:    "no data",
			threads: &domain.CommentsResult{},
			reviews: nil,
			want:    "",
		},
		{
			name: "from thread comments",
			threads: &domain.CommentsResult{
				Threads: []domain.ReviewThread{
					{Comments: []domain.Comment{
						{CreatedAt: time.Date(2026, 2, 20, 12, 0, 0, 0, time.UTC)},
					}},
				},
			},
			reviews: nil,
			want:    "3d",
		},
		{
			name:    "from reviews",
			threads: &domain.CommentsResult{},
			reviews: []domain.Review{
				{SubmittedAt: time.Date(2026, 2, 22, 12, 0, 0, 0, time.UTC)},
			},
			want: "1d",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computePRAge(tt.threads, tt.reviews, now)
			if got != tt.want {
				t.Errorf("computePRAge() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestComputeLastUpdate(t *testing.T) {
	now := time.Date(2026, 2, 23, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		threads *domain.CommentsResult
		reviews []domain.Review
		want    string
	}{
		{
			name:    "no data",
			threads: &domain.CommentsResult{},
			reviews: nil,
			want:    "",
		},
		{
			name: "recent comment",
			threads: &domain.CommentsResult{
				Threads: []domain.ReviewThread{
					{Comments: []domain.Comment{
						{CreatedAt: time.Date(2026, 2, 23, 7, 0, 0, 0, time.UTC)},
					}},
				},
			},
			reviews: nil,
			want:    "5h",
		},
		{
			name:    "recent review",
			threads: &domain.CommentsResult{},
			reviews: []domain.Review{
				{SubmittedAt: time.Date(2026, 2, 23, 11, 30, 0, 0, time.UTC)},
			},
			want: "30m",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeLastUpdate(tt.threads, tt.reviews, now)
			if got != tt.want {
				t.Errorf("computeLastUpdate() = %q, want %q", got, tt.want)
			}
		})
	}
}
