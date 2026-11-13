package github

import (
	"context"
	"errors"
	"fmt"

	"github.com/indrasvat/gh-ghent/internal/domain"
)

type workflowRunsResponse struct {
	TotalCount   int `json:"total_count"`
	WorkflowRuns []struct {
		ID int64 `json:"id"`
	} `json:"workflow_runs"`
}

// jobsResponse lists a workflow run's jobs. A job carries the same fields as
// its check run, and its ID is that check run's ID, so job logs still resolve.
type jobsResponse struct {
	TotalCount int            `json:"total_count"`
	Jobs       []checkRunNode `json:"jobs"`
}

type combinedStatusResponse struct {
	TotalCount int `json:"total_count"`
	Statuses   []struct {
		ID        int64   `json:"id"`
		State     string  `json:"state"` // error, failure, pending, success
		Context   string  `json:"context"`
		TargetURL *string `json:"target_url"`
		CreatedAt string  `json:"created_at"`
		UpdatedAt string  `json:"updated_at"`
	} `json:"statuses"`
}

// fetchChecksFallback rebuilds a PR's checks from GitHub Actions jobs and
// commit statuses, for tokens that may not read check runs. Third-party
// check runs (apps that are not Actions) and annotations are not visible
// this way, which the result's Fallback text says.
func (c *Client) fetchChecksFallback(ctx context.Context, owner, repo string, pr int, sha string, denied *PermissionError) (*domain.ChecksResult, error) {
	jobs, err := c.fetchActionsJobs(ctx, owner, repo, sha)
	if err != nil {
		return nil, fallbackError(denied, "Actions jobs", "actions: read", err)
	}
	statuses, err := c.fetchCommitStatuses(ctx, owner, repo, sha)
	if err != nil {
		return nil, fallbackError(denied, "commit statuses", "commit statuses: read", err)
	}
	result, err := mapChecksToDomain(ctx, c, owner, repo, pr, sha, append(jobs, statuses...))
	if err != nil {
		return nil, err
	}
	result.Fallback = fmt.Sprintf("check runs are not readable with this token (403: %s); "+
		"checks were built from GitHub Actions jobs and commit statuses instead, "+
		"so check runs from other apps and annotations are missing", denied.Message)
	return result, nil
}

// fallbackError reports that check runs were denied and the fallback source
// failed too, naming the permission that would make the fallback work.
func fallbackError(denied *PermissionError, source, permission string, err error) error {
	var also *PermissionError
	if errors.As(err, &also) {
		return &PermissionError{
			Message: fmt.Sprintf("check runs: %s; %s: %s. Grant the token %q to read checks without the Checks permission",
				denied.Message, source, also.Message, permission),
			Err: err,
		}
	}
	return fmt.Errorf("check runs denied (%s); fetch %s instead: %w", denied.Message, source, err)
}

func (c *Client) fetchActionsJobs(ctx context.Context, owner, repo, sha string) ([]checkRunNode, error) {
	var runIDs []int64
	for page := 1; ; page++ {
		path := fmt.Sprintf("repos/%s/%s/actions/runs?head_sha=%s&per_page=100&page=%d", owner, repo, sha, page)
		var resp workflowRunsResponse
		if err := doWithRetry(func() error { return c.rest.DoWithContext(ctx, "GET", path, nil, &resp) }); err != nil {
			return nil, classifyError(err)
		}
		for _, r := range resp.WorkflowRuns {
			runIDs = append(runIDs, r.ID)
		}
		if len(resp.WorkflowRuns) == 0 || len(runIDs) >= resp.TotalCount {
			break
		}
	}

	var nodes []checkRunNode
	for _, runID := range runIDs {
		seen := 0
		for page := 1; ; page++ {
			// The default filter=latest returns each job's most recent attempt,
			// matching what check runs report.
			path := fmt.Sprintf("repos/%s/%s/actions/runs/%d/jobs?per_page=100&page=%d", owner, repo, runID, page)
			var resp jobsResponse
			if err := doWithRetry(func() error { return c.rest.DoWithContext(ctx, "GET", path, nil, &resp) }); err != nil {
				return nil, classifyError(err)
			}
			nodes = append(nodes, resp.Jobs...)
			seen += len(resp.Jobs)
			if len(resp.Jobs) == 0 || seen >= resp.TotalCount {
				break
			}
		}
	}
	return nodes, nil
}

func (c *Client) fetchCommitStatuses(ctx context.Context, owner, repo, sha string) ([]checkRunNode, error) {
	var nodes []checkRunNode
	for page := 1; ; page++ {
		// The combined status holds the latest status per context.
		path := fmt.Sprintf("repos/%s/%s/commits/%s/status?per_page=100&page=%d", owner, repo, sha, page)
		var resp combinedStatusResponse
		if err := doWithRetry(func() error { return c.rest.DoWithContext(ctx, "GET", path, nil, &resp) }); err != nil {
			return nil, classifyError(err)
		}
		for _, s := range resp.Statuses {
			n := checkRunNode{ID: s.ID, Name: s.Context, Status: "completed", StartedAt: s.CreatedAt}
			if s.TargetURL != nil {
				n.HTMLURL = *s.TargetURL
			}
			conclusion := s.State
			switch s.State {
			case "pending":
				n.Status = "in_progress"
			case "error":
				conclusion = "failure"
			}
			if n.Status == "completed" {
				n.Conclusion = &conclusion
				n.CompletedAt = &s.UpdatedAt
			}
			nodes = append(nodes, n)
		}
		if len(resp.Statuses) == 0 || len(nodes) >= resp.TotalCount {
			break
		}
	}
	return nodes, nil
}
