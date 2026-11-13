package cli

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// prFinder looks up open PRs by head branch.
type prFinder interface {
	FindOpenPRs(ctx context.Context, owner, repo, headOwner, branch string) ([]int, error)
}

// gitCurrentBranch returns the checked-out branch, or an error on a detached HEAD.
func gitCurrentBranch(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx, "git", "branch", "--show-current").Output()
	if err != nil {
		return "", fmt.Errorf("git branch --show-current: %w", err)
	}
	branch := strings.TrimSpace(string(out))
	if branch == "" {
		return "", fmt.Errorf("HEAD is detached")
	}
	return branch, nil
}

// resolvePR turns the --pr value into a PR number. The value is a number, a
// head branch, or OWNER:BRANCH for a PR opened from a fork; when empty, the
// current git branch is used. A branch's head owner defaults to the target
// repo's owner, and exactly one open PR must match.
func resolvePR(ctx context.Context, finder prFinder, currentBranch func(context.Context) (string, error), owner, repo, arg string) (int, error) {
	headOwner, branch := owner, arg
	if arg == "" {
		// An inferred branch is always a branch, even one named "123".
		current, err := currentBranch(ctx)
		if err != nil {
			return 0, fmt.Errorf("--pr is required: no current branch to look up (%w)", err)
		}
		branch = current
	} else if n, err := strconv.Atoi(arg); err == nil {
		if n <= 0 {
			return 0, fmt.Errorf("invalid --pr %q: want a PR number, BRANCH, or OWNER:BRANCH", arg)
		}
		return n, nil
	} else if o, b, ok := strings.Cut(arg, ":"); ok {
		headOwner, branch = o, b
	}
	if headOwner == "" || branch == "" {
		return 0, fmt.Errorf("invalid --pr %q: want a PR number, BRANCH, or OWNER:BRANCH", arg)
	}

	nums, err := finder.FindOpenPRs(ctx, owner, repo, headOwner, branch)
	if err != nil {
		return 0, fmt.Errorf("look up PR for %s:%s: %w", headOwner, branch, err)
	}
	switch len(nums) {
	case 1:
		return nums[0], nil
	case 0:
		return 0, fmt.Errorf("no open PR for %s:%s in %s/%s; pass --pr <number>, or --pr <fork-owner>:%s for a PR opened from a fork",
			headOwner, branch, owner, repo, branch)
	default:
		refs := make([]string, len(nums))
		for i, n := range nums {
			refs[i] = "#" + strconv.Itoa(n)
		}
		return 0, fmt.Errorf("%s:%s has %d open PRs in %s/%s (%s); pass --pr <number>",
			headOwner, branch, len(nums), owner, repo, strings.Join(refs, ", "))
	}
}
