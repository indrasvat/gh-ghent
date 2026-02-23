package cli

import (
	"testing"

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
