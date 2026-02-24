package cli

import (
	"errors"
	"fmt"
	"time"

	"github.com/indrasvat/gh-ghent/internal/github"
)

// FormatError returns a user-friendly error message and exit code for the error.
// Exit code 2 is used for auth, rate limit, and not-found errors.
// Exit code 1 is used for all other errors.
func FormatError(err error) (string, int) {
	var authErr *github.AuthError
	if errors.As(err, &authErr) {
		return "Not authenticated. Run `gh auth login` first.", 2
	}

	var rlErr *github.RateLimitError
	if errors.As(err, &rlErr) {
		if !rlErr.ResetAt.IsZero() {
			return fmt.Sprintf("Rate limit exceeded. Resets at %s.", rlErr.ResetAt.Format(time.Kitchen)), 2
		}
		return "Rate limit exceeded. Try again later.", 2
	}

	var nfErr *github.NotFoundError
	if errors.As(err, &nfErr) {
		return nfErr.Error() + ".", 2
	}

	return err.Error(), 1
}
