package github

import (
	"context"
	"fmt"
	"time"

	"github.com/indrasvat/gh-ghent/internal/domain"
)

// checkRunsResponse represents the REST API response for listing check runs.
type checkRunsResponse struct {
	TotalCount int            `json:"total_count"`
	CheckRuns  []checkRunNode `json:"check_runs"`
}

type checkRunNode struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	Status      string  `json:"status"`
	Conclusion  *string `json:"conclusion"` // null when not completed
	StartedAt   string  `json:"started_at"`
	CompletedAt *string `json:"completed_at"` // null when not completed
	HTMLURL     string  `json:"html_url"`
	Output      struct {
		AnnotationsCount int `json:"annotations_count"`
	} `json:"output"`
}

type annotationNode struct {
	Path            string `json:"path"`
	StartLine       int    `json:"start_line"`
	EndLine         int    `json:"end_line"`
	AnnotationLevel string `json:"annotation_level"`
	Title           string `json:"title"`
	Message         string `json:"message"`
}

// fetchHeadSHA resolves a PR number to its HEAD commit SHA via REST.
func (c *Client) fetchHeadSHA(ctx context.Context, owner, repo string, pr int) (string, error) {
	path := fmt.Sprintf("repos/%s/%s/pulls/%d", owner, repo, pr)
	var resp struct {
		Head struct {
			SHA string `json:"sha"`
		} `json:"head"`
	}
	if err := c.rest.DoWithContext(ctx, "GET", path, nil, &resp); err != nil {
		return "", fmt.Errorf("get PR head SHA: %w", err)
	}
	return resp.Head.SHA, nil
}

// fetchCheckRuns retrieves all check runs for a commit SHA, paginating through results.
func (c *Client) fetchCheckRuns(ctx context.Context, owner, repo, ref string) ([]checkRunNode, error) {
	var allRuns []checkRunNode
	page := 1
	for {
		path := fmt.Sprintf("repos/%s/%s/commits/%s/check-runs?per_page=100&page=%d", owner, repo, ref, page)
		var resp checkRunsResponse
		if err := c.rest.DoWithContext(ctx, "GET", path, nil, &resp); err != nil {
			return nil, fmt.Errorf("list check runs: %w", err)
		}
		allRuns = append(allRuns, resp.CheckRuns...)
		if len(allRuns) >= resp.TotalCount {
			break
		}
		page++
	}
	return allRuns, nil
}

// fetchAnnotations retrieves annotations for a single check run.
func (c *Client) fetchAnnotations(ctx context.Context, owner, repo string, checkRunID int64) ([]annotationNode, error) {
	path := fmt.Sprintf("repos/%s/%s/check-runs/%d/annotations", owner, repo, checkRunID)
	var resp []annotationNode
	if err := c.rest.DoWithContext(ctx, "GET", path, nil, &resp); err != nil {
		return nil, fmt.Errorf("list annotations for check %d: %w", checkRunID, err)
	}
	return resp, nil
}

// FetchChecks retrieves all check runs for a PR, fetches annotations for
// failed checks, and aggregates the overall status.
func (c *Client) FetchChecks(ctx context.Context, owner, repo string, pr int) (*domain.ChecksResult, error) {
	sha, err := c.fetchHeadSHA(ctx, owner, repo, pr)
	if err != nil {
		return nil, err
	}

	runs, err := c.fetchCheckRuns(ctx, owner, repo, sha)
	if err != nil {
		return nil, err
	}

	return mapChecksToDomain(ctx, c, owner, repo, pr, sha, runs)
}

// mapChecksToDomain converts REST check run nodes to a domain ChecksResult,
// fetching annotations for failed checks.
func mapChecksToDomain(ctx context.Context, c *Client, owner, repo string, pr int, sha string, runs []checkRunNode) (*domain.ChecksResult, error) {
	checks := make([]domain.CheckRun, 0, len(runs))
	var statuses []domain.OverallStatus
	var passCount, failCount, pendingCount int

	for _, run := range runs {
		check := domain.CheckRun{
			ID:      run.ID,
			Name:    run.Name,
			Status:  run.Status,
			HTMLURL: run.HTMLURL,
		}

		if run.Conclusion != nil {
			check.Conclusion = *run.Conclusion
		}
		if run.StartedAt != "" {
			t, err := time.Parse(time.RFC3339, run.StartedAt)
			if err != nil {
				return nil, fmt.Errorf("parse started_at for check %q: %w", run.Name, err)
			}
			check.StartedAt = t
		}
		if run.CompletedAt != nil && *run.CompletedAt != "" {
			t, err := time.Parse(time.RFC3339, *run.CompletedAt)
			if err != nil {
				return nil, fmt.Errorf("parse completed_at for check %q: %w", run.Name, err)
			}
			check.CompletedAt = t
		}

		status := classifyCheckStatus(run.Status, check.Conclusion)
		statuses = append(statuses, status)

		switch status {
		case domain.StatusPass:
			passCount++
		case domain.StatusFail:
			failCount++
		case domain.StatusPending:
			pendingCount++
		}

		// Fetch annotations for failed checks that have annotations
		if status == domain.StatusFail && run.Output.AnnotationsCount > 0 {
			annots, err := c.fetchAnnotations(ctx, owner, repo, run.ID)
			if err != nil {
				return nil, err
			}
			for _, a := range annots {
				check.Annotations = append(check.Annotations, domain.Annotation{
					Path:            a.Path,
					StartLine:       a.StartLine,
					EndLine:         a.EndLine,
					AnnotationLevel: a.AnnotationLevel,
					Title:           a.Title,
					Message:         a.Message,
				})
			}
		}

		checks = append(checks, check)
	}

	return &domain.ChecksResult{
		PRNumber:      pr,
		HeadSHA:       sha,
		OverallStatus: domain.AggregateStatus(statuses),
		Checks:        checks,
		PassCount:     passCount,
		FailCount:     failCount,
		PendingCount:  pendingCount,
	}, nil
}

// classifyCheckStatus maps a check run's status and conclusion to an OverallStatus.
func classifyCheckStatus(status, conclusion string) domain.OverallStatus {
	if status != "completed" {
		return domain.StatusPending
	}
	switch conclusion {
	case "success", "neutral", "skipped":
		return domain.StatusPass
	case "failure", "timed_out", "action_required", "startup_failure", "stale":
		return domain.StatusFail
	case "cancelled":
		return domain.StatusFail
	default:
		return domain.StatusPending
	}
}
