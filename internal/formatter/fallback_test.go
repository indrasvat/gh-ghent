package formatter

import (
	"bytes"
	"strings"
	"testing"

	"github.com/indrasvat/gh-ghent/internal/domain"
)

// When checks came from the Actions/statuses fallback, every output an agent
// may read has to say so: the check list is knowingly incomplete.
func TestChecksFallbackIsVisibleInEveryFormat(t *testing.T) {
	const note = "check runs are not readable with this token"
	checks := domain.ChecksResult{PRNumber: 1, OverallStatus: domain.StatusPass, Fallback: note}
	status := &domain.StatusResult{PRNumber: 1, Checks: checks}

	for _, format := range []string{"json", "md", "xml"} {
		f, err := New(format)
		if err != nil {
			t.Fatal(err)
		}
		outputs := map[string]func(*bytes.Buffer) error{
			"checks":         func(b *bytes.Buffer) error { return f.FormatChecks(b, &checks) },
			"status":         func(b *bytes.Buffer) error { return f.FormatStatus(b, status) },
			"compact status": func(b *bytes.Buffer) error { return f.FormatCompactStatus(b, status) },
			"watch status":   func(b *bytes.Buffer) error { return f.FormatWatchStatus(b, &domain.WatchStatus{Fallback: note}) },
		}
		for name, render := range outputs {
			t.Run(format+"/"+name, func(t *testing.T) {
				var b bytes.Buffer
				if err := render(&b); err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(b.String(), note) {
					t.Errorf("fallback note missing from output:\n%s", b.String())
				}
			})
		}
	}
}
