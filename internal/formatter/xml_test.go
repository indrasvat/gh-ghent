package formatter

import (
	"bytes"
	"encoding/xml"
	"strings"
	"testing"
)

func TestXMLFormatterWellFormed(t *testing.T) {
	var buf bytes.Buffer
	f := &XMLFormatter{}

	if err := f.FormatComments(&buf, sampleCommentsResult()); err != nil {
		t.Fatalf("FormatComments: %v", err)
	}

	// Must be well-formed XML
	var v xmlComments
	if err := xml.Unmarshal(buf.Bytes(), &v); err != nil {
		t.Fatalf("invalid XML: %v\noutput:\n%s", err, buf.String())
	}
}

func TestXMLFormatterFields(t *testing.T) {
	var buf bytes.Buffer
	f := &XMLFormatter{}

	if err := f.FormatComments(&buf, sampleCommentsResult()); err != nil {
		t.Fatalf("FormatComments: %v", err)
	}

	var v xmlComments
	if err := xml.Unmarshal(buf.Bytes(), &v); err != nil {
		t.Fatalf("invalid XML: %v", err)
	}

	if v.PRNumber != 42 {
		t.Errorf("PRNumber = %d, want 42", v.PRNumber)
	}
	if v.UnresolvedCount != 2 {
		t.Errorf("UnresolvedCount = %d, want 2", v.UnresolvedCount)
	}
	if len(v.Threads) != 1 {
		t.Fatalf("len(Threads) = %d, want 1", len(v.Threads))
	}
	if v.Threads[0].Path != "main.go" {
		t.Errorf("thread.Path = %q, want %q", v.Threads[0].Path, "main.go")
	}
	if len(v.Threads[0].Comments) != 1 {
		t.Fatalf("len(Comments) = %d, want 1", len(v.Threads[0].Comments))
	}
	if v.Threads[0].Comments[0].Author != "alice" {
		t.Errorf("comment.Author = %q, want %q", v.Threads[0].Comments[0].Author, "alice")
	}
}

func TestXMLFormatterNoANSI(t *testing.T) {
	var buf bytes.Buffer
	f := &XMLFormatter{}

	if err := f.FormatComments(&buf, sampleCommentsResult()); err != nil {
		t.Fatalf("FormatComments: %v", err)
	}

	if strings.Contains(buf.String(), "\033") {
		t.Error("XML output contains ANSI escape sequences")
	}
}

func TestXMLResolveResultsWellFormed(t *testing.T) {
	var buf bytes.Buffer
	f := &XMLFormatter{}

	if err := f.FormatResolveResults(&buf, sampleResolveResults()); err != nil {
		t.Fatalf("FormatResolveResults: %v", err)
	}

	var v xmlResolveResults
	if err := xml.Unmarshal(buf.Bytes(), &v); err != nil {
		t.Fatalf("invalid XML: %v\noutput:\n%s", err, buf.String())
	}
}

func TestXMLResolveResultsFields(t *testing.T) {
	var buf bytes.Buffer
	f := &XMLFormatter{}

	if err := f.FormatResolveResults(&buf, sampleResolveResults()); err != nil {
		t.Fatalf("FormatResolveResults: %v", err)
	}

	var v xmlResolveResults
	if err := xml.Unmarshal(buf.Bytes(), &v); err != nil {
		t.Fatalf("invalid XML: %v", err)
	}

	if v.SuccessCount != 2 {
		t.Errorf("SuccessCount = %d, want 2", v.SuccessCount)
	}
	if v.FailureCount != 1 {
		t.Errorf("FailureCount = %d, want 1", v.FailureCount)
	}
	if len(v.Results) != 2 {
		t.Fatalf("len(Results) = %d, want 2", len(v.Results))
	}
	if v.Results[0].ThreadID != "PRRT_1" {
		t.Errorf("result[0].ThreadID = %q, want %q", v.Results[0].ThreadID, "PRRT_1")
	}
	if v.Results[0].Action != "resolved" {
		t.Errorf("result[0].Action = %q, want %q", v.Results[0].Action, "resolved")
	}
	if len(v.Errors) != 1 {
		t.Fatalf("len(Errors) = %d, want 1", len(v.Errors))
	}
	if v.Errors[0].ThreadID != "PRRT_3" {
		t.Errorf("error[0].ThreadID = %q, want %q", v.Errors[0].ThreadID, "PRRT_3")
	}
}

func TestXMLFormatterHasHeader(t *testing.T) {
	var buf bytes.Buffer
	f := &XMLFormatter{}

	if err := f.FormatComments(&buf, sampleCommentsResult()); err != nil {
		t.Fatalf("FormatComments: %v", err)
	}

	if !strings.HasPrefix(buf.String(), "<?xml") {
		t.Error("XML output missing XML declaration header")
	}
}
