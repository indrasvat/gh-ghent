package github

import (
	"context"
	"fmt"

	"github.com/indrasvat/gh-ghent/internal/domain"
)

const resolveThreadMutation = `
mutation ResolveThread($threadId: ID!) {
  resolveReviewThread(input: { threadId: $threadId }) {
    thread {
      id
      isResolved
      path
      line
    }
  }
}
`

const unresolveThreadMutation = `
mutation UnresolveThread($threadId: ID!) {
  unresolveReviewThread(input: { threadId: $threadId }) {
    thread {
      id
      isResolved
      path
      line
    }
  }
}
`

type resolveResponse struct {
	ResolveReviewThread struct {
		Thread resolvedThreadNode `json:"thread"`
	} `json:"resolveReviewThread"`
}

type unresolveResponse struct {
	UnresolveReviewThread struct {
		Thread resolvedThreadNode `json:"thread"`
	} `json:"unresolveReviewThread"`
}

type resolvedThreadNode struct {
	ID         string `json:"id"`
	IsResolved bool   `json:"isResolved"`
	Path       string `json:"path"`
	Line       int    `json:"line"`
}

// ResolveThread marks a review thread as resolved via GraphQL mutation.
func (c *Client) ResolveThread(ctx context.Context, threadID string) (*domain.ResolveResult, error) {
	vars := map[string]interface{}{
		"threadId": threadID,
	}

	var resp resolveResponse
	if err := c.gql.DoWithContext(ctx, resolveThreadMutation, vars, &resp); err != nil {
		return nil, fmt.Errorf("resolve thread %s: %w", threadID, err)
	}

	t := resp.ResolveReviewThread.Thread
	return &domain.ResolveResult{
		ThreadID:   t.ID,
		Path:       t.Path,
		Line:       t.Line,
		IsResolved: t.IsResolved,
		Action:     "resolved",
	}, nil
}

// UnresolveThread marks a review thread as unresolved via GraphQL mutation.
func (c *Client) UnresolveThread(ctx context.Context, threadID string) (*domain.ResolveResult, error) {
	vars := map[string]interface{}{
		"threadId": threadID,
	}

	var resp unresolveResponse
	if err := c.gql.DoWithContext(ctx, unresolveThreadMutation, vars, &resp); err != nil {
		return nil, fmt.Errorf("unresolve thread %s: %w", threadID, err)
	}

	t := resp.UnresolveReviewThread.Thread
	return &domain.ResolveResult{
		ThreadID:   t.ID,
		Path:       t.Path,
		Line:       t.Line,
		IsResolved: t.IsResolved,
		Action:     "unresolved",
	}, nil
}
