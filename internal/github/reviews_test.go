package github

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/indrasvat/gh-ghent/internal/domain"
)

func loadReviewsFixture(t *testing.T, path string) reviewsResponse {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", path, err)
	}
	var envelope struct {
		Data reviewsResponse `json:"data"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		t.Fatalf("unmarshal fixture %s: %v", path, err)
	}
	return envelope.Data
}

func TestMapReviewsToDomain(t *testing.T) {
	resp := loadReviewsFixture(t, "../../testdata/graphql/pr_reviews.json")
	nodes := resp.Repository.PullRequest.Reviews.Nodes

	reviews, err := mapReviewsToDomain(nodes)
	if err != nil {
		t.Fatalf("mapReviewsToDomain: %v", err)
	}

	if len(reviews) != 4 {
		t.Fatalf("len(reviews) = %d, want 4", len(reviews))
	}

	// Verify first review (APPROVED)
	got := reviews[0]
	want := domain.Review{
		ID:          "PRR_review1",
		Author:      "reviewer1",
		State:       domain.ReviewApproved,
		Body:        "Looks good!",
		SubmittedAt: mustParseTime(t, "2026-02-20T12:00:00Z"),
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("review[0] mismatch (-want +got):\n%s", diff)
	}

	// Verify second review (CHANGES_REQUESTED)
	if reviews[1].State != domain.ReviewChangesRequested {
		t.Errorf("review[1].State = %q, want %q", reviews[1].State, domain.ReviewChangesRequested)
	}

	// Verify third review (COMMENTED)
	if reviews[2].State != domain.ReviewCommented {
		t.Errorf("review[2].State = %q, want %q", reviews[2].State, domain.ReviewCommented)
	}

	// Verify fourth review (DISMISSED)
	if reviews[3].State != domain.ReviewDismissed {
		t.Errorf("review[3].State = %q, want %q", reviews[3].State, domain.ReviewDismissed)
	}
}

func TestMapReviewsEmpty(t *testing.T) {
	reviews, err := mapReviewsToDomain(nil)
	if err != nil {
		t.Fatalf("mapReviewsToDomain(nil): %v", err)
	}
	if len(reviews) != 0 {
		t.Errorf("len(reviews) = %d, want 0", len(reviews))
	}
}

func TestMapReviewsAllStates(t *testing.T) {
	mkAuthor := func(login string) struct {
		Login string `json:"login"`
	} {
		return struct {
			Login string `json:"login"`
		}{Login: login}
	}

	nodes := []reviewNode{
		{ID: "r1", Author: mkAuthor("a"), State: "APPROVED", SubmittedAt: "2026-02-20T10:00:00Z"},
		{ID: "r2", Author: mkAuthor("b"), State: "CHANGES_REQUESTED", SubmittedAt: "2026-02-20T11:00:00Z"},
		{ID: "r3", Author: mkAuthor("c"), State: "COMMENTED", SubmittedAt: "2026-02-20T12:00:00Z"},
		{ID: "r4", Author: mkAuthor("d"), State: "PENDING", SubmittedAt: ""},
		{ID: "r5", Author: mkAuthor("e"), State: "DISMISSED", SubmittedAt: "2026-02-20T14:00:00Z"},
	}

	reviews, err := mapReviewsToDomain(nodes)
	if err != nil {
		t.Fatalf("mapReviewsToDomain: %v", err)
	}

	wantStates := []domain.ReviewState{
		domain.ReviewApproved,
		domain.ReviewChangesRequested,
		domain.ReviewCommented,
		domain.ReviewPending,
		domain.ReviewDismissed,
	}

	for i, want := range wantStates {
		if reviews[i].State != want {
			t.Errorf("reviews[%d].State = %q, want %q", i, reviews[i].State, want)
		}
	}

	// PENDING review has zero-value SubmittedAt
	if !reviews[3].SubmittedAt.IsZero() {
		t.Errorf("PENDING review should have zero SubmittedAt, got %v", reviews[3].SubmittedAt)
	}
}
