package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/indrasvat/gh-ghent/internal/github"
)

// FormatError returns a user-friendly error message and exit code for the error.
// Exit code 2 is used for auth, rate limit, and not-found errors.
// Exit code 1 is used for all other errors.
func FormatError(err error) (string, int) {
	var authErr *github.AuthError
	if errors.As(err, &authErr) {
		if authErr.NoToken {
			return "No GitHub token found. ghent needs one even for public repos: review threads " +
				"come from GitHub's GraphQL API, which rejects anonymous requests. " +
				"Set GH_TOKEN (or GITHUB_TOKEN), or run `gh auth login`.", 2
		}
		msg := "GitHub rejected the token (401 Bad credentials). Check GH_TOKEN, or run `gh auth status`."
		if !strings.EqualFold(os.Getenv("GH_GHENT_AUTH_SCHEME"), "bearer") {
			msg += " If a vault or proxy swaps the real token in only on Bearer requests, set GH_GHENT_AUTH_SCHEME=bearer."
		}
		return msg, 2
	}

	var permErr *github.PermissionError
	if errors.As(err, &permErr) {
		return fmt.Sprintf("GitHub denied the request (403: %s). The token is valid but lacks a permission this command needs.", permErr.Message), 2
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
