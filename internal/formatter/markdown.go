package formatter

import (
	"fmt"
	"io"

	"github.com/indrasvat/ghent/internal/domain"
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
	fmt.Fprintf(w, "**Status:** %s | **Pass:** %d | **Fail:** %d | **Pending:** %d\n",
		result.OverallStatus, result.PassCount, result.FailCount, result.PendingCount)
	for _, ch := range result.Checks {
		fmt.Fprintf(w, "\n- **%s** — %s (%s)\n", ch.Name, ch.Conclusion, ch.Status)
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

func (f *MarkdownFormatter) FormatSummary(w io.Writer, result *domain.SummaryResult) error {
	fmt.Fprintf(w, "# PR #%d — Summary\n\n", result.PRNumber)
	fmt.Fprintf(w, "**Merge Ready:** %v\n\n", result.IsMergeReady)
	fmt.Fprintf(w, "## Comments\n\n")
	fmt.Fprintf(w, "Unresolved: %d | Resolved: %d\n\n", result.Comments.UnresolvedCount, result.Comments.ResolvedCount)
	fmt.Fprintf(w, "## Checks\n\n")
	fmt.Fprintf(w, "Status: %s | Pass: %d | Fail: %d\n", result.Checks.OverallStatus, result.Checks.PassCount, result.Checks.FailCount)
	return nil
}
