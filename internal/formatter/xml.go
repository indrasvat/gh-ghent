package formatter

import (
	"encoding/xml"
	"io"
	"time"

	"github.com/indrasvat/ghent/internal/domain"
)

// XMLFormatter outputs results as XML.
type XMLFormatter struct{}

func (f *XMLFormatter) FormatComments(w io.Writer, result *domain.CommentsResult) error {
	out := xmlComments{
		PRNumber:        result.PRNumber,
		TotalCount:      result.TotalCount,
		ResolvedCount:   result.ResolvedCount,
		UnresolvedCount: result.UnresolvedCount,
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

func (f *XMLFormatter) FormatChecks(w io.Writer, result *domain.ChecksResult) error {
	out := xmlChecks{
		PRNumber:      result.PRNumber,
		HeadSHA:       result.HeadSHA,
		OverallStatus: string(result.OverallStatus),
		PassCount:     result.PassCount,
		FailCount:     result.FailCount,
		PendingCount:  result.PendingCount,
	}
	for _, ch := range result.Checks {
		xc := xmlCheckRun{
			ID:         ch.ID,
			Name:       ch.Name,
			Status:     ch.Status,
			Conclusion: ch.Conclusion,
			HTMLURL:    ch.HTMLURL,
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
	if _, err := io.WriteString(w, xml.Header); err != nil {
		return err
	}
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	return enc.Encode(result)
}

type xmlComments struct {
	XMLName         xml.Name    `xml:"comments"`
	PRNumber        int         `xml:"pr_number,attr"`
	TotalCount      int         `xml:"total_count,attr"`
	ResolvedCount   int         `xml:"resolved_count,attr"`
	UnresolvedCount int         `xml:"unresolved_count,attr"`
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
	Checks        []xmlCheckRun `xml:"check"`
}

type xmlCheckRun struct {
	ID          int64           `xml:"id,attr"`
	Name        string          `xml:"name,attr"`
	Status      string          `xml:"status,attr"`
	Conclusion  string          `xml:"conclusion,attr"`
	HTMLURL     string          `xml:"html_url,attr"`
	Annotations []xmlAnnotation `xml:"annotation,omitempty"`
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
