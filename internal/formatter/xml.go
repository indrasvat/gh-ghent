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
	if _, err := io.WriteString(w, xml.Header); err != nil {
		return err
	}
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	return enc.Encode(result)
}

func (f *XMLFormatter) FormatReply(w io.Writer, result *domain.ReplyResult) error {
	if _, err := io.WriteString(w, xml.Header); err != nil {
		return err
	}
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	return enc.Encode(result)
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
