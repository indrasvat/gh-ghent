package github

import (
	"context"
	"fmt"
	"time"

	"github.com/indrasvat/gh-ghent/internal/domain"
)

// reviewsQuery is the GraphQL query for fetching PR reviews.
const reviewsQuery = `
query($owner: String!, $repo: String!, $pr: Int!) {
  repository(owner: $owner, name: $repo) {
    pullRequest(number: $pr) {
      reviews(first: 100) {
        nodes {
          id
          author { login }
          state
          body
          submittedAt
        }
      }
    }
  }
}
`

type reviewsResponse struct {
	Repository struct {
		PullRequest struct {
			Reviews struct {
				Nodes []reviewNode `json:"nodes"`
			} `json:"reviews"`
		} `json:"pullRequest"`
	} `json:"repository"`
}

type reviewNode struct {
	ID     string `json:"id"`
	Author struct {
		Login string `json:"login"`
	} `json:"author"`
	State       string `json:"state"`
	Body        string `json:"body"`
	SubmittedAt string `json:"submittedAt"`
}

// FetchReviews retrieves all reviews for a PR via GraphQL.
func (c *Client) FetchReviews(ctx context.Context, owner, repo string, pr int) ([]domain.Review, error) {
	vars := map[string]interface{}{
		"owner": owner,
		"repo":  repo,
		"pr":    pr,
	}

	var resp reviewsResponse
	if err := c.gql.DoWithContext(ctx, reviewsQuery, vars, &resp); err != nil {
		return nil, fmt.Errorf("graphql reviews: %w", err)
	}

	return mapReviewsToDomain(resp.Repository.PullRequest.Reviews.Nodes)
}

// mapReviewsToDomain converts GraphQL review nodes to domain Review types.
func mapReviewsToDomain(nodes []reviewNode) ([]domain.Review, error) {
	reviews := make([]domain.Review, 0, len(nodes))
	for _, n := range nodes {
		var submittedAt time.Time
		if n.SubmittedAt != "" {
			t, err := time.Parse(time.RFC3339, n.SubmittedAt)
			if err != nil {
				return nil, fmt.Errorf("parse review time %q: %w", n.SubmittedAt, err)
			}
			submittedAt = t
		}

		reviews = append(reviews, domain.Review{
			ID:          n.ID,
			Author:      n.Author.Login,
			State:       domain.ReviewState(n.State),
			Body:        n.Body,
			SubmittedAt: submittedAt,
		})
	}
	return reviews, nil
}
