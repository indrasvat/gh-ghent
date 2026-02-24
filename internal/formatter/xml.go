package formatter

import (
	"encoding/xml"
	"io"
	"time"

	"github.com/indrasvat/gh-ghent/internal/domain"
)

// XMLFormatter outputs results as XML.
type XMLFormatter struct{}

func (f *XMLFormatter) FormatComments(w io.Writer, result *domain.CommentsResult) error {
	out := xmlComments{
		PRNumber:        result.PRNumber,
		TotalCount:      result.TotalCount,
		ResolvedCount:   result.ResolvedCount,
		UnresolvedCount: result.UnresolvedCount,
		Since:           result.Since,
	}
	for _, t := range result.Threads {
		xt := xmlThread{
			ID:         t.ID,
			Path:       t.Path,
			Line:       t.Line,
			IsResolved: t.IsResolved,
			IsOutdated: t.IsOutdated,
		}
		for _, c := range t.Comments {
			xt.Comments = append(xt.Comments, xmlComment{
				ID:        c.ID,
				Author:    c.Author,
				Body:      c.Body,
				CreatedAt: c.CreatedAt.Format(time.RFC3339),
				URL:       c.URL,
				DiffHunk:  c.DiffHunk,
			})
		}
		out.Threads = append(out.Threads, xt)
	}
	if _, err := io.WriteString(w, xml.Header); err != nil {
		return err
	}
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	if err := enc.Encode(out); err != nil {
		return err
	}
	_, err := io.WriteString(w, "\n")
	return err
}

func (f *XMLFormatter) FormatGroupedComments(w io.Writer, result *domain.GroupedCommentsResult) error {
	out := xmlGroupedComments{
		PRNumber:        result.PRNumber,
		GroupBy:         result.GroupBy,
		TotalCount:      result.TotalCount,
		ResolvedCount:   result.ResolvedCount,
		UnresolvedCount: result.UnresolvedCount,
	}
	for _, g := range result.Groups {
		xg := xmlCommentGroup{Key: g.Key}
		for _, t := range g.Threads {
			xt := xmlThread{
				ID:         t.ID,
				Path:       t.Path,
				Line:       t.Line,
				IsResolved: t.IsResolved,
				IsOutdated: t.IsOutdated,
			}
			for _, c := range t.Comments {
				xt.Comments = append(xt.Comments, xmlComment{
					ID:        c.ID,
					Author:    c.Author,
					Body:      c.Body,
					CreatedAt: c.CreatedAt.Format(time.RFC3339),
					URL:       c.URL,
					DiffHunk:  c.DiffHunk,
				})
			}
			xg.Threads = append(xg.Threads, xt)
		}
		out.Groups = append(out.Groups, xg)
	}
	if _, err := io.WriteString(w, xml.Header); err != nil {
		return err
	}
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	if err := enc.Encode(out); err != nil {
		return err
	}
	_, err := io.WriteString(w, "\n")
	return err
}

func (f *XMLFormatter) FormatChecks(w io.Writer, result *domain.ChecksResult) error {
	out := xmlChecks{
		PRNumber:      result.PRNumber,
		HeadSHA:       result.HeadSHA,
		OverallStatus: string(result.OverallStatus),
		PassCount:     result.PassCount,
		FailCount:     result.FailCount,
		PendingCount:  result.PendingCount,
		Since:         result.Since,
	}
	for _, ch := range result.Checks {
		xc := xmlCheckRun{
			ID:         ch.ID,
			Name:       ch.Name,
			Status:     ch.Status,
			Conclusion: ch.Conclusion,
			HTMLURL:    ch.HTMLURL,
			LogExcerpt: ch.LogExcerpt,
		}
		for _, a := range ch.Annotations {
			xc.Annotations = append(xc.Annotations, xmlAnnotation{
				Path:            a.Path,
				StartLine:       a.StartLine,
				EndLine:         a.EndLine,
				AnnotationLevel: a.AnnotationLevel,
				Title:           a.Title,
				Message:         a.Message,
			})
		}
		out.Checks = append(out.Checks, xc)
	}
	if _, err := io.WriteString(w, xml.Header); err != nil {
		return err
	}
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	if err := enc.Encode(out); err != nil {
		return err
	}
	_, err := io.WriteString(w, "\n")
	return err
}

func (f *XMLFormatter) FormatReply(w io.Writer, result *domain.ReplyResult) error {
	if _, err := io.WriteString(w, xml.Header); err != nil {
		return err
	}
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	return enc.Encode(result)
}

func (f *XMLFormatter) FormatResolveResults(w io.Writer, result *domain.ResolveResults) error {
	out := xmlResolveResults{
		SuccessCount: result.SuccessCount,
		FailureCount: result.FailureCount,
	}
	for _, r := range result.Results {
		out.Results = append(out.Results, xmlResolveResult{
			ThreadID:   r.ThreadID,
			Path:       r.Path,
			Line:       r.Line,
			IsResolved: r.IsResolved,
			Action:     r.Action,
		})
	}
	for _, e := range result.Errors {
		out.Errors = append(out.Errors, xmlResolveError{
			ThreadID: e.ThreadID,
			Message:  e.Message,
		})
	}
	if _, err := io.WriteString(w, xml.Header); err != nil {
		return err
	}
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	if err := enc.Encode(out); err != nil {
		return err
	}
	_, err := io.WriteString(w, "\n")
	return err
}

func (f *XMLFormatter) FormatSummary(w io.Writer, result *domain.SummaryResult) error {
	out := xmlSummary{
		PRNumber:     result.PRNumber,
		IsMergeReady: result.IsMergeReady,
		Comments: xmlSummaryComments{
			TotalCount:      result.Comments.TotalCount,
			ResolvedCount:   result.Comments.ResolvedCount,
			UnresolvedCount: result.Comments.UnresolvedCount,
		},
		Checks: xmlSummaryChecks{
			OverallStatus: string(result.Checks.OverallStatus),
			PassCount:     result.Checks.PassCount,
			FailCount:     result.Checks.FailCount,
			PendingCount:  result.Checks.PendingCount,
		},
	}
	// Add unresolved threads to comments section.
	for _, t := range result.Comments.Threads {
		if t.IsResolved {
			continue
		}
		xt := xmlThread{
			ID:         t.ID,
			Path:       t.Path,
			Line:       t.Line,
			IsResolved: t.IsResolved,
			IsOutdated: t.IsOutdated,
		}
		for _, c := range t.Comments {
			xt.Comments = append(xt.Comments, xmlComment{
				ID:        c.ID,
				Author:    c.Author,
				Body:      c.Body,
				CreatedAt: c.CreatedAt.Format(time.RFC3339),
				URL:       c.URL,
			})
		}
		out.Comments.Threads = append(out.Comments.Threads, xt)
	}
	// Add failed checks with annotations and log excerpts.
	for _, ch := range result.Checks.Checks {
		if ch.Conclusion != "failure" {
			continue
		}
		xc := xmlCheckRun{
			ID:         ch.ID,
			Name:       ch.Name,
			Status:     ch.Status,
			Conclusion: ch.Conclusion,
			HTMLURL:    ch.HTMLURL,
			LogExcerpt: ch.LogExcerpt,
		}
		for _, a := range ch.Annotations {
			xc.Annotations = append(xc.Annotations, xmlAnnotation{
				Path:            a.Path,
				StartLine:       a.StartLine,
				EndLine:         a.EndLine,
				AnnotationLevel: a.AnnotationLevel,
				Title:           a.Title,
				Message:         a.Message,
			})
		}
		out.Checks.FailedChecks = append(out.Checks.FailedChecks, xc)
	}
	for _, r := range result.Reviews {
		out.Reviews = append(out.Reviews, xmlReview{
			ID:          r.ID,
			Author:      r.Author,
			State:       string(r.State),
			Body:        r.Body,
			SubmittedAt: r.SubmittedAt.Format(time.RFC3339),
		})
	}
	if _, err := io.WriteString(w, xml.Header); err != nil {
		return err
	}
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	if err := enc.Encode(out); err != nil {
		return err
	}
	_, err := io.WriteString(w, "\n")
	return err
}

func (f *XMLFormatter) FormatCompactSummary(w io.Writer, result *domain.SummaryResult) error {
	out := xmlCompactSummary{
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
		out.Threads = append(out.Threads, xmlCompactThread{
			File:        t.Path,
			Line:        t.Line,
			Author:      first.Author,
			BodyPreview: preview,
		})
	}

	for _, ch := range result.Checks.Checks {
		if ch.Conclusion != "failure" {
			continue
		}
		xc := xmlCheckRun{
			ID:         ch.ID,
			Name:       ch.Name,
			Status:     ch.Status,
			Conclusion: ch.Conclusion,
			HTMLURL:    ch.HTMLURL,
			LogExcerpt: ch.LogExcerpt,
		}
		for _, a := range ch.Annotations {
			xc.Annotations = append(xc.Annotations, xmlAnnotation{
				Path:            a.Path,
				StartLine:       a.StartLine,
				EndLine:         a.EndLine,
				AnnotationLevel: a.AnnotationLevel,
				Title:           a.Title,
				Message:         a.Message,
			})
		}
		out.FailedChecks = append(out.FailedChecks, xc)
	}

	if _, err := io.WriteString(w, xml.Header); err != nil {
		return err
	}
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	if err := enc.Encode(out); err != nil {
		return err
	}
	_, err := io.WriteString(w, "\n")
	return err
}

func (f *XMLFormatter) FormatWatchStatus(w io.Writer, status *domain.WatchStatus) error {
	out := xmlWatchStatus{
		Timestamp:     status.Timestamp.Format(time.RFC3339),
		OverallStatus: string(status.OverallStatus),
		Completed:     status.Completed,
		Total:         status.Total,
		PassCount:     status.PassCount,
		FailCount:     status.FailCount,
		PendingCount:  status.PendingCount,
		Final:         status.Final,
	}
	for _, ev := range status.Events {
		out.Events = append(out.Events, xmlWatchEvent{
			Name:       ev.Name,
			Status:     ev.Status,
			Conclusion: ev.Conclusion,
			Timestamp:  ev.Timestamp.Format(time.RFC3339),
		})
	}
	if _, err := io.WriteString(w, xml.Header); err != nil {
		return err
	}
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	if err := enc.Encode(out); err != nil {
		return err
	}
	_, err := io.WriteString(w, "\n")
	return err
}

type xmlWatchStatus struct {
	XMLName       xml.Name        `xml:"watch_status"`
	Timestamp     string          `xml:"timestamp,attr"`
	OverallStatus string          `xml:"overall_status,attr"`
	Completed     int             `xml:"completed,attr"`
	Total         int             `xml:"total,attr"`
	PassCount     int             `xml:"pass_count,attr"`
	FailCount     int             `xml:"fail_count,attr"`
	PendingCount  int             `xml:"pending_count,attr"`
	Final         bool            `xml:"final,attr"`
	Events        []xmlWatchEvent `xml:"event,omitempty"`
}

type xmlWatchEvent struct {
	Name       string `xml:"name,attr"`
	Status     string `xml:"status,attr"`
	Conclusion string `xml:"conclusion,attr"`
	Timestamp  string `xml:"timestamp,attr"`
}

type xmlGroupedComments struct {
	XMLName         xml.Name          `xml:"grouped_comments"`
	PRNumber        int               `xml:"pr_number,attr"`
	GroupBy         string            `xml:"group_by,attr"`
	TotalCount      int               `xml:"total_count,attr"`
	ResolvedCount   int               `xml:"resolved_count,attr"`
	UnresolvedCount int               `xml:"unresolved_count,attr"`
	Groups          []xmlCommentGroup `xml:"group"`
}

type xmlCommentGroup struct {
	Key     string      `xml:"key,attr"`
	Threads []xmlThread `xml:"thread"`
}

type xmlComments struct {
	XMLName         xml.Name    `xml:"comments"`
	PRNumber        int         `xml:"pr_number,attr"`
	TotalCount      int         `xml:"total_count,attr"`
	ResolvedCount   int         `xml:"resolved_count,attr"`
	UnresolvedCount int         `xml:"unresolved_count,attr"`
	Since           string      `xml:"since,attr,omitempty"`
	Threads         []xmlThread `xml:"thread"`
}

type xmlThread struct {
	ID         string       `xml:"id,attr"`
	Path       string       `xml:"path,attr"`
	Line       int          `xml:"line,attr"`
	IsResolved bool         `xml:"resolved,attr"`
	IsOutdated bool         `xml:"outdated,attr"`
	Comments   []xmlComment `xml:"comment"`
}

type xmlComment struct {
	ID        string `xml:"id,attr"`
	Author    string `xml:"author,attr"`
	CreatedAt string `xml:"created_at,attr"`
	URL       string `xml:"url,attr"`
	Body      string `xml:"body"`
	DiffHunk  string `xml:"diff_hunk,omitempty"`
}

type xmlChecks struct {
	XMLName       xml.Name      `xml:"checks"`
	PRNumber      int           `xml:"pr_number,attr"`
	HeadSHA       string        `xml:"head_sha,attr"`
	OverallStatus string        `xml:"overall_status,attr"`
	PassCount     int           `xml:"pass_count,attr"`
	FailCount     int           `xml:"fail_count,attr"`
	PendingCount  int           `xml:"pending_count,attr"`
	Since         string        `xml:"since,attr,omitempty"`
	Checks        []xmlCheckRun `xml:"check"`
}

type xmlCheckRun struct {
	ID          int64           `xml:"id,attr"`
	Name        string          `xml:"name,attr"`
	Status      string          `xml:"status,attr"`
	Conclusion  string          `xml:"conclusion,attr"`
	HTMLURL     string          `xml:"html_url,attr"`
	Annotations []xmlAnnotation `xml:"annotation,omitempty"`
	LogExcerpt  string          `xml:"log_excerpt,omitempty"`
}

type xmlAnnotation struct {
	Path            string `xml:"path,attr"`
	StartLine       int    `xml:"start_line,attr"`
	EndLine         int    `xml:"end_line,attr"`
	AnnotationLevel string `xml:"level,attr"`
	Title           string `xml:"title"`
	Message         string `xml:"message"`
}

type xmlResolveResults struct {
	XMLName      xml.Name           `xml:"resolve_results"`
	SuccessCount int                `xml:"success_count,attr"`
	FailureCount int                `xml:"failure_count,attr"`
	Results      []xmlResolveResult `xml:"result"`
	Errors       []xmlResolveError  `xml:"error,omitempty"`
}

type xmlResolveResult struct {
	ThreadID   string `xml:"thread_id,attr"`
	Path       string `xml:"path,attr"`
	Line       int    `xml:"line,attr"`
	IsResolved bool   `xml:"is_resolved,attr"`
	Action     string `xml:"action,attr"`
}

type xmlResolveError struct {
	ThreadID string `xml:"thread_id,attr"`
	Message  string `xml:",chardata"`
}

type xmlSummary struct {
	XMLName      xml.Name           `xml:"summary"`
	PRNumber     int                `xml:"pr_number,attr"`
	IsMergeReady bool               `xml:"is_merge_ready,attr"`
	Comments     xmlSummaryComments `xml:"comments"`
	Checks       xmlSummaryChecks   `xml:"checks"`
	Reviews      []xmlReview        `xml:"review,omitempty"`
}

type xmlSummaryComments struct {
	TotalCount      int         `xml:"total_count,attr"`
	ResolvedCount   int         `xml:"resolved_count,attr"`
	UnresolvedCount int         `xml:"unresolved_count,attr"`
	Threads         []xmlThread `xml:"thread,omitempty"`
}

type xmlSummaryChecks struct {
	OverallStatus string        `xml:"overall_status,attr"`
	PassCount     int           `xml:"pass_count,attr"`
	FailCount     int           `xml:"fail_count,attr"`
	PendingCount  int           `xml:"pending_count,attr"`
	FailedChecks  []xmlCheckRun `xml:"failed_check,omitempty"`
}

type xmlReview struct {
	ID          string `xml:"id,attr"`
	Author      string `xml:"author,attr"`
	State       string `xml:"state,attr"`
	SubmittedAt string `xml:"submitted_at,attr"`
	Body        string `xml:"body,omitempty"`
}

type xmlCompactSummary struct {
	XMLName      xml.Name           `xml:"compact_summary"`
	PRNumber     int                `xml:"pr_number,attr"`
	IsMergeReady bool               `xml:"is_merge_ready,attr"`
	PRAge        string             `xml:"pr_age,attr,omitempty"`
	LastUpdate   string             `xml:"last_update,attr,omitempty"`
	ReviewCycles int                `xml:"review_cycles,attr,omitempty"`
	Unresolved   int                `xml:"unresolved,attr"`
	CheckStatus  string             `xml:"check_status,attr"`
	PassCount    int                `xml:"pass_count,attr"`
	FailCount    int                `xml:"fail_count,attr"`
	Threads      []xmlCompactThread `xml:"thread,omitempty"`
	FailedChecks []xmlCheckRun      `xml:"failed_check,omitempty"`
}

type xmlCompactThread struct {
	File        string `xml:"file,attr"`
	Line        int    `xml:"line,attr"`
	Author      string `xml:"author,attr"`
	BodyPreview string `xml:"body_preview"`
}
