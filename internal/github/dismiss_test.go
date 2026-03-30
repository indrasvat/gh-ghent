package github

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/indrasvat/gh-ghent/internal/domain"
)

func TestDismissReviewResponseParsing(t *testing.T) {
	data, err := os.ReadFile("../../testdata/rest/dismiss_review.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	var resp dismissReviewResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}

	want := dismissReviewResponse{State: "DISMISSED"}
	if diff := cmp.Diff(want, resp); diff != "" {
		t.Errorf("dismissReviewResponse mismatch (-want +got):\n%s", diff)
	}
}

func TestDismissResultMapping(t *testing.T) {
	review := domain.Review{
		ID:         "PRR_review2",
		DatabaseID: 102,
		Author:     "reviewer2",
		IsBot:      true,
		State:      domain.ReviewChangesRequested,
		CommitID:   "abc123",
		IsStale:    true,
	}

	got := &domain.DismissResult{
		ReviewID:   review.ID,
		DatabaseID: review.DatabaseID,
		Author:     review.Author,
		IsBot:      review.IsBot,
		State:      domain.ReviewDismissed,
		CommitID:   review.CommitID,
		IsStale:    review.IsStale,
		Dismissed:  true,
		Action:     "dismissed",
		Message:    "superseded by current HEAD",
	}

	want := &domain.DismissResult{
		ReviewID:   "PRR_review2",
		DatabaseID: 102,
		Author:     "reviewer2",
		IsBot:      true,
		State:      domain.ReviewDismissed,
		CommitID:   "abc123",
		IsStale:    true,
		Dismissed:  true,
		Action:     "dismissed",
		Message:    "superseded by current HEAD",
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("dismiss result mismatch (-want +got):\n%s", diff)
	}
}
