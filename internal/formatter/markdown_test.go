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

func TestMarkdownStatusStructure(t *testing.T) {
	var buf bytes.Buffer
	f := &MarkdownFormatter{}

	if err := f.FormatStatus(&buf, sampleStatusResult()); err != nil {
		t.Fatalf("FormatStatus: %v", err)
	}

	out := buf.String()

	checks := []struct {
		name string
		want string
	}{
		{"PR header with status", "# PR #42 — Status [NOT READY]"},
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
		{"reviewer table", "| Reviewer | State | Commit |"},
		{"alice approved", "| @alice | APPROVED | - |"},
		{"bob commented", "| @bob | COMMENTED | - |"},
	}

	for _, tc := range checks {
		if !strings.Contains(out, tc.want) {
			t.Errorf("%s: output missing %q\noutput:\n%s", tc.name, tc.want, out)
		}
	}
}

func TestMarkdownStatusStaleReviewGuidance(t *testing.T) {
	var buf bytes.Buffer
	f := &MarkdownFormatter{}

	if err := f.FormatStatus(&buf, sampleStatusWithStaleReview()); err != nil {
		t.Fatalf("FormatStatus: %v", err)
	}

	out := buf.String()
	checks := []string{
		"| @coderabbitai | CHANGES_REQUESTED (stale) | deadbee |",
		"Stale blocking reviews detected: 1.",
		"gh ghent dismiss --pr 42 --message \"superseded by current HEAD\"",
	}
	for _, want := range checks {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\noutput:\n%s", want, out)
		}
	}
}

func TestMarkdownStatusMergeReady(t *testing.T) {
	var buf bytes.Buffer
	f := &MarkdownFormatter{}

	result := sampleStatusResult()
	result.IsMergeReady = true

	if err := f.FormatStatus(&buf, result); err != nil {
		t.Fatalf("FormatStatus: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "[READY]") {
		t.Errorf("merge-ready status missing [READY]\noutput:\n%s", out)
	}
}

func TestMarkdownStatusNoReviews(t *testing.T) {
	var buf bytes.Buffer
	f := &MarkdownFormatter{}

	result := sampleStatusResult()
	result.Reviews = nil

	if err := f.FormatStatus(&buf, result); err != nil {
		t.Fatalf("FormatStatus: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "No reviews yet.") {
		t.Errorf("no-reviews status missing 'No reviews yet.'\noutput:\n%s", out)
	}
}

func TestMarkdownStatusWithFailedChecks(t *testing.T) {
	var buf bytes.Buffer
	f := &MarkdownFormatter{}

	if err := f.FormatStatus(&buf, sampleStatusWithFailures()); err != nil {
		t.Fatalf("FormatStatus: %v", err)
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
		{"timed_out fail header", "### FAIL: e2e-tests"},
		{"timed_out log excerpt", "test timed out after 30m0s"},
	}

	for _, tc := range checks {
		if !strings.Contains(out, tc.want) {
			t.Errorf("%s: output missing %q\noutput:\n%s", tc.name, tc.want, out)
		}
	}
}

func TestMarkdownStatusNoFailedChecksWhenAllPass(t *testing.T) {
	var buf bytes.Buffer
	f := &MarkdownFormatter{}

	if err := f.FormatStatus(&buf, sampleStatusResult()); err != nil {
		t.Fatalf("FormatStatus: %v", err)
	}

	out := buf.String()
	if strings.Contains(out, "### FAIL:") {
		t.Errorf("all-pass status should not contain FAIL sections\noutput:\n%s", out)
	}
}

func TestMarkdownCompactStatusWithFailedChecks(t *testing.T) {
	var buf bytes.Buffer
	f := &MarkdownFormatter{}

	if err := f.FormatCompactStatus(&buf, sampleStatusWithFailures()); err != nil {
		t.Fatalf("FormatCompactStatus: %v", err)
	}

	out := buf.String()

	checks := []struct {
		name string
		want string
	}{
		{"fail name", "FAIL: lint-check"},
		{"annotation location", "src/main.go:42"},
		{"annotation message", "unused variable: x"},
		{"timed_out fail name", "FAIL: e2e-tests"},
	}

	for _, tc := range checks {
		if !strings.Contains(out, tc.want) {
			t.Errorf("%s: output missing %q\noutput:\n%s", tc.name, tc.want, out)
		}
	}
}

func TestMarkdownCompactStatusIncludesStaleCount(t *testing.T) {
	var buf bytes.Buffer
	f := &MarkdownFormatter{}

	if err := f.FormatCompactStatus(&buf, sampleStatusWithStaleReview()); err != nil {
		t.Fatalf("FormatCompactStatus: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "stale:1") {
		t.Errorf("compact output missing stale count\noutput:\n%s", out)
	}
}

func TestMarkdownDismissResultsStructure(t *testing.T) {
	var buf bytes.Buffer
	f := &MarkdownFormatter{}

	if err := f.FormatDismissResults(&buf, sampleDismissResults()); err != nil {
		t.Fatalf("FormatDismissResults: %v", err)
	}

	out := buf.String()
	checks := []string{
		"# Dismiss Results",
		"**Success:** 1 | **Failed:** 1",
		"| Review | Author | State | Commit | Action |",
		"| PRR_3 | @coderabbitai | DISMISSED | deadbee | dismissed |",
		"## Errors",
		"already dismissed",
	}
	for _, want := range checks {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\noutput:\n%s", want, out)
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
