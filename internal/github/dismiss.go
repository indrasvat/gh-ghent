package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/indrasvat/gh-ghent/internal/domain"
)

type dismissReviewResponse struct {
	State string `json:"state"`
}

// DismissReview dismisses a stale blocking review via GitHub REST.
func (c *Client) DismissReview(
	ctx context.Context,
	owner, repo string,
	pr int,
	review domain.Review,
	message string,
) (*domain.DismissResult, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	if review.DatabaseID == 0 {
		return nil, fmt.Errorf("dismiss review: review %s is missing numeric database ID", review.ID)
	}

	reqBody := struct {
		Message string `json:"message"`
	}{Message: message}

	encoded, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("dismiss review: encode body: %w", err)
	}

	endpoint := fmt.Sprintf("repos/%s/%s/pulls/%d/reviews/%d/dismissals", owner, repo, pr, review.DatabaseID)

	var resp dismissReviewResponse
	if err := doWithRetry(func() error {
		return c.rest.DoWithContext(ctx, http.MethodPut, endpoint, bytes.NewReader(encoded), &resp)
	}); err != nil {
		return nil, classifyError(err)
	}

	state := domain.ReviewDismissed
	if resp.State != "" {
		state = domain.ReviewState(resp.State)
	}

	return &domain.DismissResult{
		ReviewID:    review.ID,
		DatabaseID:  review.DatabaseID,
		Author:      review.Author,
		IsBot:       review.IsBot,
		State:       state,
		CommitID:    review.CommitID,
		IsStale:     review.IsStale,
		Dismissed:   true,
		Action:      "dismissed",
		Message:     message,
		SubmittedAt: review.SubmittedAt,
	}, nil
}
