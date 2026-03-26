package domain

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestReviewThread_IsBotOriginated(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		comments []Comment
		want     bool
	}{
		{
			name:     "empty comments",
			comments: nil,
			want:     false,
		},
		{
			name:     "first comment is bot",
			comments: []Comment{{Author: "coderabbitai", IsBot: true}},
			want:     true,
		},
		{
			name:     "first comment is human",
			comments: []Comment{{Author: "alice", IsBot: false}},
			want:     false,
		},
		{
			name: "bot first then human reply",
			comments: []Comment{
				{Author: "coderabbitai", IsBot: true},
				{Author: "alice", IsBot: false},
			},
			want: true,
		},
		{
			name: "human first then bot reply",
			comments: []Comment{
				{Author: "alice", IsBot: false},
				{Author: "coderabbitai", IsBot: true},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			thread := ReviewThread{Comments: tt.comments}
			if got := thread.IsBotOriginated(); got != tt.want {
				t.Errorf("IsBotOriginated() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReviewThread_IsUnanswered(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		comments []Comment
		want     bool
	}{
		{"zero comments", nil, true},
		{"one comment", []Comment{{Author: "bot"}}, true},
		{"two comments", []Comment{{Author: "bot"}, {Author: "human"}}, false},
		{"three comments", []Comment{{}, {}, {}}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			thread := ReviewThread{Comments: tt.comments}
			if got := thread.IsUnanswered(); got != tt.want {
				t.Errorf("IsUnanswered() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReplyResult_ResolvedField_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	result := ReplyResult{
		ThreadID:  "PRRT_abc",
		CommentID: 42,
		URL:       "https://github.com/test",
		Body:      "Fixed",
		CreatedAt: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		Resolved: &ResolveResult{
			ThreadID:   "PRRT_abc",
			Path:       "main.go",
			Line:       10,
			IsResolved: true,
			Action:     "resolved",
		},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got ReplyResult
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.Resolved == nil {
		t.Fatal("Resolved is nil after round trip")
	}
	if got.Resolved.Action != "resolved" {
		t.Errorf("Resolved.Action = %q, want %q", got.Resolved.Action, "resolved")
	}
	if got.Resolved.Path != "main.go" {
		t.Errorf("Resolved.Path = %q, want %q", got.Resolved.Path, "main.go")
	}
}

func TestReplyResult_ResolvedOmittedWhenNil(t *testing.T) {
	t.Parallel()

	result := ReplyResult{
		ThreadID:  "PRRT_abc",
		CommentID: 42,
		URL:       "https://github.com/test",
		Body:      "Fixed",
		CreatedAt: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if _, ok := m["resolved"]; ok {
		t.Error("'resolved' field should be omitted when nil")
	}
	if _, ok := m["resolve_error"]; ok {
		t.Error("'resolve_error' field should be omitted when empty")
	}
}

func TestReplyResult_ResolveErrorInJSON(t *testing.T) {
	t.Parallel()

	result := ReplyResult{
		ThreadID:     "PRRT_abc",
		CommentID:    42,
		URL:          "https://github.com/test",
		Body:         "Fixed",
		CreatedAt:    time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		ResolveError: "permission denied: cannot resolve",
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	errVal, ok := m["resolve_error"]
	if !ok {
		t.Fatal("'resolve_error' field missing from JSON output")
	}
	if errVal != "permission denied: cannot resolve" {
		t.Errorf("resolve_error = %v, want %q", errVal, "permission denied: cannot resolve")
	}
}

func TestReplyResult_AlreadyResolved(t *testing.T) {
	t.Parallel()

	result := ReplyResult{
		ThreadID:  "PRRT_abc",
		CommentID: 42,
		URL:       "https://github.com/test",
		Body:      "Done",
		CreatedAt: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		Resolved: &ResolveResult{
			ThreadID:   "PRRT_abc",
			IsResolved: true,
			Action:     "already_resolved",
		},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got ReplyResult
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.Resolved == nil {
		t.Fatal("Resolved is nil after round trip")
	}
	if got.Resolved.Action != "already_resolved" {
		t.Errorf("Action = %q, want %q", got.Resolved.Action, "already_resolved")
	}
	if !got.Resolved.IsResolved {
		t.Error("IsResolved should be true for already_resolved")
	}
	if got.ResolveError != "" {
		t.Errorf("ResolveError should be empty for already_resolved, got %q", got.ResolveError)
	}
}

func TestComment_IsBotField_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	c := Comment{
		ID:        "C_abc",
		Author:    "coderabbitai",
		IsBot:     true,
		Body:      "Finding",
		CreatedAt: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
	}

	data, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got Comment
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if !got.IsBot {
		t.Error("IsBot should be true after round trip")
	}

	// Verify the field name in JSON.
	var m map[string]any
	_ = json.Unmarshal(data, &m)
	if _, ok := m["is_bot"]; !ok {
		t.Error("JSON should contain 'is_bot' field")
	}
}

func TestCommentsResult_BotCounters_JSON(t *testing.T) {
	t.Parallel()

	result := CommentsResult{
		PRNumber:        1,
		TotalCount:      4,
		ResolvedCount:   1,
		UnresolvedCount: 3,
		BotThreadCount:  2,
		UnansweredCount: 1,
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got CommentsResult
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if diff := cmp.Diff(result.BotThreadCount, got.BotThreadCount); diff != "" {
		t.Errorf("BotThreadCount mismatch: %s", diff)
	}
	if diff := cmp.Diff(result.UnansweredCount, got.UnansweredCount); diff != "" {
		t.Errorf("UnansweredCount mismatch: %s", diff)
	}
}
