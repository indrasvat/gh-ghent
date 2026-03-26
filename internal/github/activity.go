package github

import (
	"context"
	"crypto/sha256"
	"fmt"
	"slices"
	"time"

	"github.com/indrasvat/gh-ghent/internal/domain"
)

// activityProbeQuery fetches minimal metadata for review settlement fingerprinting.
// This is deliberately lightweight: no comment bodies, no diff hunks, no deep pagination.
// One query per poll cycle at 15s intervals.
//
// Uses updatedAt on the last comment (not just createdAt) to detect comment edits.
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
              updatedAt
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
							UpdatedAt string `json:"updatedAt"`
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

// threadEntry groups thread metadata for sorted fingerprinting.
// Sorting IDs alone would mis-pair parallel slices.
type threadEntry struct {
	id       string
	resolved bool
	editedAt time.Time
}

// reviewEntry groups review metadata for sorted fingerprinting.
type reviewEntry struct {
	id          string
	state       string
	submittedAt time.Time
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

	// Build thread entries — keep ID and metadata together for correct sorting.
	threads := make([]threadEntry, 0, len(pr_.ReviewThreads.Nodes))
	for _, t := range pr_.ReviewThreads.Nodes {
		entry := threadEntry{id: t.ID, resolved: t.IsResolved}
		if len(t.Comments.Nodes) > 0 {
			c := t.Comments.Nodes[0]
			// Prefer updatedAt (tracks edits); fall back to createdAt.
			if c.UpdatedAt != "" {
				entry.editedAt, _ = time.Parse(time.RFC3339, c.UpdatedAt)
			} else if c.CreatedAt != "" {
				entry.editedAt, _ = time.Parse(time.RFC3339, c.CreatedAt)
			}
		}
		threads = append(threads, entry)
	}

	// Sort by ID for deterministic fingerprinting — keeps parallel fields aligned.
	slices.SortFunc(threads, func(a, b threadEntry) int {
		if a.id < b.id {
			return -1
		}
		if a.id > b.id {
			return 1
		}
		return 0
	})

	// Build review entries — merge reviews + latestReviews, deduplicate by ID.
	seenReview := make(map[string]bool)
	reviews := make([]reviewEntry, 0, len(pr_.Reviews.Nodes)+len(pr_.LatestReviews.Nodes))

	allNodes := make([]struct {
		ID          string `json:"id"`
		State       string `json:"state"`
		SubmittedAt string `json:"submittedAt"`
	}, 0, len(pr_.Reviews.Nodes)+len(pr_.LatestReviews.Nodes))
	allNodes = append(allNodes, pr_.Reviews.Nodes...)
	allNodes = append(allNodes, pr_.LatestReviews.Nodes...)

	for _, r := range allNodes {
		if seenReview[r.ID] {
			continue
		}
		seenReview[r.ID] = true
		ts, _ := time.Parse(time.RFC3339, r.SubmittedAt)
		reviews = append(reviews, reviewEntry{id: r.ID, state: r.State, submittedAt: ts})
	}

	slices.SortFunc(reviews, func(a, b reviewEntry) int {
		if a.id < b.id {
			return -1
		}
		if a.id > b.id {
			return 1
		}
		return 0
	})

	// Build snapshot from sorted entries.
	snap := &domain.ActivitySnapshot{
		HeadSHA:     pr_.HeadRefOid,
		ThreadCount: pr_.ReviewThreads.TotalCount,
		ReviewCount: pr_.Reviews.TotalCount,
	}
	for _, t := range threads {
		snap.ThreadIDs = append(snap.ThreadIDs, t.id)
		snap.ThreadStates = append(snap.ThreadStates, t.resolved)
		snap.ThreadEdits = append(snap.ThreadEdits, t.editedAt)
	}
	for _, r := range reviews {
		snap.ReviewIDs = append(snap.ReviewIDs, r.id)
		snap.ReviewStates = append(snap.ReviewStates, r.state)
		snap.ReviewTimes = append(snap.ReviewTimes, r.submittedAt)
	}

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
