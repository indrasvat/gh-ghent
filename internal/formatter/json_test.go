package formatter

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/indrasvat/gh-ghent/internal/domain"
)

func sampleCommentsResult() *domain.CommentsResult {
	return &domain.CommentsResult{
		PRNumber: 42,
		Threads: []domain.ReviewThread{
			{
				ID:               "PRRT_1",
				Path:             "main.go",
				Line:             10,
				IsResolved:       false,
				ViewerCanResolve: true,
				ViewerCanReply:   true,
				Comments: []domain.Comment{
					{
						ID:         "PRRC_1",
						DatabaseID: 100,
						Author:     "alice",
						Body:       "Please fix this",
						CreatedAt:  time.Date(2026, 2, 20, 10, 0, 0, 0, time.UTC),
						URL:        "https://github.com/o/r/pull/42#discussion_r100",
						DiffHunk:   "@@ -8,3 +8,5 @@",
						Path:       "main.go",
					},
				},
			},
		},
		TotalCount:      3,
		ResolvedCount:   1,
		UnresolvedCount: 2,
	}
}

func TestJSONFormatterValid(t *testing.T) {
	var buf bytes.Buffer
	f := &JSONFormatter{}

	if err := f.FormatComments(&buf, sampleCommentsResult()); err != nil {
		t.Fatalf("FormatComments: %v", err)
	}

	// Must be valid JSON
	var parsed map[string]any
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v\noutput:\n%s", err, buf.String())
	}
}

func TestJSONFormatterFields(t *testing.T) {
	var buf bytes.Buffer
	f := &JSONFormatter{}

	if err := f.FormatComments(&buf, sampleCommentsResult()); err != nil {
		t.Fatalf("FormatComments: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	// Check top-level fields
	if v, ok := parsed["pr_number"].(float64); !ok || int(v) != 42 {
		t.Errorf("pr_number = %v, want 42", parsed["pr_number"])
	}
	if v, ok := parsed["unresolved_count"].(float64); !ok || int(v) != 2 {
		t.Errorf("unresolved_count = %v, want 2", parsed["unresolved_count"])
	}

	threads, ok := parsed["threads"].([]any)
	if !ok || len(threads) != 1 {
		t.Fatalf("threads count = %d, want 1", len(threads))
	}

	thread := threads[0].(map[string]any)
	if thread["path"] != "main.go" {
		t.Errorf("thread.path = %v, want main.go", thread["path"])
	}
}

func sampleResolveResults() *domain.ResolveResults {
	return &domain.ResolveResults{
		Results: []domain.ResolveResult{
			{
				ThreadID:   "PRRT_1",
				Path:       "main.go",
				Line:       10,
				IsResolved: true,
				Action:     "resolved",
			},
			{
				ThreadID:   "PRRT_2",
				Path:       "config.go",
				Line:       25,
				IsResolved: true,
				Action:     "resolved",
			},
		},
		SuccessCount: 2,
		FailureCount: 1,
		Errors: []domain.ResolveError{
			{ThreadID: "PRRT_3", Message: "permission denied"},
		},
	}
}

func TestJSONResolveResultsValid(t *testing.T) {
	var buf bytes.Buffer
	f := &JSONFormatter{}

	if err := f.FormatResolveResults(&buf, sampleResolveResults()); err != nil {
		t.Fatalf("FormatResolveResults: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v\noutput:\n%s", err, buf.String())
	}
}

func TestJSONResolveResultsFields(t *testing.T) {
	var buf bytes.Buffer
	f := &JSONFormatter{}

	if err := f.FormatResolveResults(&buf, sampleResolveResults()); err != nil {
		t.Fatalf("FormatResolveResults: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if v, ok := parsed["success_count"].(float64); !ok || int(v) != 2 {
		t.Errorf("success_count = %v, want 2", parsed["success_count"])
	}
	if v, ok := parsed["failure_count"].(float64); !ok || int(v) != 1 {
		t.Errorf("failure_count = %v, want 1", parsed["failure_count"])
	}

	results, ok := parsed["results"].([]any)
	if !ok || len(results) != 2 {
		t.Fatalf("results count = %d, want 2", len(results))
	}

	first := results[0].(map[string]any)
	if first["thread_id"] != "PRRT_1" {
		t.Errorf("result[0].thread_id = %v, want PRRT_1", first["thread_id"])
	}
	if first["action"] != "resolved" {
		t.Errorf("result[0].action = %v, want resolved", first["action"])
	}

	errors, ok := parsed["errors"].([]any)
	if !ok || len(errors) != 1 {
		t.Fatalf("errors count = %d, want 1", len(errors))
	}
}

func TestJSONFormatterNoANSI(t *testing.T) {
	var buf bytes.Buffer
	f := &JSONFormatter{}

	if err := f.FormatComments(&buf, sampleCommentsResult()); err != nil {
		t.Fatalf("FormatComments: %v", err)
	}

	if strings.Contains(buf.String(), "\033") {
		t.Error("JSON output contains ANSI escape sequences")
	}
}

func sampleSummaryResult() *domain.SummaryResult {
	return &domain.SummaryResult{
		PRNumber: 42,
		Comments: domain.CommentsResult{
			PRNumber:        42,
			TotalCount:      3,
			ResolvedCount:   1,
			UnresolvedCount: 2,
			Threads: []domain.ReviewThread{
				{
					ID:   "PRRT_1",
					Path: "main.go",
					Line: 10,
					Comments: []domain.Comment{
						{
							ID:        "PRRC_1",
							Author:    "alice",
							Body:      "Please fix this",
							CreatedAt: time.Date(2026, 2, 20, 10, 0, 0, 0, time.UTC),
						},
					},
				},
			},
		},
		Checks: domain.ChecksResult{
			PRNumber:      42,
			HeadSHA:       "abc123",
			OverallStatus: domain.StatusPass,
			PassCount:     3,
			FailCount:     0,
			PendingCount:  0,
		},
		Reviews: []domain.Review{
			{
				ID:          "PRR_1",
				Author:      "alice",
				State:       domain.ReviewApproved,
				Body:        "LGTM",
				SubmittedAt: time.Date(2026, 2, 20, 12, 0, 0, 0, time.UTC),
			},
			{
				ID:          "PRR_2",
				Author:      "bob",
				State:       domain.ReviewCommented,
				Body:        "Minor nit",
				SubmittedAt: time.Date(2026, 2, 20, 13, 0, 0, 0, time.UTC),
			},
		},
		IsMergeReady: false,
		PRAge:        "3d",
		LastUpdate:   "5h",
		ReviewCycles: 1,
	}
}

func TestJSONSummaryValid(t *testing.T) {
	var buf bytes.Buffer
	f := &JSONFormatter{}

	if err := f.FormatSummary(&buf, sampleSummaryResult()); err != nil {
		t.Fatalf("FormatSummary: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v\noutput:\n%s", err, buf.String())
	}
}

func TestJSONSummaryFields(t *testing.T) {
	var buf bytes.Buffer
	f := &JSONFormatter{}

	if err := f.FormatSummary(&buf, sampleSummaryResult()); err != nil {
		t.Fatalf("FormatSummary: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if v, ok := parsed["pr_number"].(float64); !ok || int(v) != 42 {
		t.Errorf("pr_number = %v, want 42", parsed["pr_number"])
	}
	if v, ok := parsed["is_merge_ready"].(bool); !ok || v != false {
		t.Errorf("is_merge_ready = %v, want false", parsed["is_merge_ready"])
	}

	comments, ok := parsed["comments"].(map[string]any)
	if !ok {
		t.Fatal("missing comments section")
	}
	if v, ok := comments["unresolved_count"].(float64); !ok || int(v) != 2 {
		t.Errorf("comments.unresolved_count = %v, want 2", comments["unresolved_count"])
	}

	checks, ok := parsed["checks"].(map[string]any)
	if !ok {
		t.Fatal("missing checks section")
	}
	if checks["overall_status"] != "pass" {
		t.Errorf("checks.overall_status = %v, want pass", checks["overall_status"])
	}

	reviews, ok := parsed["reviews"].([]any)
	if !ok || len(reviews) != 2 {
		t.Fatalf("reviews count = %v, want 2", len(reviews))
	}
	first := reviews[0].(map[string]any)
	if first["author"] != "alice" {
		t.Errorf("reviews[0].author = %v, want alice", first["author"])
	}
	if first["state"] != "APPROVED" {
		t.Errorf("reviews[0].state = %v, want APPROVED", first["state"])
	}
}

func TestJSONSummaryMergeReady(t *testing.T) {
	var buf bytes.Buffer
	f := &JSONFormatter{}

	result := sampleSummaryResult()
	result.IsMergeReady = true

	if err := f.FormatSummary(&buf, result); err != nil {
		t.Fatalf("FormatSummary: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if v, ok := parsed["is_merge_ready"].(bool); !ok || !v {
		t.Errorf("is_merge_ready = %v, want true", parsed["is_merge_ready"])
	}
}

func TestJSONCompactSummaryValid(t *testing.T) {
	var buf bytes.Buffer
	f := &JSONFormatter{}

	if err := f.FormatCompactSummary(&buf, sampleSummaryResult()); err != nil {
		t.Fatalf("FormatCompactSummary: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v\noutput:\n%s", err, buf.String())
	}
}

func TestJSONCompactSummaryFlat(t *testing.T) {
	var buf bytes.Buffer
	f := &JSONFormatter{}

	if err := f.FormatCompactSummary(&buf, sampleSummaryResult()); err != nil {
		t.Fatalf("FormatCompactSummary: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	// Compact format has flat fields, not nested comments/checks objects.
	if v, ok := parsed["pr_number"].(float64); !ok || int(v) != 42 {
		t.Errorf("pr_number = %v, want 42", parsed["pr_number"])
	}
	if v, ok := parsed["unresolved"].(float64); !ok || int(v) != 2 {
		t.Errorf("unresolved = %v, want 2", parsed["unresolved"])
	}
	if parsed["check_status"] != "pass" {
		t.Errorf("check_status = %v, want pass", parsed["check_status"])
	}
	if v, ok := parsed["pass_count"].(float64); !ok || int(v) != 3 {
		t.Errorf("pass_count = %v, want 3", parsed["pass_count"])
	}
	if parsed["pr_age"] != "3d" {
		t.Errorf("pr_age = %v, want 3d", parsed["pr_age"])
	}
	if parsed["last_update"] != "5h" {
		t.Errorf("last_update = %v, want 5h", parsed["last_update"])
	}
	if v, ok := parsed["review_cycles"].(float64); !ok || int(v) != 1 {
		t.Errorf("review_cycles = %v, want 1", parsed["review_cycles"])
	}

	// Should NOT have nested "comments" or "checks" objects.
	if _, ok := parsed["comments"]; ok {
		t.Error("compact format should not have nested 'comments' object")
	}
	if _, ok := parsed["checks"]; ok {
		t.Error("compact format should not have nested 'checks' object")
	}

	// Threads should be flat array with preview.
	threads, ok := parsed["threads"].([]any)
	if !ok || len(threads) != 1 {
		t.Fatalf("threads count = %v, want 1", len(threads))
	}
	thread := threads[0].(map[string]any)
	if thread["file"] != "main.go" {
		t.Errorf("thread.file = %v, want main.go", thread["file"])
	}
	if thread["author"] != "alice" {
		t.Errorf("thread.author = %v, want alice", thread["author"])
	}
	if thread["body_preview"] != "Please fix this" {
		t.Errorf("thread.body_preview = %v, want 'Please fix this'", thread["body_preview"])
	}
}

func TestJSONCompactSummaryShorterThanFull(t *testing.T) {
	f := &JSONFormatter{}
	result := sampleSummaryResult()

	var fullBuf, compactBuf bytes.Buffer
	if err := f.FormatSummary(&fullBuf, result); err != nil {
		t.Fatalf("FormatSummary: %v", err)
	}
	if err := f.FormatCompactSummary(&compactBuf, result); err != nil {
		t.Fatalf("FormatCompactSummary: %v", err)
	}

	if compactBuf.Len() >= fullBuf.Len() {
		t.Errorf("compact (%d bytes) should be shorter than full (%d bytes)", compactBuf.Len(), fullBuf.Len())
	}
}

func TestJSONCompactSummaryNoThreads(t *testing.T) {
	f := &JSONFormatter{}
	result := &domain.SummaryResult{
		PRNumber: 99,
		Comments: domain.CommentsResult{},
		Checks: domain.ChecksResult{
			OverallStatus: domain.StatusPass,
		},
	}

	var buf bytes.Buffer
	if err := f.FormatCompactSummary(&buf, result); err != nil {
		t.Fatalf("FormatCompactSummary: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	// threads should be null/absent when empty.
	if threads, ok := parsed["threads"]; ok && threads != nil {
		t.Errorf("threads should be null/absent for empty result, got %v", threads)
	}
}

func TestJSONCompactSummaryBodyTruncation(t *testing.T) {
	f := &JSONFormatter{}
	longBody := strings.Repeat("x", 100)
	result := &domain.SummaryResult{
		PRNumber: 1,
		Comments: domain.CommentsResult{
			Threads: []domain.ReviewThread{
				{
					Path: "a.go",
					Line: 1,
					Comments: []domain.Comment{
						{Author: "bob", Body: longBody},
					},
				},
			},
			UnresolvedCount: 1,
		},
		Checks: domain.ChecksResult{OverallStatus: domain.StatusFail},
	}

	var buf bytes.Buffer
	if err := f.FormatCompactSummary(&buf, result); err != nil {
		t.Fatalf("FormatCompactSummary: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	threads := parsed["threads"].([]any)
	thread := threads[0].(map[string]any)
	preview := thread["body_preview"].(string)
	if len(preview) > 84 { // 80 + "..."
		t.Errorf("body_preview not truncated: len=%d", len(preview))
	}
	if !strings.HasSuffix(preview, "...") {
		t.Errorf("body_preview should end with '...', got %q", preview)
	}
}
