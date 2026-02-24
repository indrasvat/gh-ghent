package formatter

import (
	"encoding/json"
	"io"

	"github.com/indrasvat/gh-ghent/internal/domain"
)

// JSONFormatter outputs results as indented JSON.
type JSONFormatter struct{}

func (f *JSONFormatter) FormatComments(w io.Writer, result *domain.CommentsResult) error {
	return encodeJSON(w, result)
}

func (f *JSONFormatter) FormatGroupedComments(w io.Writer, result *domain.GroupedCommentsResult) error {
	return encodeJSON(w, result)
}

func (f *JSONFormatter) FormatChecks(w io.Writer, result *domain.ChecksResult) error {
	return encodeJSON(w, result)
}

func (f *JSONFormatter) FormatReply(w io.Writer, result *domain.ReplyResult) error {
	return encodeJSON(w, result)
}

func (f *JSONFormatter) FormatResolveResults(w io.Writer, result *domain.ResolveResults) error {
	return encodeJSON(w, result)
}

func (f *JSONFormatter) FormatSummary(w io.Writer, result *domain.SummaryResult) error {
	return encodeJSON(w, result)
}

func (f *JSONFormatter) FormatCompactSummary(w io.Writer, result *domain.SummaryResult) error {
	type compactThread struct {
		File        string `json:"file"`
		Line        int    `json:"line"`
		Author      string `json:"author"`
		BodyPreview string `json:"body_preview"`
	}
	type compactAnnotation struct {
		Path    string `json:"path"`
		Line    int    `json:"line"`
		Level   string `json:"level"`
		Message string `json:"message"`
	}
	type compactFailedCheck struct {
		Name        string              `json:"name"`
		Annotations []compactAnnotation `json:"annotations,omitempty"`
		LogExcerpt  string              `json:"log_excerpt,omitempty"`
	}
	type compactSummary struct {
		PRNumber     int                  `json:"pr_number"`
		IsMergeReady bool                 `json:"is_merge_ready"`
		PRAge        string               `json:"pr_age,omitempty"`
		LastUpdate   string               `json:"last_update,omitempty"`
		ReviewCycles int                  `json:"review_cycles,omitempty"`
		Unresolved   int                  `json:"unresolved"`
		CheckStatus  string               `json:"check_status"`
		PassCount    int                  `json:"pass_count"`
		FailCount    int                  `json:"fail_count"`
		Threads      []compactThread      `json:"threads,omitempty"`
		FailedChecks []compactFailedCheck `json:"failed_checks,omitempty"`
	}

	compact := compactSummary{
		PRNumber:     result.PRNumber,
		IsMergeReady: result.IsMergeReady,
		PRAge:        result.PRAge,
		LastUpdate:   result.LastUpdate,
		ReviewCycles: result.ReviewCycles,
		Unresolved:   result.Comments.UnresolvedCount,
		CheckStatus:  string(result.Checks.OverallStatus),
		PassCount:    result.Checks.PassCount,
		FailCount:    result.Checks.FailCount,
	}

	for _, t := range result.Comments.Threads {
		if len(t.Comments) == 0 {
			continue
		}
		first := t.Comments[0]
		preview := first.Body
		if len(preview) > 80 {
			preview = preview[:80] + "..."
		}
		compact.Threads = append(compact.Threads, compactThread{
			File:        t.Path,
			Line:        t.Line,
			Author:      first.Author,
			BodyPreview: preview,
		})
	}

	for _, ch := range result.Checks.Checks {
		if !domain.IsFailConclusion(ch.Conclusion) {
			continue
		}
		fc := compactFailedCheck{
			Name:       ch.Name,
			LogExcerpt: ch.LogExcerpt,
		}
		for _, a := range ch.Annotations {
			fc.Annotations = append(fc.Annotations, compactAnnotation{
				Path:    a.Path,
				Line:    a.StartLine,
				Level:   a.AnnotationLevel,
				Message: a.Message,
			})
		}
		compact.FailedChecks = append(compact.FailedChecks, fc)
	}

	return encodeJSON(w, compact)
}

func (f *JSONFormatter) FormatWatchStatus(w io.Writer, status *domain.WatchStatus) error {
	// NDJSON: one compact JSON object per line (no indentation).
	enc := json.NewEncoder(w)
	return enc.Encode(status)
}

func encodeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
