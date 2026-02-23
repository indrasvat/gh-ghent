package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/indrasvat/gh-ghent/internal/domain"
)

// replyResponse represents the REST API response for a review comment reply.
type replyResponse struct {
	ID        int64  `json:"id"`
	NodeID    string `json:"node_id"`
	Body      string `json:"body"`
	HTMLURL   string `json:"html_url"`
	CreatedAt string `json:"created_at"`
}

// ReplyToThread validates a thread exists via GraphQL, then posts a reply via REST.
func (c *Client) ReplyToThread(ctx context.Context, owner, repo string, pr int, threadID, body string) (*domain.ReplyResult, error) {
	thread, err := c.findThreadByID(ctx, owner, repo, pr, threadID)
	if err != nil {
		return nil, err
	}

	if !thread.ViewerCanReply {
		return nil, fmt.Errorf("reply: viewer cannot reply to thread %s", threadID)
	}

	if len(thread.Comments.Nodes) == 0 {
		return nil, fmt.Errorf("reply: thread %s has no comments", threadID)
	}

	// REST reply targets the last comment's databaseId.
	lastComment := thread.Comments.Nodes[len(thread.Comments.Nodes)-1]
	commentID := lastComment.DatabaseID

	reqBody := struct {
		Body string `json:"body"`
	}{Body: body}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(reqBody); err != nil {
		return nil, fmt.Errorf("reply: encode body: %w", err)
	}

	endpoint := fmt.Sprintf("repos/%s/%s/pulls/%d/comments/%d/replies", owner, repo, pr, commentID)

	var resp replyResponse
	if err := c.rest.DoWithContext(ctx, http.MethodPost, endpoint, &buf, &resp); err != nil {
		return nil, fmt.Errorf("reply: REST post: %w", err)
	}

	createdAt, err := time.Parse(time.RFC3339, resp.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("reply: parse created_at %q: %w", resp.CreatedAt, err)
	}

	return &domain.ReplyResult{
		ThreadID:  threadID,
		CommentID: resp.ID,
		URL:       resp.HTMLURL,
		Body:      resp.Body,
		CreatedAt: createdAt,
	}, nil
}

// findThreadByID fetches all review threads and finds the one matching threadID.
// Unlike FetchThreads, this searches all threads (including resolved).
func (c *Client) findThreadByID(ctx context.Context, owner, repo string, pr int, threadID string) (*threadNode, error) {
	var cursor *string

	for {
		vars := map[string]interface{}{
			"owner": owner,
			"repo":  repo,
			"pr":    pr,
		}
		if cursor != nil {
			vars["cursor"] = *cursor
		}

		var resp threadsResponse
		if err := c.gql.DoWithContext(ctx, reviewThreadsQuery, vars, &resp); err != nil {
			return nil, fmt.Errorf("reply: graphql thread lookup: %w", err)
		}

		rt := resp.Repository.PullRequest.ReviewThreads
		for i := range rt.Nodes {
			if rt.Nodes[i].ID == threadID {
				return &rt.Nodes[i], nil
			}
		}

		if !rt.PageInfo.HasNextPage {
			break
		}
		cursor = &rt.PageInfo.EndCursor
	}

	return nil, fmt.Errorf("reply: thread %s not found", threadID)
}
