package cli

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type fakeFinder struct {
	gotHeadOwner, gotBranch string
	nums                    []int
	err                     error
}

func (f *fakeFinder) FindOpenPRs(_ context.Context, _, _, headOwner, branch string) ([]int, error) {
	f.gotHeadOwner, f.gotBranch = headOwner, branch
	return f.nums, f.err
}

func TestResolvePR(t *testing.T) {
	tests := []struct {
		name          string
		arg           string
		current       string // what `git branch --show-current` reports
		nums          []int
		want          int
		wantHeadOwner string
		wantBranch    string
		wantErr       string
	}{
		{name: "number needs no lookup", arg: "42", want: 42},
		{name: "branch defaults head owner to repo owner", arg: "feat/x", nums: []int{7}, want: 7, wantHeadOwner: "base", wantBranch: "feat/x"},
		{name: "owner:branch targets a fork", arg: "fork:feat/x", nums: []int{8}, want: 8, wantHeadOwner: "fork", wantBranch: "feat/x"},
		{name: "omitted uses current branch", arg: "", current: "feat/y", nums: []int{9}, want: 9, wantHeadOwner: "base", wantBranch: "feat/y"},
		{name: "omitted on detached HEAD", arg: "", current: "", wantErr: "--pr is required"},
		{name: "no open PR names the head and the fork form", arg: "feat/x", nums: nil, wantErr: "no open PR for base:feat/x"},
		{name: "two open PRs is ambiguous", arg: "feat/x", nums: []int{3, 5}, wantErr: "#3, #5"},
		{name: "zero is not a PR", arg: "0", wantErr: "invalid --pr"},
		{name: "negative is not a PR", arg: "-4", wantErr: "invalid --pr"},
		{name: "empty owner", arg: ":feat/x", wantErr: "invalid --pr"},
		{name: "empty branch", arg: "fork:", wantErr: "invalid --pr"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeFinder{nums: tt.nums}
			branchFn := func(context.Context) (string, error) {
				if tt.current == "" {
					return "", errors.New("detached")
				}
				return tt.current, nil
			}
			got, err := resolvePR(context.Background(), f, branchFn, "base", "repo", tt.arg)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolvePR: %v", err)
			}
			if got != tt.want {
				t.Errorf("pr = %d, want %d", got, tt.want)
			}
			if f.gotHeadOwner != tt.wantHeadOwner || f.gotBranch != tt.wantBranch {
				t.Errorf("lookup head = %q:%q, want %q:%q", f.gotHeadOwner, f.gotBranch, tt.wantHeadOwner, tt.wantBranch)
			}
		})
	}
}
