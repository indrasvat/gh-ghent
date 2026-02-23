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
