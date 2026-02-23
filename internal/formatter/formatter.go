// Package formatter provides pipe-mode output formatters (JSON, XML, Markdown).
package formatter

import (
	"fmt"

	"github.com/indrasvat/ghent/internal/domain"
)

// New creates a formatter for the given format string.
func New(format string) (domain.Formatter, error) {
	switch format {
	case "json":
		return &JSONFormatter{}, nil
	case "md", "markdown":
		return &MarkdownFormatter{}, nil
	case "xml":
		return &XMLFormatter{}, nil
	default:
		return nil, fmt.Errorf("unsupported format: %q", format)
	}
}
