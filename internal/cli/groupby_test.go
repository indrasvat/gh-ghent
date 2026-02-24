package cli

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/indrasvat/gh-ghent/internal/domain"
)

func sampleGroupByResult() *domain.CommentsResult {
	return &domain.CommentsResult{
		PRNumber: 42,
		Threads: []domain.ReviewThread{
			{
				ID:         "PRRT_1",
				Path:       "cmd/main.go",
				Line:       10,
				IsResolved: false,
				Comments: []domain.Comment{
					{ID: "C1", Author: "alice", Body: "fix this", CreatedAt: time.Date(2026, 2, 20, 10, 0, 0, 0, time.UTC)},
				},
			},
			{
				ID:         "PRRT_2",
				Path:       "internal/app.go",
				Line:       20,
				IsResolved: false,
				Comments: []domain.Comment{
					{ID: "C2", Author: "bob", Body: "change that", CreatedAt: time.Date(2026, 2, 20, 11, 0, 0, 0, time.UTC)},
				},
			},
			{
				ID:         "PRRT_3",
				Path:       "cmd/main.go",
				Line:       30,
				IsResolved: true,
				Comments: []domain.Comment{
					{ID: "C3", Author: "alice", Body: "done", CreatedAt: time.Date(2026, 2, 20, 12, 0, 0, 0, time.UTC)},
				},
			},
		},
		TotalCount:      3,
		ResolvedCount:   1,
		UnresolvedCount: 2,
	}
}

func TestGroupThreadsByFile(t *testing.T) {
	result := sampleGroupByResult()
	grouped, err := groupThreads(result, "file")
	if err != nil {
		t.Fatalf("groupThreads: %v", err)
	}

	if grouped.GroupBy != "file" {
		t.Errorf("GroupBy = %q, want %q", grouped.GroupBy, "file")
	}
	if grouped.PRNumber != 42 {
		t.Errorf("PRNumber = %d, want 42", grouped.PRNumber)
	}
	if grouped.TotalCount != 3 {
		t.Errorf("TotalCount = %d, want 3", grouped.TotalCount)
	}
	if len(grouped.Groups) != 2 {
		t.Fatalf("len(Groups) = %d, want 2", len(grouped.Groups))
	}

	// Alphabetical: cmd/main.go before internal/app.go
	if grouped.Groups[0].Key != "cmd/main.go" {
		t.Errorf("Groups[0].Key = %q, want %q", grouped.Groups[0].Key, "cmd/main.go")
	}
	if len(grouped.Groups[0].Threads) != 2 {
		t.Errorf("Groups[0] thread count = %d, want 2", len(grouped.Groups[0].Threads))
	}
	if grouped.Groups[1].Key != "internal/app.go" {
		t.Errorf("Groups[1].Key = %q, want %q", grouped.Groups[1].Key, "internal/app.go")
	}
	if len(grouped.Groups[1].Threads) != 1 {
		t.Errorf("Groups[1] thread count = %d, want 1", len(grouped.Groups[1].Threads))
	}
}

func TestGroupThreadsByAuthor(t *testing.T) {
	result := sampleGroupByResult()
	grouped, err := groupThreads(result, "author")
	if err != nil {
		t.Fatalf("groupThreads: %v", err)
	}

	if grouped.GroupBy != "author" {
		t.Errorf("GroupBy = %q, want %q", grouped.GroupBy, "author")
	}
	if len(grouped.Groups) != 2 {
		t.Fatalf("len(Groups) = %d, want 2", len(grouped.Groups))
	}

	// Alphabetical: alice before bob
	if grouped.Groups[0].Key != "alice" {
		t.Errorf("Groups[0].Key = %q, want %q", grouped.Groups[0].Key, "alice")
	}
	if len(grouped.Groups[0].Threads) != 2 {
		t.Errorf("Groups[0] thread count = %d, want 2", len(grouped.Groups[0].Threads))
	}
	if grouped.Groups[1].Key != "bob" {
		t.Errorf("Groups[1].Key = %q, want %q", grouped.Groups[1].Key, "bob")
	}
	if len(grouped.Groups[1].Threads) != 1 {
		t.Errorf("Groups[1] thread count = %d, want 1", len(grouped.Groups[1].Threads))
	}
}

func TestGroupThreadsByStatus(t *testing.T) {
	result := sampleGroupByResult()
	grouped, err := groupThreads(result, "status")
	if err != nil {
		t.Fatalf("groupThreads: %v", err)
	}

	if grouped.GroupBy != "status" {
		t.Errorf("GroupBy = %q, want %q", grouped.GroupBy, "status")
	}
	if len(grouped.Groups) != 2 {
		t.Fatalf("len(Groups) = %d, want 2", len(grouped.Groups))
	}

	// Unresolved first
	if grouped.Groups[0].Key != "unresolved" {
		t.Errorf("Groups[0].Key = %q, want %q", grouped.Groups[0].Key, "unresolved")
	}
	if len(grouped.Groups[0].Threads) != 2 {
		t.Errorf("Groups[0] thread count = %d, want 2", len(grouped.Groups[0].Threads))
	}
	if grouped.Groups[1].Key != "resolved" {
		t.Errorf("Groups[1].Key = %q, want %q", grouped.Groups[1].Key, "resolved")
	}
	if len(grouped.Groups[1].Threads) != 1 {
		t.Errorf("Groups[1] thread count = %d, want 1", len(grouped.Groups[1].Threads))
	}
}

func TestGroupThreadsInvalidValue(t *testing.T) {
	result := sampleGroupByResult()
	_, err := groupThreads(result, "bogus")
	if err == nil {
		t.Fatal("expected error for invalid group-by value")
	}
}

func TestGroupThreadsEmpty(t *testing.T) {
	result := &domain.CommentsResult{
		PRNumber:        42,
		Threads:         nil,
		TotalCount:      0,
		ResolvedCount:   0,
		UnresolvedCount: 0,
	}
	grouped, err := groupThreads(result, "file")
	if err != nil {
		t.Fatalf("groupThreads: %v", err)
	}
	if len(grouped.Groups) != 0 {
		t.Errorf("len(Groups) = %d, want 0", len(grouped.Groups))
	}
}

func TestGroupThreadsSingleFile(t *testing.T) {
	result := &domain.CommentsResult{
		PRNumber: 1,
		Threads: []domain.ReviewThread{
			{ID: "T1", Path: "main.go", Comments: []domain.Comment{{Author: "alice"}}},
			{ID: "T2", Path: "main.go", Comments: []domain.Comment{{Author: "bob"}}},
		},
		TotalCount:      2,
		UnresolvedCount: 2,
	}
	grouped, err := groupThreads(result, "file")
	if err != nil {
		t.Fatalf("groupThreads: %v", err)
	}
	if len(grouped.Groups) != 1 {
		t.Fatalf("len(Groups) = %d, want 1", len(grouped.Groups))
	}
	if grouped.Groups[0].Key != "main.go" {
		t.Errorf("key = %q, want main.go", grouped.Groups[0].Key)
	}
	if len(grouped.Groups[0].Threads) != 2 {
		t.Errorf("thread count = %d, want 2", len(grouped.Groups[0].Threads))
	}
}

func TestGroupThreadsPreservesCounts(t *testing.T) {
	result := sampleGroupByResult()
	for _, mode := range []string{"file", "author", "status"} {
		grouped, err := groupThreads(result, mode)
		if err != nil {
			t.Fatalf("groupThreads(%s): %v", mode, err)
		}
		if diff := cmp.Diff(result.TotalCount, grouped.TotalCount); diff != "" {
			t.Errorf("TotalCount mismatch for %s: %s", mode, diff)
		}
		if diff := cmp.Diff(result.ResolvedCount, grouped.ResolvedCount); diff != "" {
			t.Errorf("ResolvedCount mismatch for %s: %s", mode, diff)
		}
		if diff := cmp.Diff(result.UnresolvedCount, grouped.UnresolvedCount); diff != "" {
			t.Errorf("UnresolvedCount mismatch for %s: %s", mode, diff)
		}
	}
}

func TestGroupThreadsAuthorNoComments(t *testing.T) {
	result := &domain.CommentsResult{
		PRNumber: 1,
		Threads: []domain.ReviewThread{
			{ID: "T1", Path: "main.go", Comments: nil},
		},
		TotalCount:      1,
		UnresolvedCount: 1,
	}
	grouped, err := groupThreads(result, "author")
	if err != nil {
		t.Fatalf("groupThreads: %v", err)
	}
	if len(grouped.Groups) != 1 {
		t.Fatalf("len(Groups) = %d, want 1", len(grouped.Groups))
	}
	if grouped.Groups[0].Key != "unknown" {
		t.Errorf("key = %q, want unknown", grouped.Groups[0].Key)
	}
}
