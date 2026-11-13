package github

import (
	"context"
	"fmt"
	"net/url"
)

// FindOpenPRs returns the numbers of open PRs in owner/repo whose head is
// headOwner:branch. GitHub filters on the head owner too, so a fork that
// happens to use the same branch name never matches.
func (c *Client) FindOpenPRs(ctx context.Context, owner, repo, headOwner, branch string) ([]int, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	q := url.Values{"head": {headOwner + ":" + branch}, "state": {"open"}, "per_page": {"100"}}
	path := fmt.Sprintf("repos/%s/%s/pulls?%s", owner, repo, q.Encode())
	var resp []struct {
		Number int `json:"number"`
	}
	if err := doWithRetry(func() error {
		return c.rest.DoWithContext(ctx, "GET", path, nil, &resp)
	}); err != nil {
		return nil, classifyWithContext(err, "repository", owner+"/"+repo)
	}
	nums := make([]int, 0, len(resp))
	for _, pr := range resp {
		nums = append(nums, pr.Number)
	}
	return nums, nil
}
