package github

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/indrasvat/gh-ghent/internal/domain"
)

// reviewThreadsQuery is the GraphQL query for fetching PR review threads.
const reviewThreadsQuery = `
query($owner: String!, $repo: String!, $pr: Int!, $cursor: String) {
  repository(owner: $owner, name: $repo) {
    pullRequest(number: $pr) {
      reviewThreads(first: 100, after: $cursor) {
        totalCount
        nodes {
          id
          isResolved
          isOutdated
          path
          line
          startLine
          viewerCanResolve
          viewerCanUnresolve
          viewerCanReply
          comments(first: 50) {
            nodes {
              id
              databaseId
              body
              author { login }
              path
              diffHunk
              createdAt
              url
            }
          }
        }
        pageInfo {
          hasNextPage
          endCursor
        }
      }
    }
  }
}
`

type threadsResponse struct {
	Repository struct {
		PullRequest *struct {
			ReviewThreads struct {
				TotalCount int          `json:"totalCount"`
				Nodes      []threadNode `json:"nodes"`
				PageInfo   struct {
					HasNextPage bool   `json:"hasNextPage"`
					EndCursor   string `json:"endCursor"`
				} `json:"pageInfo"`
			} `json:"reviewThreads"`
		} `json:"pullRequest"`
	} `json:"repository"`
}

type threadNode struct {
	ID                 string `json:"id"`
	IsResolved         bool   `json:"isResolved"`
	IsOutdated         bool   `json:"isOutdated"`
	Path               string `json:"path"`
	Line               int    `json:"line"`
	StartLine          int    `json:"startLine"`
	ViewerCanResolve   bool   `json:"viewerCanResolve"`
	ViewerCanUnresolve bool   `json:"viewerCanUnresolve"`
	ViewerCanReply     bool   `json:"viewerCanReply"`
	Comments           struct {
		Nodes []commentNode `json:"nodes"`
	} `json:"comments"`
}

type commentNode struct {
	ID         string `json:"id"`
	DatabaseID int64  `json:"databaseId"`
	Body       string `json:"body"`
	Author     struct {
		Login string `json:"login"`
	} `json:"author"`
	Path      string `json:"path"`
	DiffHunk  string `json:"diffHunk"`
	CreatedAt string `json:"createdAt"`
	URL       string `json:"url"`
}

// FetchThreads retrieves all review threads for a PR via GraphQL,
// paginating through results and filtering to unresolved threads only.
func (c *Client) FetchThreads(ctx context.Context, owner, repo string, pr int) (*domain.CommentsResult, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	start := time.Now()
	slog.Debug("fetching review threads", "owner", owner, "repo", repo, "pr", pr)

	var allNodes []threadNode
	var totalCount int
	var cursor *string
	page := 1

	for {
		vars := map[string]interface{}{
			"owner": owner,
			"repo":  repo,
			"pr":    pr,
		}
		if cursor != nil {
			vars["cursor"] = *cursor
		}

		slog.Debug("fetching threads page", "page", page)

		var resp threadsResponse
		if err := doWithRetry(func() error {
			return c.gql.DoWithContext(ctx, reviewThreadsQuery, vars, &resp)
		}); err != nil {
			return nil, classifyWithContext(err, "pull request", fmt.Sprintf("PR #%d in %s/%s", pr, owner, repo))
		}

		if resp.Repository.PullRequest == nil {
			return nil, &NotFoundError{
				Resource: "pull request",
				Detail:   fmt.Sprintf("PR #%d in %s/%s", pr, owner, repo),
			}
		}

		rt := resp.Repository.PullRequest.ReviewThreads
		totalCount = rt.TotalCount
		allNodes = append(allNodes, rt.Nodes...)

		if !rt.PageInfo.HasNextPage {
			break
		}
		cursor = &rt.PageInfo.EndCursor
		page++
	}

	slog.Debug("fetched review threads", "owner", owner, "repo", repo, "pr", pr,
		"total", totalCount, "fetched", len(allNodes), "duration", time.Since(start))

	return mapThreadsToResult(pr, totalCount, allNodes)
}

// FetchResolvedThreads retrieves all resolved review threads for a PR.
// Used by --all --unresolve to find threads that can be unresolved.
func (c *Client) FetchResolvedThreads(ctx context.Context, owner, repo string, pr int) (*domain.CommentsResult, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	start := time.Now()
	slog.Debug("fetching resolved threads", "owner", owner, "repo", repo, "pr", pr)

	var allNodes []threadNode
	var totalCount int
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
		if err := doWithRetry(func() error {
			return c.gql.DoWithContext(ctx, reviewThreadsQuery, vars, &resp)
		}); err != nil {
			return nil, classifyWithContext(err, "pull request", fmt.Sprintf("PR #%d in %s/%s", pr, owner, repo))
		}

		if resp.Repository.PullRequest == nil {
			return nil, &NotFoundError{
				Resource: "pull request",
				Detail:   fmt.Sprintf("PR #%d in %s/%s", pr, owner, repo),
			}
		}

		rt := resp.Repository.PullRequest.ReviewThreads
		totalCount = rt.TotalCount
		allNodes = append(allNodes, rt.Nodes...)

		if !rt.PageInfo.HasNextPage {
			break
		}
		cursor = &rt.PageInfo.EndCursor
	}

	slog.Debug("fetched resolved threads", "owner", owner, "repo", repo, "pr", pr,
		"total", totalCount, "fetched", len(allNodes), "duration", time.Since(start))

	return mapThreadsWithFilter(pr, totalCount, allNodes, true)
}

// mapThreadsToResult converts GraphQL thread nodes to a domain CommentsResult,
// filtering to unresolved threads only.
func mapThreadsToResult(pr, totalCount int, nodes []threadNode) (*domain.CommentsResult, error) {
	return mapThreadsWithFilter(pr, totalCount, nodes, false)
}

// mapThreadsWithFilter converts GraphQL thread nodes to a domain CommentsResult.
// When keepResolved is false, only unresolved threads are included (default for comments).
// When keepResolved is true, only resolved threads are included (for --all --unresolve).
func mapThreadsWithFilter(pr, totalCount int, nodes []threadNode, keepResolved bool) (*domain.CommentsResult, error) {
	var resolved, unresolved int
	var threads []domain.ReviewThread

	for _, n := range nodes {
		if n.IsResolved {
			resolved++
		} else {
			unresolved++
		}

		if n.IsResolved != keepResolved {
			continue
		}

		comments, err := mapComments(n.Comments.Nodes)
		if err != nil {
			return nil, err
		}

		threads = append(threads, domain.ReviewThread{
			ID:                 n.ID,
			Path:               n.Path,
			Line:               n.Line,
			StartLine:          n.StartLine,
			IsResolved:         n.IsResolved,
			IsOutdated:         n.IsOutdated,
			ViewerCanResolve:   n.ViewerCanResolve,
			ViewerCanUnresolve: n.ViewerCanUnresolve,
			ViewerCanReply:     n.ViewerCanReply,
			Comments:           comments,
		})
	}

	return &domain.CommentsResult{
		PRNumber:        pr,
		Threads:         threads,
		TotalCount:      totalCount,
		ResolvedCount:   resolved,
		UnresolvedCount: unresolved,
	}, nil
}

func mapComments(nodes []commentNode) ([]domain.Comment, error) {
	comments := make([]domain.Comment, 0, len(nodes))
	for _, cn := range nodes {
		t, err := time.Parse(time.RFC3339, cn.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("parse comment time %q: %w", cn.CreatedAt, err)
		}
		comments = append(comments, domain.Comment{
			ID:         cn.ID,
			DatabaseID: cn.DatabaseID,
			Author:     cn.Author.Login,
			Body:       cn.Body,
			CreatedAt:  t,
			URL:        cn.URL,
			DiffHunk:   cn.DiffHunk,
			Path:       cn.Path,
		})
	}
	return comments, nil
}
