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
