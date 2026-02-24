package formatter

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"strings"
	"testing"
	"time"

	"github.com/indrasvat/gh-ghent/internal/domain"
)

func sampleGroupedResult() *domain.GroupedCommentsResult {
	return &domain.GroupedCommentsResult{
		PRNumber: 42,
		GroupBy:  "file",
		Groups: []domain.CommentGroup{
			{
				Key: "cmd/main.go",
				Threads: []domain.ReviewThread{
					{
						ID:   "PRRT_1",
						Path: "cmd/main.go",
						Line: 10,
						Comments: []domain.Comment{
							{
								ID:        "C1",
								Author:    "alice",
								Body:      "fix this",
								CreatedAt: time.Date(2026, 2, 20, 10, 0, 0, 0, time.UTC),
								URL:       "https://github.com/o/r/pull/42#discussion_r1",
							},
						},
					},
				},
			},
			{
				Key: "internal/app.go",
				Threads: []domain.ReviewThread{
					{
						ID:   "PRRT_2",
						Path: "internal/app.go",
						Line: 20,
						Comments: []domain.Comment{
							{
								ID:        "C2",
								Author:    "bob",
								Body:      "change that",
								CreatedAt: time.Date(2026, 2, 20, 11, 0, 0, 0, time.UTC),
								URL:       "https://github.com/o/r/pull/42#discussion_r2",
							},
						},
					},
				},
			},
		},
		TotalCount:      3,
		ResolvedCount:   1,
		UnresolvedCount: 2,
	}
}

func TestJSONGroupedCommentsValid(t *testing.T) {
	var buf bytes.Buffer
	f := &JSONFormatter{}

	if err := f.FormatGroupedComments(&buf, sampleGroupedResult()); err != nil {
		t.Fatalf("FormatGroupedComments: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v\noutput:\n%s", err, buf.String())
	}
}

func TestJSONGroupedCommentsFields(t *testing.T) {
	var buf bytes.Buffer
	f := &JSONFormatter{}

	if err := f.FormatGroupedComments(&buf, sampleGroupedResult()); err != nil {
		t.Fatalf("FormatGroupedComments: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if v, ok := parsed["pr_number"].(float64); !ok || int(v) != 42 {
		t.Errorf("pr_number = %v, want 42", parsed["pr_number"])
	}
	if parsed["group_by"] != "file" {
		t.Errorf("group_by = %v, want file", parsed["group_by"])
	}
	if v, ok := parsed["unresolved_count"].(float64); !ok || int(v) != 2 {
		t.Errorf("unresolved_count = %v, want 2", parsed["unresolved_count"])
	}

	groups, ok := parsed["groups"].([]any)
	if !ok || len(groups) != 2 {
		t.Fatalf("groups count = %d, want 2", len(groups))
	}

	first := groups[0].(map[string]any)
	if first["key"] != "cmd/main.go" {
		t.Errorf("groups[0].key = %v, want cmd/main.go", first["key"])
	}
	threads, ok := first["threads"].([]any)
	if !ok || len(threads) != 1 {
		t.Fatalf("groups[0].threads count = %d, want 1", len(threads))
	}
}

func TestJSONGroupedCommentsNoANSI(t *testing.T) {
	var buf bytes.Buffer
	f := &JSONFormatter{}

	if err := f.FormatGroupedComments(&buf, sampleGroupedResult()); err != nil {
		t.Fatalf("FormatGroupedComments: %v", err)
	}

	if strings.Contains(buf.String(), "\033") {
		t.Error("JSON grouped output contains ANSI escape sequences")
	}
}

func TestMarkdownGroupedComments(t *testing.T) {
	var buf bytes.Buffer
	f := &MarkdownFormatter{}

	if err := f.FormatGroupedComments(&buf, sampleGroupedResult()); err != nil {
		t.Fatalf("FormatGroupedComments: %v", err)
	}

	out := buf.String()

	checks := []string{
		"# PR #42 — Review Comments (by file)",
		"**Unresolved:** 2",
		"## cmd/main.go",
		"### cmd/main.go:10",
		"**@alice**",
		"> fix this",
		"## internal/app.go",
		"### internal/app.go:20",
		"**@bob**",
		"> change that",
	}
	for _, want := range checks {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q", want)
		}
	}
}

func TestXMLGroupedCommentsValid(t *testing.T) {
	var buf bytes.Buffer
	f := &XMLFormatter{}

	if err := f.FormatGroupedComments(&buf, sampleGroupedResult()); err != nil {
		t.Fatalf("FormatGroupedComments: %v", err)
	}

	// Must be valid XML
	var parsed any
	if err := xml.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("invalid XML: %v\noutput:\n%s", err, buf.String())
	}
}

func TestXMLGroupedCommentsStructure(t *testing.T) {
	var buf bytes.Buffer
	f := &XMLFormatter{}

	if err := f.FormatGroupedComments(&buf, sampleGroupedResult()); err != nil {
		t.Fatalf("FormatGroupedComments: %v", err)
	}

	out := buf.String()

	checks := []string{
		`<grouped_comments`,
		`group_by="file"`,
		`pr_number="42"`,
		`<group key="cmd/main.go"`,
		`<group key="internal/app.go"`,
		`<thread`,
	}
	for _, want := range checks {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q", want)
		}
	}
}
