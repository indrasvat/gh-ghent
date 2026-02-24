package formatter

import (
	"bytes"
	"strings"
	"testing"
)

func TestMarkdownFormatterStructure(t *testing.T) {
	var buf bytes.Buffer
	f := &MarkdownFormatter{}

	if err := f.FormatComments(&buf, sampleCommentsResult()); err != nil {
		t.Fatalf("FormatComments: %v", err)
	}

	out := buf.String()

	checks := []struct {
		name string
		want string
	}{
		{"PR header", "# PR #42"},
		{"unresolved count", "**Unresolved:** 2"},
		{"resolved count", "**Resolved:** 1"},
		{"total count", "**Total:** 3"},
		{"file path", "main.go:10"},
		{"author", "@alice"},
		{"comment body", "Please fix this"},
		{"diff hunk", "```diff"},
	}

	for _, tc := range checks {
		if !strings.Contains(out, tc.want) {
			t.Errorf("%s: output missing %q\noutput:\n%s", tc.name, tc.want, out)
		}
	}
}

func TestMarkdownFormatterNoANSI(t *testing.T) {
	var buf bytes.Buffer
	f := &MarkdownFormatter{}

	if err := f.FormatComments(&buf, sampleCommentsResult()); err != nil {
		t.Fatalf("FormatComments: %v", err)
	}

	if strings.Contains(buf.String(), "\033") {
		t.Error("Markdown output contains ANSI escape sequences")
	}
}

func TestMarkdownResolveResultsStructure(t *testing.T) {
	var buf bytes.Buffer
	f := &MarkdownFormatter{}

	if err := f.FormatResolveResults(&buf, sampleResolveResults()); err != nil {
		t.Fatalf("FormatResolveResults: %v", err)
	}

	out := buf.String()

	checks := []struct {
		name string
		want string
	}{
		{"header", "# Resolve Results"},
		{"success count", "**Success:** 2"},
		{"failed count", "**Failed:** 1"},
		{"table header", "| Thread | File | Line | Action |"},
		{"thread ID", "PRRT_1"},
		{"file path", "main.go"},
		{"action", "resolved"},
		{"error section", "## Errors"},
		{"error thread", "PRRT_3"},
		{"error message", "permission denied"},
	}

	for _, tc := range checks {
		if !strings.Contains(out, tc.want) {
			t.Errorf("%s: output missing %q\noutput:\n%s", tc.name, tc.want, out)
		}
	}
}

func TestMarkdownResolveResultsNoANSI(t *testing.T) {
	var buf bytes.Buffer
	f := &MarkdownFormatter{}

	if err := f.FormatResolveResults(&buf, sampleResolveResults()); err != nil {
		t.Fatalf("FormatResolveResults: %v", err)
	}

	if strings.Contains(buf.String(), "\033") {
		t.Error("Markdown resolve output contains ANSI escape sequences")
	}
}

func TestMarkdownSummaryStructure(t *testing.T) {
	var buf bytes.Buffer
	f := &MarkdownFormatter{}

	if err := f.FormatSummary(&buf, sampleSummaryResult()); err != nil {
		t.Fatalf("FormatSummary: %v", err)
	}

	out := buf.String()

	checks := []struct {
		name string
		want string
	}{
		{"PR header with status", "# PR #42 — Summary [NOT READY]"},
		{"comments section", "## Review Comments"},
		{"unresolved count", "**Unresolved:** 2"},
		{"resolved count", "**Resolved:** 1"},
		{"total count", "**Total:** 3"},
		{"checks section", "## CI Checks"},
		{"checks status", "**Status:** pass"},
		{"pass count", "**Pass:** 3"},
		{"fail count", "**Fail:** 0"},
		{"pending count", "**Pending:** 0"},
		{"approvals section", "## Approvals"},
		{"reviewer table", "| Reviewer | State |"},
		{"alice approved", "| @alice | APPROVED |"},
		{"bob commented", "| @bob | COMMENTED |"},
	}

	for _, tc := range checks {
		if !strings.Contains(out, tc.want) {
			t.Errorf("%s: output missing %q\noutput:\n%s", tc.name, tc.want, out)
		}
	}
}

func TestMarkdownSummaryMergeReady(t *testing.T) {
	var buf bytes.Buffer
	f := &MarkdownFormatter{}

	result := sampleSummaryResult()
	result.IsMergeReady = true

	if err := f.FormatSummary(&buf, result); err != nil {
		t.Fatalf("FormatSummary: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "[READY]") {
		t.Errorf("merge-ready summary missing [READY]\noutput:\n%s", out)
	}
}

func TestMarkdownSummaryNoReviews(t *testing.T) {
	var buf bytes.Buffer
	f := &MarkdownFormatter{}

	result := sampleSummaryResult()
	result.Reviews = nil

	if err := f.FormatSummary(&buf, result); err != nil {
		t.Fatalf("FormatSummary: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "No reviews yet.") {
		t.Errorf("no-reviews summary missing 'No reviews yet.'\noutput:\n%s", out)
	}
}

func TestMarkdownSummaryWithFailedChecks(t *testing.T) {
	var buf bytes.Buffer
	f := &MarkdownFormatter{}

	if err := f.FormatSummary(&buf, sampleSummaryWithFailures()); err != nil {
		t.Fatalf("FormatSummary: %v", err)
	}

	out := buf.String()

	checks := []struct {
		name string
		want string
	}{
		{"fail header", "### FAIL: lint-check"},
		{"annotation level", "**failure**"},
		{"annotation location", "`src/main.go:42`"},
		{"annotation message", "unused variable: x"},
		{"log excerpt fence", "```"},
		{"log excerpt content", "src/main.go:42:5: x declared and not used"},
		{"thread detail", "**main.go:10** @alice"},
	}

	for _, tc := range checks {
		if !strings.Contains(out, tc.want) {
			t.Errorf("%s: output missing %q\noutput:\n%s", tc.name, tc.want, out)
		}
	}
}

func TestMarkdownSummaryNoFailedChecksWhenAllPass(t *testing.T) {
	var buf bytes.Buffer
	f := &MarkdownFormatter{}

	if err := f.FormatSummary(&buf, sampleSummaryResult()); err != nil {
		t.Fatalf("FormatSummary: %v", err)
	}

	out := buf.String()
	if strings.Contains(out, "### FAIL:") {
		t.Errorf("all-pass summary should not contain FAIL sections\noutput:\n%s", out)
	}
}

func TestMarkdownCompactSummaryWithFailedChecks(t *testing.T) {
	var buf bytes.Buffer
	f := &MarkdownFormatter{}

	if err := f.FormatCompactSummary(&buf, sampleSummaryWithFailures()); err != nil {
		t.Fatalf("FormatCompactSummary: %v", err)
	}

	out := buf.String()

	checks := []struct {
		name string
		want string
	}{
		{"fail name", "FAIL: lint-check"},
		{"annotation location", "src/main.go:42"},
		{"annotation message", "unused variable: x"},
	}

	for _, tc := range checks {
		if !strings.Contains(out, tc.want) {
			t.Errorf("%s: output missing %q\noutput:\n%s", tc.name, tc.want, out)
		}
	}
}

func TestMarkdownFormatterEmpty(t *testing.T) {
	var buf bytes.Buffer
	f := &MarkdownFormatter{}

	result := sampleCommentsResult()
	result.Threads = nil
	result.UnresolvedCount = 0
	result.ResolvedCount = 0
	result.TotalCount = 0

	if err := f.FormatComments(&buf, result); err != nil {
		t.Fatalf("FormatComments: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "# PR #42") {
		t.Error("empty result should still have PR header")
	}
	if strings.Contains(out, "---") {
		t.Error("empty result should not have thread separators")
	}
}
