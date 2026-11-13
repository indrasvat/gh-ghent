package cli

import (
	"os"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
)

// TestResolveRepo covers the -R/--repo flag value itself: valid OWNER/REPO
// passes through, anything else fails with a message naming the format.
func TestResolveRepo(t *testing.T) {
	tests := []struct {
		name     string
		flag     string
		wantOwn  string
		wantRepo string
		wantErr  string
	}{
		{name: "owner/repo", flag: "oven-sh/bun", wantOwn: "oven-sh", wantRepo: "bun"},
		{name: "same owner and repo", flag: "cli/cli", wantOwn: "cli", wantRepo: "cli"},
		{name: "single word", flag: "cli", wantErr: `invalid repo format "cli", expected OWNER/REPO`},
		{name: "missing repo", flag: "oven-sh/", wantErr: `invalid repo format "oven-sh/", expected OWNER/REPO`},
		{name: "missing owner", flag: "/bun", wantErr: `invalid repo format "/bun", expected OWNER/REPO`},
		{name: "extra segment", flag: "oven-sh/bun/extra", wantErr: `invalid repo format "oven-sh/bun/extra", expected OWNER/REPO`},
		{name: "host prefix is not OWNER/REPO", flag: "github.com/oven-sh/bun", wantErr: `invalid repo format "github.com/oven-sh/bun", expected OWNER/REPO`},
		{name: "empty segment", flag: "oven-sh//bun", wantErr: `invalid repo format "oven-sh//bun", expected OWNER/REPO`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, err := resolveRepo(tt.flag)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("resolveRepo(%q) = %q/%q, nil error; want error %q", tt.flag, owner, repo, tt.wantErr)
				}
				if err.Error() != tt.wantErr {
					t.Errorf("resolveRepo(%q) error = %q; want %q", tt.flag, err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveRepo(%q) returned error: %v", tt.flag, err)
			}
			if owner != tt.wantOwn || repo != tt.wantRepo {
				t.Errorf("resolveRepo(%q) = %q/%q; want %q/%q", tt.flag, owner, repo, tt.wantOwn, tt.wantRepo)
			}
		})
	}
}

// TestResolveRepoFlagBeatsRemote proves the -R flag wins over whatever repo
// the checkout points at — and over GH_REPO, which is set to a different
// repo here on purpose. The test runs inside the gh-ghent checkout, so the
// remote (if any) is a different repo than the flag value.
func TestResolveRepoFlagBeatsRemote(t *testing.T) {
	t.Setenv("GH_REPO", "cli/cli")

	owner, repo, err := resolveRepo("oven-sh/bun")
	if err != nil {
		t.Fatalf("resolveRepo(%q) returned error: %v", "oven-sh/bun", err)
	}
	if owner != "oven-sh" || repo != "bun" {
		t.Errorf("resolveRepo(%q) = %q/%q; want oven-sh/bun", "oven-sh/bun", owner, repo)
	}
}

// TestResolveRepoFallsBackToRemote proves the no-flag path consults the
// environment/remote instead of the flag. It compares against
// repository.Current directly so the test does not hard-code this checkout's
// remote. It skips where go-gh cannot determine a repo (e.g. no gh auth, so
// no known hosts to match the remote against).
func TestResolveRepoFallsBackToRemote(t *testing.T) {
	t.Setenv("GH_REPO", "")
	want, err := repository.Current()
	if err != nil {
		t.Skipf("no git remote to compare against: %v", err)
	}
	owner, repo, err := resolveRepo("")
	if err != nil {
		t.Fatalf("resolveRepo(%q) returned error: %v", "", err)
	}
	if owner != want.Owner || repo != want.Name {
		t.Errorf("resolveRepo(%q) = %q/%q; want %q/%q (repository.Current)", "", owner, repo, want.Owner, want.Name)
	}
}

// TestResolveRepoGHRepoFallback proves GH_REPO is honored when no -R flag is
// given, even outside any git checkout.
func TestResolveRepoGHRepoFallback(t *testing.T) {
	chdirOutsideCheckout(t)
	t.Setenv("GH_REPO", "cli/cli")

	owner, repo, err := resolveRepo("")
	if err != nil {
		t.Fatalf("resolveRepo(%q) with GH_REPO=cli/cli returned error: %v", "", err)
	}
	if owner != "cli" || repo != "cli" {
		t.Errorf("resolveRepo(%q) with GH_REPO=cli/cli = %q/%q; want cli/cli", "", owner, repo)
	}
}

// TestResolveRepoGHRepoInvalid proves a malformed GH_REPO surfaces as a
// resolution error instead of silently targeting the wrong repo.
func TestResolveRepoGHRepoInvalid(t *testing.T) {
	chdirOutsideCheckout(t)
	t.Setenv("GH_REPO", "not-a-repo")

	_, _, err := resolveRepo("")
	if err == nil {
		t.Fatal("resolveRepo(\"\") with GH_REPO=not-a-repo returned no error")
	}
	if !strings.Contains(err.Error(), "could not determine repository") {
		t.Errorf("resolveRepo(\"\") error = %q; want it to name repository determination", err.Error())
	}
}

// TestResolveRepoNoRepoError proves the no-flag, no-remote case fails with an
// error that tells the user to pass -R.
func TestResolveRepoNoRepoError(t *testing.T) {
	chdirOutsideCheckout(t)
	t.Setenv("GH_REPO", "")

	_, _, err := resolveRepo("")
	if err == nil {
		t.Fatal("resolveRepo(\"\") outside a checkout returned no error")
	}
	if !strings.Contains(err.Error(), "use -R owner/repo") {
		t.Errorf("resolveRepo(\"\") error = %q; want it to suggest -R owner/repo", err.Error())
	}
}

// TestSubcommandsParseRepoFlag proves every subcommand wires the persistent
// -R flag through to Flags.Repo. The commands run without --pr, outside a
// checkout: with no current branch to look up, --pr resolution fails before
// any API call, so the test makes no network requests with or without auth.
// It fails if a subcommand stops inheriting -R or if PersistentPreRunE stops
// copying it into Flags.
func TestSubcommandsParseRepoFlag(t *testing.T) {
	for _, sub := range []string{"comments", "checks", "resolve", "reply", "dismiss", "status"} {
		t.Run(sub, func(t *testing.T) {
			Flags = GlobalFlags{}
			chdirOutsideCheckout(t)

			cmd := NewRootCmd()
			cmd.SetArgs([]string{sub, "-R", "oven-sh/bun", "--no-tui"})
			_ = cmd.Execute()

			if Flags.Repo != "oven-sh/bun" {
				t.Errorf("%s: Flags.Repo = %q; want %q", sub, Flags.Repo, "oven-sh/bun")
			}
		})
	}
}

// chdirOutsideCheckout moves to an empty directory and drops the GIT_DIR /
// GIT_WORK_TREE a git hook exports, which would otherwise still point git at
// this repo (lefthook's pre-push runs these tests).
func chdirOutsideCheckout(t *testing.T) {
	t.Helper()
	t.Chdir(t.TempDir())
	for _, k := range []string{"GIT_DIR", "GIT_WORK_TREE"} {
		t.Setenv(k, "") // restores the original value after the test
		os.Unsetenv(k)
	}
}
