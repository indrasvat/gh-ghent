package cli

import (
	"fmt"
	"strconv"

	"github.com/indrasvat/gh-ghent/internal/domain"
)

func staleBlockingReviews(reviews []domain.Review) []domain.Review {
	out := make([]domain.Review, 0, len(reviews))
	for _, r := range reviews {
		if r.State == domain.ReviewChangesRequested && r.IsStale {
			out = append(out, r)
		}
	}
	return out
}

func matchReviewSelector(review domain.Review, selector string) bool {
	if selector == "" {
		return true
	}
	return selector == review.ID || selector == strconv.FormatInt(review.DatabaseID, 10)
}

func selectDismissReviews(
	reviews []domain.Review,
	selector string,
	author string,
	botsOnly bool,
) ([]domain.Review, error) {
	candidates := staleBlockingReviews(reviews)
	selected := make([]domain.Review, 0, len(candidates))

	for _, review := range candidates {
		if !matchReviewSelector(review, selector) {
			continue
		}
		if author != "" && author != review.Author {
			continue
		}
		if botsOnly && !review.IsBot {
			continue
		}
		selected = append(selected, review)
	}

	if selector != "" && len(selected) == 0 {
		return nil, fmt.Errorf("review %q is not a stale CHANGES_REQUESTED review on this PR", selector)
	}

	return selected, nil
}
