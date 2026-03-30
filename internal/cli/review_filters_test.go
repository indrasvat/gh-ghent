package cli

import (
	"testing"

	"github.com/indrasvat/gh-ghent/internal/domain"
)

func TestStaleBlockingReviews(t *testing.T) {
	reviews := []domain.Review{
		{ID: "1", State: domain.ReviewApproved, IsStale: true},
		{ID: "2", State: domain.ReviewChangesRequested, IsStale: false},
		{ID: "3", State: domain.ReviewChangesRequested, IsStale: true},
	}

	got := staleBlockingReviews(reviews)
	if len(got) != 1 {
		t.Fatalf("len(staleBlockingReviews) = %d, want 1", len(got))
	}
	if got[0].ID != "3" {
		t.Errorf("stale blocking review = %q, want 3", got[0].ID)
	}
}

func TestSelectDismissReviews(t *testing.T) {
	reviews := []domain.Review{
		{ID: "PRR_1", DatabaseID: 101, Author: "alice", State: domain.ReviewChangesRequested, IsStale: true},
		{ID: "PRR_2", DatabaseID: 102, Author: "bot", IsBot: true, State: domain.ReviewChangesRequested, IsStale: true},
		{ID: "PRR_3", DatabaseID: 103, Author: "carol", State: domain.ReviewChangesRequested, IsStale: false},
	}

	tests := []struct {
		name     string
		selector string
		author   string
		botsOnly bool
		wantIDs  []string
		wantErr  bool
	}{
		{name: "all stale blockers", wantIDs: []string{"PRR_1", "PRR_2"}},
		{name: "filter by author", author: "alice", wantIDs: []string{"PRR_1"}},
		{name: "filter bots only", botsOnly: true, wantIDs: []string{"PRR_2"}},
		{name: "selector by node id", selector: "PRR_2", wantIDs: []string{"PRR_2"}},
		{name: "selector by numeric id", selector: "101", wantIDs: []string{"PRR_1"}},
		{name: "selector on non-stale review errors", selector: "PRR_3", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := selectDismissReviews(reviews, tt.selector, tt.author, tt.botsOnly)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("selectDismissReviews: %v", err)
			}
			if len(got) != len(tt.wantIDs) {
				t.Fatalf("len(got) = %d, want %d", len(got), len(tt.wantIDs))
			}
			for i := range got {
				if got[i].ID != tt.wantIDs[i] {
					t.Errorf("got[%d].ID = %q, want %q", i, got[i].ID, tt.wantIDs[i])
				}
			}
		})
	}
}
