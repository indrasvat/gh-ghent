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
