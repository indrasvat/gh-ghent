package cli

import (
	"fmt"
	"strings"

	"github.com/cli/go-gh/v2/pkg/repository"
)

// resolveRepo returns owner and repo name from the flag value or git remote.
func resolveRepo(flag string) (string, string, error) {
	if flag != "" {
		parts := strings.SplitN(flag, "/", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return "", "", fmt.Errorf("invalid repo format %q, expected OWNER/REPO", flag)
		}
		return parts[0], parts[1], nil
	}

	repo, err := repository.Current()
	if err != nil {
		return "", "", fmt.Errorf("could not determine repository: %w (use -R owner/repo)", err)
	}
	return repo.Owner, repo.Name, nil
}
