package github

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sort"
	"time"

	"github.com/indrasvat/gh-ghent/internal/domain"
)

// activityProbeQuery fetches minimal metadata for review settlement fingerprinting.
// This is deliberately lightweight: no comment bodies, no diff hunks, no deep pagination.
// One query per poll cycle at 15s intervals.
//
// Note: PullRequestReviewThread does not expose updatedAt, so we use the
// last comment's createdAt as a proxy for thread edit activity.
const activityProbeQuery = `
query($owner: String!, $repo: String!, $pr: Int!) {
  repository(owner: $owner, name: $repo) {
    pullRequest(number: $pr) {
      headRefOid
      reviewThreads(first: 100) {
        totalCount
        nodes {
          id
          isResolved
          comments(last: 1) {
            nodes {
              createdAt
            }
          }
        }
      }
      reviews(first: 50) {
        totalCount
        nodes {
          id
          state
          submittedAt
        }
      }
      latestReviews(first: 10) {
        nodes {
          id
          state
          submittedAt
        }
      }
    }
  }
}
`

type activityResponse struct {
	Repository struct {
		PullRequest *struct {
			HeadRefOid    string `json:"headRefOid"`
			ReviewThreads struct {
				TotalCount int `json:"totalCount"`
				Nodes      []struct {
					ID         string `json:"id"`
					IsResolved bool   `json:"isResolved"`
					Comments   struct {
						Nodes []struct {
							CreatedAt string `json:"createdAt"`
						} `json:"nodes"`
					} `json:"comments"`
				} `json:"nodes"`
			} `json:"reviewThreads"`
			Reviews struct {
				TotalCount int `json:"totalCount"`
				Nodes      []struct {
					ID          string `json:"id"`
					State       string `json:"state"`
					SubmittedAt string `json:"submittedAt"`
				} `json:"nodes"`
			} `json:"reviews"`
			LatestReviews struct {
				Nodes []struct {
					ID          string `json:"id"`
					State       string `json:"state"`
					SubmittedAt string `json:"submittedAt"`
				} `json:"nodes"`
			} `json:"latestReviews"`
		} `json:"pullRequest"`
	} `json:"repository"`
}

// ProbeActivity fetches lightweight review activity metadata for settlement fingerprinting.
func (c *Client) ProbeActivity(ctx context.Context, owner, repo string, pr int) (*domain.ActivitySnapshot, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	vars := map[string]any{
		"owner": owner,
		"repo":  repo,
		"pr":    pr,
	}

	var resp activityResponse
	err := doWithRetry(func() error {
		return c.gql.DoWithContext(ctx, activityProbeQuery, vars, &resp)
	})
	if err != nil {
		return nil, fmt.Errorf("probe activity: %w", classifyError(err))
	}
	if resp.Repository.PullRequest == nil {
		return nil, fmt.Errorf("probe activity: pull request #%d not found", pr)
	}

	pr_ := resp.Repository.PullRequest

	snap := &domain.ActivitySnapshot{
		HeadSHA:     pr_.HeadRefOid,
		ThreadCount: pr_.ReviewThreads.TotalCount,
		ReviewCount: pr_.Reviews.TotalCount,
	}

	// Thread metadata — use last comment's createdAt as proxy for thread activity
	// (PullRequestReviewThread does not expose updatedAt).
	for _, t := range pr_.ReviewThreads.Nodes {
		snap.ThreadIDs = append(snap.ThreadIDs, t.ID)
		snap.ThreadStates = append(snap.ThreadStates, t.IsResolved)
		var lastCommentAt time.Time
		if len(t.Comments.Nodes) > 0 {
			lastCommentAt, _ = time.Parse(time.RFC3339, t.Comments.Nodes[0].CreatedAt)
		}
		snap.ThreadEdits = append(snap.ThreadEdits, lastCommentAt)
	}

	// Review metadata — merge reviews + latestReviews for completeness, deduplicate by ID.
	seenReview := make(map[string]bool)
	allReviewNodes := make([]struct {
		ID          string `json:"id"`
		State       string `json:"state"`
		SubmittedAt string `json:"submittedAt"`
	}, 0, len(pr_.Reviews.Nodes)+len(pr_.LatestReviews.Nodes))
	allReviewNodes = append(allReviewNodes, pr_.Reviews.Nodes...)
	allReviewNodes = append(allReviewNodes, pr_.LatestReviews.Nodes...)
	for _, r := range allReviewNodes {
		if seenReview[r.ID] {
			continue
		}
		seenReview[r.ID] = true
		snap.ReviewIDs = append(snap.ReviewIDs, r.ID)
		snap.ReviewStates = append(snap.ReviewStates, r.State)
		ts, _ := time.Parse(time.RFC3339, r.SubmittedAt)
		snap.ReviewTimes = append(snap.ReviewTimes, ts)
	}

	// Sort for deterministic fingerprinting.
	sort.Strings(snap.ThreadIDs)
	sort.Strings(snap.ReviewIDs)

	return snap, nil
}

// Fingerprint computes a SHA-256 hash of the activity snapshot for change detection.
// Any structural change — new thread, edited thread, resolved thread, new review,
// review state change, or new push — produces a different hash.
func Fingerprint(snap *domain.ActivitySnapshot) string {
	h := sha256.New()
	fmt.Fprintf(h, "head:%s\n", snap.HeadSHA)
	fmt.Fprintf(h, "tc:%d rc:%d\n", snap.ThreadCount, snap.ReviewCount)
	for i, id := range snap.ThreadIDs {
		resolved := false
		if i < len(snap.ThreadStates) {
			resolved = snap.ThreadStates[i]
		}
		var edited time.Time
		if i < len(snap.ThreadEdits) {
			edited = snap.ThreadEdits[i]
		}
		fmt.Fprintf(h, "t:%s:%v:%d\n", id, resolved, edited.UnixNano())
	}
	for i, id := range snap.ReviewIDs {
		state := ""
		if i < len(snap.ReviewStates) {
			state = snap.ReviewStates[i]
		}
		var submitted time.Time
		if i < len(snap.ReviewTimes) {
			submitted = snap.ReviewTimes[i]
		}
		fmt.Fprintf(h, "r:%s:%s:%d\n", id, state, submitted.UnixNano())
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}
