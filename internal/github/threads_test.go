package github

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/indrasvat/ghent/internal/domain"
)

func mustParseTime(t *testing.T, s string) time.Time {
	t.Helper()
	v, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatalf("parse time %q: %v", s, err)
	}
	return v
}

func loadFixture(t *testing.T, path string) threadsResponse {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", path, err)
	}
	// The fixture has a "data" envelope matching the GraphQL response format.
	// We unwrap it to get the threadsResponse.
	var envelope struct {
		Data threadsResponse `json:"data"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		t.Fatalf("unmarshal fixture %s: %v", path, err)
	}
	return envelope.Data
}

func TestMapThreadsToDomain(t *testing.T) {
	resp := loadFixture(t, "../../testdata/graphql/review_threads.json")
	nodes := resp.Repository.PullRequest.ReviewThreads.Nodes
	totalCount := resp.Repository.PullRequest.ReviewThreads.TotalCount

	result, err := mapThreadsToResult(42, totalCount, nodes)
	if err != nil {
		t.Fatalf("mapThreadsToResult: %v", err)
	}

	if result.PRNumber != 42 {
		t.Errorf("PRNumber = %d, want 42", result.PRNumber)
	}
	if result.TotalCount != 3 {
		t.Errorf("TotalCount = %d, want 3", result.TotalCount)
	}
	if result.ResolvedCount != 1 {
		t.Errorf("ResolvedCount = %d, want 1", result.ResolvedCount)
	}
	if result.UnresolvedCount != 2 {
		t.Errorf("UnresolvedCount = %d, want 2", result.UnresolvedCount)
	}
	if len(result.Threads) != 2 {
		t.Fatalf("len(Threads) = %d, want 2 (unresolved only)", len(result.Threads))
	}

	// First unresolved thread
	got := result.Threads[0]
	want := domain.ReviewThread{
		ID:               "PRRT_thread1",
		Path:             "internal/api/graphql.go",
		Line:             47,
		StartLine:        45,
		IsResolved:       false,
		IsOutdated:       false,
		ViewerCanResolve: true,
		ViewerCanReply:   true,
		Comments: []domain.Comment{
			{
				ID:         "PRRC_comment1",
				DatabaseID: 1001,
				Author:     "reviewer1",
				Body:       "This should handle the nil case",
				CreatedAt:  mustParseTime(t, "2026-02-20T10:00:00Z"),
				URL:        "https://github.com/owner/repo/pull/42#discussion_r1001",
				DiffHunk:   "@@ -40,6 +40,8 @@\n func fetchData() {\n+\treturn nil\n }",
				Path:       "internal/api/graphql.go",
			},
		},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("thread[0] mismatch (-want +got):\n%s", diff)
	}
}

func TestFilterUnresolved(t *testing.T) {
	resp := loadFixture(t, "../../testdata/graphql/review_threads.json")
	nodes := resp.Repository.PullRequest.ReviewThreads.Nodes

	result, err := mapThreadsToResult(42, 3, nodes)
	if err != nil {
		t.Fatalf("mapThreadsToResult: %v", err)
	}

	// The resolved thread (thread2) should be excluded from Threads
	for _, thread := range result.Threads {
		if thread.IsResolved {
			t.Errorf("found resolved thread %s in result", thread.ID)
		}
	}

	// Verify the IDs of included threads
	wantIDs := []string{"PRRT_thread1", "PRRT_thread3"}
	var gotIDs []string
	for _, thread := range result.Threads {
		gotIDs = append(gotIDs, thread.ID)
	}
	if diff := cmp.Diff(wantIDs, gotIDs); diff != "" {
		t.Errorf("thread IDs mismatch (-want +got):\n%s", diff)
	}
}

func TestPaginationMerge(t *testing.T) {
	page1 := loadFixture(t, "../../testdata/graphql/review_threads.json")
	page2 := loadFixture(t, "../../testdata/graphql/review_threads_page2.json")

	// Merge nodes from both pages
	allNodes := make([]threadNode, 0,
		len(page1.Repository.PullRequest.ReviewThreads.Nodes)+
			len(page2.Repository.PullRequest.ReviewThreads.Nodes))
	allNodes = append(allNodes, page1.Repository.PullRequest.ReviewThreads.Nodes...)
	allNodes = append(allNodes, page2.Repository.PullRequest.ReviewThreads.Nodes...)

	result, err := mapThreadsToResult(42, 4, allNodes)
	if err != nil {
		t.Fatalf("mapThreadsToResult: %v", err)
	}

	if result.TotalCount != 4 {
		t.Errorf("TotalCount = %d, want 4", result.TotalCount)
	}
	// 3 unresolved (thread1, thread3 from page1 + thread4 from page2), 1 resolved
	if result.UnresolvedCount != 3 {
		t.Errorf("UnresolvedCount = %d, want 3", result.UnresolvedCount)
	}
	if result.ResolvedCount != 1 {
		t.Errorf("ResolvedCount = %d, want 1", result.ResolvedCount)
	}
	if len(result.Threads) != 3 {
		t.Fatalf("len(Threads) = %d, want 3", len(result.Threads))
	}
}

func TestMultipleCommentsInThread(t *testing.T) {
	resp := loadFixture(t, "../../testdata/graphql/review_threads.json")
	nodes := resp.Repository.PullRequest.ReviewThreads.Nodes

	result, err := mapThreadsToResult(42, 3, nodes)
	if err != nil {
		t.Fatalf("mapThreadsToResult: %v", err)
	}

	// thread3 has 2 comments
	thread3 := result.Threads[1]
	if thread3.ID != "PRRT_thread3" {
		t.Fatalf("expected thread3, got %s", thread3.ID)
	}
	if len(thread3.Comments) != 2 {
		t.Fatalf("len(Comments) = %d, want 2", len(thread3.Comments))
	}
	if thread3.Comments[0].Author != "reviewer1" {
		t.Errorf("comment[0].Author = %q, want %q", thread3.Comments[0].Author, "reviewer1")
	}
	if thread3.Comments[1].Author != "author1" {
		t.Errorf("comment[1].Author = %q, want %q", thread3.Comments[1].Author, "author1")
	}
}

func TestEmptyThreads(t *testing.T) {
	result, err := mapThreadsToResult(42, 0, nil)
	if err != nil {
		t.Fatalf("mapThreadsToResult: %v", err)
	}
	if result.TotalCount != 0 {
		t.Errorf("TotalCount = %d, want 0", result.TotalCount)
	}
	if result.UnresolvedCount != 0 {
		t.Errorf("UnresolvedCount = %d, want 0", result.UnresolvedCount)
	}
	if len(result.Threads) != 0 {
		t.Errorf("len(Threads) = %d, want 0", len(result.Threads))
	}
}
