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
	if result.Since != "" {
		fmt.Fprintf(w, "> Filtered: showing activity since %s\n\n", result.Since)
	}
	fmt.Fprintf(w, "**Unresolved:** %d | **Resolved:** %d | **Total:** %d",
		result.UnresolvedCount, result.ResolvedCount, result.TotalCount)
	if result.BotThreadCount > 0 || result.UnansweredCount > 0 {
		fmt.Fprintf(w, " | **Bot:** %d | **Unanswered:** %d",
			result.BotThreadCount, result.UnansweredCount)
	}
	fmt.Fprintln(w)

	for _, t := range result.Threads {
		fmt.Fprintf(w, "\n---\n\n")
		fmt.Fprintf(w, "## %s:%d\n\n", t.Path, t.Line)

		for _, c := range t.Comments {
			botBadge := ""
			if c.IsBot {
				botBadge = " [bot]"
			}
			fmt.Fprintf(w, "**@%s%s** — %s\n\n", c.Author, botBadge, c.CreatedAt.Format("2006-01-02 15:04"))
			fmt.Fprintf(w, "> %s\n", c.Body)

			if c.DiffHunk != "" {
				fmt.Fprintf(w, "\n<details>\n<summary>Diff</summary>\n\n```diff\n%s\n```\n\n</details>\n", c.DiffHunk)
			}
			fmt.Fprintln(w)
		}
	}
	return nil
}

func (f *MarkdownFormatter) FormatGroupedComments(w io.Writer, result *domain.GroupedCommentsResult) error {
	fmt.Fprintf(w, "# PR #%d — Review Comments (by %s)\n\n", result.PRNumber, result.GroupBy)
	fmt.Fprintf(w, "**Unresolved:** %d | **Resolved:** %d | **Total:** %d\n",
		result.UnresolvedCount, result.ResolvedCount, result.TotalCount)

	for _, g := range result.Groups {
		fmt.Fprintf(w, "\n---\n\n")
		fmt.Fprintf(w, "## %s\n\n", g.Key)

		for _, t := range g.Threads {
			fmt.Fprintf(w, "### %s:%d\n\n", t.Path, t.Line)
			for _, c := range t.Comments {
				botBadge := ""
				if c.IsBot {
					botBadge = " [bot]"
				}
				fmt.Fprintf(w, "**@%s%s** — %s\n\n", c.Author, botBadge, c.CreatedAt.Format("2006-01-02 15:04"))
				fmt.Fprintf(w, "> %s\n", c.Body)
				if c.DiffHunk != "" {
					fmt.Fprintf(w, "\n<details>\n<summary>Diff</summary>\n\n```diff\n%s\n```\n\n</details>\n", c.DiffHunk)
				}
				fmt.Fprintln(w)
			}
		}
	}
	return nil
}

func (f *MarkdownFormatter) FormatChecks(w io.Writer, result *domain.ChecksResult) error {
	fmt.Fprintf(w, "# PR #%d — Check Runs\n\n", result.PRNumber)
	if result.Since != "" {
		fmt.Fprintf(w, "> Filtered: showing activity since %s\n\n", result.Since)
	}
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
	if result.Resolved != nil {
		fmt.Fprintf(w, "\n## Thread Resolved\n\n")
		fmt.Fprintf(w, "**Action:** %s\n", result.Resolved.Action)
		if result.Resolved.Path != "" {
			fmt.Fprintf(w, "**File:** %s:%d\n", result.Resolved.Path, result.Resolved.Line)
		}
	}
	if result.ResolveError != "" {
		fmt.Fprintf(w, "\n## Resolve Failed\n\n")
		fmt.Fprintf(w, "**Error:** %s\n", result.ResolveError)
	}
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

func (f *MarkdownFormatter) FormatDismissResults(w io.Writer, result *domain.DismissResults) error {
	fmt.Fprintf(w, "# Dismiss Results\n\n")
	fmt.Fprintf(w, "**Success:** %d | **Failed:** %d", result.SuccessCount, result.FailureCount)
	if result.DryRun {
		fmt.Fprintf(w, " | **Dry Run:** true")
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w)

	if len(result.Results) > 0 {
		fmt.Fprintf(w, "| Review | Author | State | Commit | Action |\n")
		fmt.Fprintf(w, "|--------|--------|-------|--------|--------|\n")
		for _, r := range result.Results {
			commitID := r.CommitID
			if len(commitID) > 7 {
				commitID = commitID[:7]
			}
			fmt.Fprintf(w, "| %s | @%s | %s | %s | %s |\n", r.ReviewID, r.Author, r.State, commitID, r.Action)
		}
	}

	if len(result.Errors) > 0 {
		fmt.Fprintf(w, "\n## Errors\n\n")
		for _, e := range result.Errors {
			fmt.Fprintf(w, "- **%s:** %s\n", e.ReviewID, e.Message)
		}
	}
	return nil
}

func (f *MarkdownFormatter) FormatCompactStatus(w io.Writer, result *domain.StatusResult) error {
	mergeStatus := "NOT READY"
	if result.IsMergeReady {
		mergeStatus = "READY"
	}

	// One-line KPI status.
	fmt.Fprintf(w, "PR #%d [%s] — unresolved:%d checks:%s (pass:%d fail:%d)",
		result.PRNumber, mergeStatus,
		result.Comments.UnresolvedCount,
		result.Checks.OverallStatus,
		result.Checks.PassCount, result.Checks.FailCount)

	if result.PRAge != "" {
		fmt.Fprintf(w, " age:%s", result.PRAge)
	}
	if result.LastUpdate != "" {
		fmt.Fprintf(w, " last:%s", result.LastUpdate)
	}
	if result.ReviewCycles > 0 {
		fmt.Fprintf(w, " cycles:%d", result.ReviewCycles)
	}
	if len(result.StaleReviews) > 0 {
		fmt.Fprintf(w, " stale:%d", len(result.StaleReviews))
	}
	fmt.Fprintln(w)

	// Thread digest table.
	if len(result.Comments.Threads) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "| File:Line | Author | Preview |")
		fmt.Fprintln(w, "|-----------|--------|---------|")
		for _, t := range result.Comments.Threads {
			if len(t.Comments) == 0 {
				continue
			}
			first := t.Comments[0]
			preview := first.Body
			if len(preview) > 60 {
				preview = preview[:60] + "..."
			}
			fmt.Fprintf(w, "| %s:%d | @%s | %s |\n", t.Path, t.Line, first.Author, preview)
		}
	}

	// Failed checks digest.
	for _, ch := range result.Checks.Checks {
		if !domain.IsFailConclusion(ch.Conclusion) {
			continue
		}
		fmt.Fprintf(w, "\nFAIL: %s", ch.Name)
		for _, a := range ch.Annotations {
			fmt.Fprintf(w, " | %s:%d %s", a.Path, a.StartLine, a.Message)
		}
		fmt.Fprintln(w)
	}

	return nil
}

func (f *MarkdownFormatter) FormatWatchStatus(w io.Writer, status *domain.WatchStatus) error {
	fmt.Fprintf(w, "[%s] %s — %d/%d completed (pass:%d fail:%d pending:%d)",
		status.Timestamp.Format("15:04:05"),
		status.OverallStatus,
		status.Completed, status.Total,
		status.PassCount, status.FailCount, status.PendingCount)
	for _, ev := range status.Events {
		fmt.Fprintf(w, " | %s→%s", ev.Name, ev.Conclusion)
	}
	if status.ReviewPhase != "" {
		fmt.Fprintf(w, " review:%s", status.ReviewPhase)
		if status.ReviewConfidence != "" {
			fmt.Fprintf(w, "/%s", status.ReviewConfidence)
		}
		fmt.Fprintf(w, " idle:%ds timeout:%ds", status.ReviewIdleSecs, status.ReviewTimeoutIn)
		if status.ReviewTailProbes > 0 {
			fmt.Fprintf(w, " tail:%d", status.ReviewTailProbes)
		}
	}
	_, err := fmt.Fprintln(w)
	return err
}

func (f *MarkdownFormatter) FormatStatus(w io.Writer, result *domain.StatusResult) error {
	mergeStatus := "NOT READY"
	if result.IsMergeReady {
		mergeStatus = "READY"
	}
	fmt.Fprintf(w, "# PR #%d — Status [%s]\n\n", result.PRNumber, mergeStatus)

	// Comments section.
	fmt.Fprintf(w, "## Review Comments\n\n")
	fmt.Fprintf(w, "**Unresolved:** %d | **Resolved:** %d | **Total:** %d\n\n",
		result.Comments.UnresolvedCount, result.Comments.ResolvedCount, result.Comments.TotalCount)

	// Unresolved thread details.
	for _, t := range result.Comments.Threads {
		if t.IsResolved || len(t.Comments) == 0 {
			continue
		}
		first := t.Comments[0]
		preview := first.Body
		if len(preview) > 80 {
			preview = preview[:80] + "..."
		}
		fmt.Fprintf(w, "- **%s:%d** @%s — %s\n", t.Path, t.Line, first.Author, preview)
	}
	if result.Comments.UnresolvedCount > 0 {
		fmt.Fprintln(w)
	}

	// Checks section.
	fmt.Fprintf(w, "## CI Checks\n\n")
	fmt.Fprintf(w, "**Status:** %s | **Pass:** %d | **Fail:** %d | **Pending:** %d\n\n",
		result.Checks.OverallStatus, result.Checks.PassCount, result.Checks.FailCount, result.Checks.PendingCount)

	// Failed check details (annotations + log excerpts).
	for _, ch := range result.Checks.Checks {
		if !domain.IsFailConclusion(ch.Conclusion) {
			continue
		}
		fmt.Fprintf(w, "### FAIL: %s\n\n", ch.Name)
		for _, a := range ch.Annotations {
			fmt.Fprintf(w, "- **%s** `%s:%d` — %s\n", a.AnnotationLevel, a.Path, a.StartLine, a.Message)
		}
		if ch.LogExcerpt != "" {
			fmt.Fprintf(w, "\n```\n%s\n```\n", ch.LogExcerpt)
		}
		fmt.Fprintln(w)
	}

	// Review monitor section (if --await-review was used).
	if result.ReviewMonitor != nil {
		fmt.Fprintf(w, "## Review Monitor\n\n")
		fmt.Fprintf(w, "**Phase:** %s | **Confidence:** %s | **Activity:** %d changes | **Wait:** %s",
			result.ReviewMonitor.Phase,
			result.ReviewMonitor.Confidence,
			result.ReviewMonitor.ActivityCount,
			formatSettlementDuration(result.ReviewMonitor.WaitSeconds))
		if result.ReviewMonitor.TailProbes > 0 {
			fmt.Fprintf(w, " | **Tail Probes:** %d", result.ReviewMonitor.TailProbes)
		}
		fmt.Fprintln(w)
		if result.ReviewMonitor.Phase == domain.ReviewPhaseTimeout {
			fmt.Fprintf(w, "\nWarning: additional bot reviews may still arrive after this timeout.\n")
		} else if result.ReviewMonitor.Confidence == domain.ReviewConfidenceHigh {
			fmt.Fprintf(w, "\nReview activity stabilized through bounded confirmation probes.\n")
		}
		fmt.Fprintln(w)
	}

	// Reviews/Approvals section.
	fmt.Fprintf(w, "## Approvals\n\n")
	if len(result.Reviews) == 0 {
		fmt.Fprintf(w, "No reviews yet.\n")
	} else {
		fmt.Fprintf(w, "| Reviewer | State | Commit |\n")
		fmt.Fprintf(w, "|----------|-------|--------|\n")
		for _, r := range result.Reviews {
			state := string(r.State)
			if r.IsStale && r.State == domain.ReviewChangesRequested {
				state += " (stale)"
			}
			commitID := r.CommitID
			if len(commitID) > 7 {
				commitID = commitID[:7]
			}
			if commitID == "" {
				commitID = "-"
			}
			fmt.Fprintf(w, "| @%s | %s | %s |\n", r.Author, state, commitID)
		}
	}

	if len(result.StaleReviews) > 0 {
		fmt.Fprintf(w, "\nStale blocking reviews detected: %d.\n", len(result.StaleReviews))
		fmt.Fprintf(w, "Suggested: `gh ghent dismiss --pr %d --message \"superseded by current HEAD\"`\n", result.PRNumber)
	}

	return nil
}

// formatSettlementDuration converts seconds to a human-readable duration string.
func formatSettlementDuration(secs int) string {
	if secs < 60 {
		return fmt.Sprintf("%ds", secs)
	}
	m := secs / 60
	s := secs % 60
	if s == 0 {
		return fmt.Sprintf("%dm", m)
	}
	return fmt.Sprintf("%dm%ds", m, s)
}
