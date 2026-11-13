package github

import (
	"context"
	"slices"
	"testing"
)

func TestFindOpenPRs(t *testing.T) {
	c, f := newFakeClient(t, map[string]fakeResponse{
		"repos/o/r/pulls?head=fork%3Afeat%2Fx&per_page=100&state=open": {200, `[{"number":7},{"number":9}]`},
		"repos/o/r/pulls?head=o%3Anone&per_page=100&state=open":        {200, `[]`},
	})
	got, err := c.FindOpenPRs(context.Background(), "o", "r", "fork", "feat/x")
	if err != nil {
		t.Fatalf("FindOpenPRs: %v", err)
	}
	if !slices.Equal(got, []int{7, 9}) {
		t.Errorf("got %v, want [7 9]", got)
	}
	got, err = c.FindOpenPRs(context.Background(), "o", "r", "o", "none")
	if err != nil || len(got) != 0 {
		t.Errorf("no match: got %v, %v; want empty, nil", got, err)
	}
	if len(f.calls) != 2 {
		t.Errorf("calls = %v, want 2", f.calls)
	}
}
