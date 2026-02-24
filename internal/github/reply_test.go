package github

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestReplyResponseParsing(t *testing.T) {
	data, err := os.ReadFile("../../testdata/rest/reply_comment.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	var resp replyResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}

	want := replyResponse{
		ID:        2001,
		NodeID:    "PRRC_reply1",
		Body:      "Acknowledged — will fix in next push",
		HTMLURL:   "https://github.com/owner/repo/pull/42#discussion_r2001",
		CreatedAt: "2026-02-22T14:00:00Z",
	}

	if diff := cmp.Diff(want, resp); diff != "" {
		t.Errorf("replyResponse mismatch (-want +got):\n%s", diff)
	}
}

func TestFindThreadByID_InFixture(t *testing.T) {
	resp := loadFixture(t, "../../testdata/graphql/review_threads.json")
	nodes := resp.Repository.PullRequest.ReviewThreads.Nodes

	tests := []struct {
		name     string
		threadID string
		wantErr  bool
		wantPath string
	}{
		{
			name:     "find unresolved thread",
			threadID: "PRRT_thread1",
			wantPath: "internal/api/graphql.go",
		},
		{
			name:     "find resolved thread",
			threadID: "PRRT_thread2",
			wantPath: "internal/config/config.go",
		},
		{
			name:     "find thread with multiple comments",
			threadID: "PRRT_thread3",
			wantPath: "cmd/main.go",
		},
		{
			name:     "thread not found",
			threadID: "PRRT_nonexistent",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var found *threadNode
			for i := range nodes {
				if nodes[i].ID == tt.threadID {
					found = &nodes[i]
					break
				}
			}

			if tt.wantErr {
				if found != nil {
					t.Errorf("expected thread %s not to be found", tt.threadID)
				}
				return
			}

			if found == nil {
				t.Fatalf("expected to find thread %s", tt.threadID)
			}
			if found.Path != tt.wantPath {
				t.Errorf("thread.Path = %q, want %q", found.Path, tt.wantPath)
			}
		})
	}
}

func TestViewerCanReplyCheck(t *testing.T) {
	resp := loadFixture(t, "../../testdata/graphql/review_threads.json")
	nodes := resp.Repository.PullRequest.ReviewThreads.Nodes

	// All threads in the fixture have viewerCanReply=true
	for _, n := range nodes {
		if !n.ViewerCanReply {
			t.Errorf("thread %s: ViewerCanReply = false, want true", n.ID)
		}
	}
}

func TestLastCommentDatabaseID(t *testing.T) {
	resp := loadFixture(t, "../../testdata/graphql/review_threads.json")
	nodes := resp.Repository.PullRequest.ReviewThreads.Nodes

	tests := []struct {
		threadID  string
		wantDBID  int64
		wantCount int
	}{
		{threadID: "PRRT_thread1", wantDBID: 1001, wantCount: 1},
		{threadID: "PRRT_thread3", wantDBID: 1004, wantCount: 2}, // last comment's databaseId
	}

	for _, tt := range tests {
		t.Run(tt.threadID, func(t *testing.T) {
			var found *threadNode
			for i := range nodes {
				if nodes[i].ID == tt.threadID {
					found = &nodes[i]
					break
				}
			}
			if found == nil {
				t.Fatalf("thread %s not found", tt.threadID)
			}
			if len(found.Comments.Nodes) != tt.wantCount {
				t.Fatalf("comment count = %d, want %d", len(found.Comments.Nodes), tt.wantCount)
			}
			lastComment := found.Comments.Nodes[len(found.Comments.Nodes)-1]
			if lastComment.DatabaseID != tt.wantDBID {
				t.Errorf("last comment databaseId = %d, want %d", lastComment.DatabaseID, tt.wantDBID)
			}
		})
	}
}
