package formatter

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/indrasvat/ghent/internal/domain"
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
