package formatter

import (
	"fmt"
	"io"

	"github.com/indrasvat/gh-ghent/internal/domain"
)

// MarkdownFormatter outputs results as readable Markdown.
type MarkdownFormatter struct{}

func (f *MarkdownFormatter) FormatComments(w io.Writer, result *domain.CommentsResult) error {
	fmt.Fprintf(w, "# PR #%d — Review Comments\n\n", result.PRNumber)
	fmt.Fprintf(w, "**Unresolved:** %d | **Resolved:** %d | **Total:** %d\n",
		result.UnresolvedCount, result.ResolvedCount, result.TotalCount)

	for _, t := range result.Threads {
		fmt.Fprintf(w, "\n---\n\n")
		fmt.Fprintf(w, "## %s:%d\n\n", t.Path, t.Line)

		for _, c := range t.Comments {
			fmt.Fprintf(w, "**@%s** — %s\n\n", c.Author, c.CreatedAt.Format("2006-01-02 15:04"))
			fmt.Fprintf(w, "> %s\n", c.Body)

			if c.DiffHunk != "" {
				fmt.Fprintf(w, "\n<details>\n<summary>Diff</summary>\n\n```diff\n%s\n```\n\n</details>\n", c.DiffHunk)
			}
			fmt.Fprintln(w)
		}
	}
	return nil
}

func (f *MarkdownFormatter) FormatChecks(w io.Writer, result *domain.ChecksResult) error {
	fmt.Fprintf(w, "# PR #%d — Check Runs\n\n", result.PRNumber)
	fmt.Fprintf(w, "**Status:** %s | **Pass:** %d | **Fail:** %d | **Pending:** %d\n\n",
		result.OverallStatus, result.PassCount, result.FailCount, result.PendingCount)
	fmt.Fprintf(w, "| Check | Status | Conclusion |\n")
	fmt.Fprintf(w, "|-------|--------|------------|\n")
	for _, ch := range result.Checks {
		conclusion := ch.Conclusion
		if conclusion == "" {
			conclusion = "-"
		}
		fmt.Fprintf(w, "| %s | %s | %s |\n", ch.Name, ch.Status, conclusion)
	}

	// Annotations and log excerpts for failed checks
	for _, ch := range result.Checks {
		if len(ch.Annotations) > 0 {
			fmt.Fprintf(w, "\n### %s — Annotations\n\n", ch.Name)
			for _, a := range ch.Annotations {
				fmt.Fprintf(w, "- **%s** `%s:%d` — %s\n", a.AnnotationLevel, a.Path, a.StartLine, a.Message)
			}
		}
		if ch.LogExcerpt != "" {
			fmt.Fprintf(w, "\n### %s — Log Excerpt\n\n```\n%s\n```\n", ch.Name, ch.LogExcerpt)
		}
	}
	return nil
}

func (f *MarkdownFormatter) FormatReply(w io.Writer, result *domain.ReplyResult) error {
	fmt.Fprintf(w, "# Reply Posted\n\n")
	fmt.Fprintf(w, "**Thread:** %s\n", result.ThreadID)
	fmt.Fprintf(w, "**URL:** %s\n\n", result.URL)
	fmt.Fprintf(w, "> %s\n", result.Body)
	return nil
}

func (f *MarkdownFormatter) FormatResolveResults(w io.Writer, result *domain.ResolveResults) error {
	fmt.Fprintf(w, "# Resolve Results\n\n")
	fmt.Fprintf(w, "**Success:** %d | **Failed:** %d\n\n", result.SuccessCount, result.FailureCount)

	if len(result.Results) > 0 {
		fmt.Fprintf(w, "| Thread | File | Line | Action |\n")
		fmt.Fprintf(w, "|--------|------|------|--------|\n")
		for _, r := range result.Results {
			fmt.Fprintf(w, "| %s | %s | %d | %s |\n", r.ThreadID, r.Path, r.Line, r.Action)
		}
	}

	if len(result.Errors) > 0 {
		fmt.Fprintf(w, "\n## Errors\n\n")
		for _, e := range result.Errors {
			fmt.Fprintf(w, "- **%s:** %s\n", e.ThreadID, e.Message)
		}
	}
	return nil
}

func (f *MarkdownFormatter) FormatSummary(w io.Writer, result *domain.SummaryResult) error {
	mergeStatus := "NOT READY"
	if result.IsMergeReady {
		mergeStatus = "READY"
	}
	fmt.Fprintf(w, "# PR #%d — Summary [%s]\n\n", result.PRNumber, mergeStatus)

	// Comments section.
	fmt.Fprintf(w, "## Review Comments\n\n")
	fmt.Fprintf(w, "**Unresolved:** %d | **Resolved:** %d | **Total:** %d\n\n",
		result.Comments.UnresolvedCount, result.Comments.ResolvedCount, result.Comments.TotalCount)

	// Checks section.
	fmt.Fprintf(w, "## CI Checks\n\n")
	fmt.Fprintf(w, "**Status:** %s | **Pass:** %d | **Fail:** %d | **Pending:** %d\n\n",
		result.Checks.OverallStatus, result.Checks.PassCount, result.Checks.FailCount, result.Checks.PendingCount)

	// Reviews/Approvals section.
	fmt.Fprintf(w, "## Approvals\n\n")
	if len(result.Reviews) == 0 {
		fmt.Fprintf(w, "No reviews yet.\n")
	} else {
		fmt.Fprintf(w, "| Reviewer | State |\n")
		fmt.Fprintf(w, "|----------|-------|\n")
		for _, r := range result.Reviews {
			fmt.Fprintf(w, "| @%s | %s |\n", r.Author, r.State)
		}
	}

	return nil
}
