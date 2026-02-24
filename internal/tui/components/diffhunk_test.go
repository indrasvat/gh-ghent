package components

import (
	"strings"
	"testing"
)

const testHunk = `@@ -44,8 +44,10 @@
 func (c *Client) FetchThreads(owner, repo string, pr int) ([]Thread, error) {
     var query threadQuery
     err := c.gql.Query("PullRequestThreads", &query, variables)
-    if err != nil {
-        return nil, err
+    if err != nil {
+        return nil, fmt.Errorf("fetch threads: %w", err)
     }
     return mapThreads(query), nil`

func TestRenderDiffHunk(t *testing.T) {
	got := RenderDiffHunk(testHunk, 80)

	// Should contain all line types
	if !strings.Contains(got, "@@") {
		t.Error("missing diff header (@@)")
	}
	if got == "" {
		t.Error("empty output from RenderDiffHunk")
	}

	// Should contain ANSI reset sequences between lines
	resetCount := strings.Count(got, "\033[0m")
	if resetCount == 0 {
		t.Error("no ANSI resets found — pitfall 7.6 not addressed")
	}
}

func TestRenderDiffHunkEmpty(t *testing.T) {
	got := RenderDiffHunk("", 80)
	if got != "" {
		t.Errorf("expected empty for empty hunk, got %q", got)
	}
}

func TestRenderDiffHunkLineTypes(t *testing.T) {
	hunk := "@@ -1,3 +1,3 @@\n context\n-removed\n+added"
	got := RenderDiffHunk(hunk, 80)

	lines := strings.Split(got, "\n")
	if len(lines) < 3 {
		t.Fatalf("expected at least 3 lines, got %d", len(lines))
	}
}

func TestRenderDiffHunkCompact(t *testing.T) {
	got := RenderDiffHunkCompact(testHunk, 4)

	lines := strings.Split(got, "\n")
	// Should have at most 4 content lines + possible "..." indicator
	contentLines := 0
	for _, line := range lines {
		if line != "" && !strings.Contains(line, "...") {
			contentLines++
		}
	}
	if contentLines > 4 {
		t.Errorf("compact mode should show at most 4 lines, got %d", contentLines)
	}
}

func TestRenderDiffHunkCompactEmpty(t *testing.T) {
	got := RenderDiffHunkCompact("", 4)
	if got != "" {
		t.Errorf("expected empty for empty hunk, got %q", got)
	}
}

func TestRenderDiffHunkWidths(t *testing.T) {
	// Should not panic at any width
	widths := []int{0, 10, 20, 40, 80, 120}
	for _, w := range widths {
		got := RenderDiffHunk(testHunk, w)
		if w > 0 && got == "" {
			t.Errorf("empty output for width %d", w)
		}
	}
}
