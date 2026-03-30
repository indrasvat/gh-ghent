package cli

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/indrasvat/gh-ghent/internal/domain"
)

type stubDismissClient struct {
	reviews        []domain.Review
	fetchErr       error
	dismissErrs    map[string]error
	dismissCalls   []string
	dismissMessage string
}

func (s *stubDismissClient) FetchReviews(context.Context, string, string, int) ([]domain.Review, error) {
	if s.fetchErr != nil {
		return nil, s.fetchErr
	}
	return s.reviews, nil
}

func (s *stubDismissClient) DismissReview(
	_ context.Context,
	_ string,
	_ string,
	_ int,
	review domain.Review,
	message string,
) (*domain.DismissResult, error) {
	s.dismissCalls = append(s.dismissCalls, review.ID)
	s.dismissMessage = message
	if err := s.dismissErrs[review.ID]; err != nil {
		return nil, err
	}
	return &domain.DismissResult{
		ReviewID:    review.ID,
		DatabaseID:  review.DatabaseID,
		Author:      review.Author,
		IsBot:       review.IsBot,
		State:       domain.ReviewDismissed,
		CommitID:    review.CommitID,
		IsStale:     review.IsStale,
		Dismissed:   true,
		Action:      "dismissed",
		Message:     message,
		SubmittedAt: review.SubmittedAt,
	}, nil
}

func staleDismissFixture() []domain.Review {
	now := time.Date(2026, 3, 30, 12, 0, 0, 0, time.UTC)
	return []domain.Review{
		{
			ID:          "PRR_1",
			DatabaseID:  101,
			Author:      "alice",
			State:       domain.ReviewChangesRequested,
			CommitID:    "deadbeefcafebabe",
			IsStale:     true,
			SubmittedAt: now,
		},
		{
			ID:          "PRR_2",
			DatabaseID:  102,
			Author:      "coderabbitai",
			IsBot:       true,
			State:       domain.ReviewChangesRequested,
			CommitID:    "feedfacecafebeef",
			IsStale:     true,
			SubmittedAt: now.Add(5 * time.Minute),
		},
		{
			ID:          "PRR_3",
			DatabaseID:  103,
			Author:      "carol",
			State:       domain.ReviewChangesRequested,
			CommitID:    "0123456789abcdef",
			IsStale:     false,
			SubmittedAt: now.Add(10 * time.Minute),
		},
	}
}

func TestBuildDismissResultsDryRun(t *testing.T) {
	client := &stubDismissClient{reviews: staleDismissFixture()}

	results, err := buildDismissResults(
		context.Background(),
		client,
		"owner",
		"repo",
		42,
		"",
		"",
		true,
		"",
		true,
	)
	if err != nil {
		t.Fatalf("buildDismissResults: %v", err)
	}

	if results.SuccessCount != 1 || results.FailureCount != 0 {
		t.Fatalf("counts = (%d, %d), want (1, 0)", results.SuccessCount, results.FailureCount)
	}
	if len(client.dismissCalls) != 0 {
		t.Fatalf("dismiss should not be called during dry-run, got %d calls", len(client.dismissCalls))
	}
	if got := results.Results[0]; got.Action != "would_dismiss" || got.ReviewID != "PRR_2" || !got.IsBot {
		t.Errorf("dry-run result = %+v, want bot review PRR_2 marked would_dismiss", got)
	}
}

func TestBuildDismissResultsPartialFailure(t *testing.T) {
	client := &stubDismissClient{
		reviews:     staleDismissFixture(),
		dismissErrs: map[string]error{"PRR_2": errors.New("review already dismissed")},
	}

	results, err := buildDismissResults(
		context.Background(),
		client,
		"owner",
		"repo",
		42,
		"",
		"",
		false,
		"superseded by current HEAD",
		false,
	)
	if err != nil {
		t.Fatalf("buildDismissResults: %v", err)
	}

	if results.SuccessCount != 1 || results.FailureCount != 1 {
		t.Fatalf("counts = (%d, %d), want (1, 1)", results.SuccessCount, results.FailureCount)
	}
	if len(results.Errors) != 1 || results.Errors[0].ReviewID != "PRR_2" {
		t.Fatalf("errors = %+v, want PRR_2 failure", results.Errors)
	}
	if len(client.dismissCalls) != 2 {
		t.Fatalf("dismiss calls = %d, want 2", len(client.dismissCalls))
	}
	if client.dismissMessage != "superseded by current HEAD" {
		t.Errorf("dismiss message = %q, want propagated message", client.dismissMessage)
	}
}

func TestBuildDismissResultsNoMatches(t *testing.T) {
	client := &stubDismissClient{reviews: staleDismissFixture()}

	_, err := buildDismissResults(
		context.Background(),
		client,
		"owner",
		"repo",
		42,
		"",
		"nobody",
		false,
		"superseded by current HEAD",
		false,
	)
	if err == nil {
		t.Fatal("expected no-match error")
	}
	if got, want := err.Error(), "no stale CHANGES_REQUESTED reviews matched"; got != want {
		t.Errorf("error = %q, want %q", got, want)
	}
}

func TestDismissExitCode(t *testing.T) {
	tests := []struct {
		name    string
		results *domain.DismissResults
		want    int
	}{
		{name: "nil", results: nil, want: 0},
		{name: "all success", results: &domain.DismissResults{SuccessCount: 2}, want: 0},
		{name: "partial failure", results: &domain.DismissResults{SuccessCount: 1, FailureCount: 1}, want: 1},
		{name: "all failure", results: &domain.DismissResults{FailureCount: 2}, want: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := dismissExitCode(tt.results); got != tt.want {
				t.Errorf("dismissExitCode() = %d, want %d", got, tt.want)
			}
		})
	}
}
