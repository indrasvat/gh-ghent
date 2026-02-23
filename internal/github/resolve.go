package github

import (
	"context"
	"log/slog"
	"time"

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
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	start := time.Now()
	slog.Debug("resolving thread", "threadID", threadID)

	vars := map[string]interface{}{
		"threadId": threadID,
	}

	var resp resolveResponse
	if err := doWithRetry(func() error {
		return c.gql.DoWithContext(ctx, resolveThreadMutation, vars, &resp)
	}); err != nil {
		return nil, classifyWithContext(err, "thread", threadID)
	}

	t := resp.ResolveReviewThread.Thread
	slog.Debug("resolved thread", "threadID", t.ID, "path", t.Path, "line", t.Line, "duration", time.Since(start))

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
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	start := time.Now()
	slog.Debug("unresolving thread", "threadID", threadID)

	vars := map[string]interface{}{
		"threadId": threadID,
	}

	var resp unresolveResponse
	if err := doWithRetry(func() error {
		return c.gql.DoWithContext(ctx, unresolveThreadMutation, vars, &resp)
	}); err != nil {
		return nil, classifyWithContext(err, "thread", threadID)
	}

	t := resp.UnresolveReviewThread.Thread
	slog.Debug("unresolved thread", "threadID", t.ID, "path", t.Path, "line", t.Line, "duration", time.Since(start))

	return &domain.ResolveResult{
		ThreadID:   t.ID,
		Path:       t.Path,
		Line:       t.Line,
		IsResolved: t.IsResolved,
		Action:     "unresolved",
	}, nil
}
