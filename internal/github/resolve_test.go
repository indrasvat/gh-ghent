package github

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/indrasvat/gh-ghent/internal/domain"
)

func TestResolveResponseMapping(t *testing.T) {
	data, err := os.ReadFile("../../testdata/graphql/resolve_thread.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	var envelope struct {
		Data resolveResponse `json:"data"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}

	thread := envelope.Data.ResolveReviewThread.Thread
	got := &domain.ResolveResult{
		ThreadID:   thread.ID,
		Path:       thread.Path,
		Line:       thread.Line,
		IsResolved: thread.IsResolved,
		Action:     "resolved",
	}

	want := &domain.ResolveResult{
		ThreadID:   "PRRT_thread1",
		Path:       "internal/api/graphql.go",
		Line:       47,
		IsResolved: true,
		Action:     "resolved",
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("resolve result mismatch (-want +got):\n%s", diff)
	}
}

func TestUnresolveResponseMapping(t *testing.T) {
	data, err := os.ReadFile("../../testdata/graphql/unresolve_thread.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	var envelope struct {
		Data unresolveResponse `json:"data"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}

	thread := envelope.Data.UnresolveReviewThread.Thread
	got := &domain.ResolveResult{
		ThreadID:   thread.ID,
		Path:       thread.Path,
		Line:       thread.Line,
		IsResolved: thread.IsResolved,
		Action:     "unresolved",
	}

	want := &domain.ResolveResult{
		ThreadID:   "PRRT_thread1",
		Path:       "internal/api/graphql.go",
		Line:       47,
		IsResolved: false,
		Action:     "unresolved",
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unresolve result mismatch (-want +got):\n%s", diff)
	}
}

func TestResolveResponseFields(t *testing.T) {
	data, err := os.ReadFile("../../testdata/graphql/resolve_thread.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	var envelope struct {
		Data resolveResponse `json:"data"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}

	thread := envelope.Data.ResolveReviewThread.Thread
	if !thread.IsResolved {
		t.Error("expected isResolved=true after resolve mutation")
	}
	if thread.ID == "" {
		t.Error("expected non-empty thread ID")
	}
	if thread.Path == "" {
		t.Error("expected non-empty path")
	}
}

func TestUnresolveResponseFields(t *testing.T) {
	data, err := os.ReadFile("../../testdata/graphql/unresolve_thread.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	var envelope struct {
		Data unresolveResponse `json:"data"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}

	thread := envelope.Data.UnresolveReviewThread.Thread
	if thread.IsResolved {
		t.Error("expected isResolved=false after unresolve mutation")
	}
}
