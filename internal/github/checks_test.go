package github

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/indrasvat/ghent/internal/domain"
)

func loadCheckRunsFixture(t *testing.T, path string) checkRunsResponse {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", path, err)
	}
	var resp checkRunsResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("unmarshal fixture %s: %v", path, err)
	}
	return resp
}

func loadAnnotationsFixture(t *testing.T, path string) []annotationNode {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", path, err)
	}
	var resp []annotationNode
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("unmarshal fixture %s: %v", path, err)
	}
	return resp
}

func TestCheckRunsParsing(t *testing.T) {
	resp := loadCheckRunsFixture(t, "../../testdata/rest/check_runs.json")

	if resp.TotalCount != 4 {
		t.Errorf("TotalCount = %d, want 4", resp.TotalCount)
	}
	if len(resp.CheckRuns) != 4 {
		t.Fatalf("len(CheckRuns) = %d, want 4", len(resp.CheckRuns))
	}

	// Verify first check run (Lint — failure)
	lint := resp.CheckRuns[0]
	if lint.Name != "Lint" {
		t.Errorf("CheckRuns[0].Name = %q, want %q", lint.Name, "Lint")
	}
	if lint.Status != "completed" {
		t.Errorf("CheckRuns[0].Status = %q, want %q", lint.Status, "completed")
	}
	if lint.Conclusion == nil || *lint.Conclusion != "failure" {
		t.Errorf("CheckRuns[0].Conclusion = %v, want %q", lint.Conclusion, "failure")
	}
	if lint.Output.AnnotationsCount != 2 {
		t.Errorf("CheckRuns[0].Output.AnnotationsCount = %d, want 2", lint.Output.AnnotationsCount)
	}

	// Verify in-progress check (Build — null conclusion)
	build := resp.CheckRuns[2]
	if build.Name != "Build" {
		t.Errorf("CheckRuns[2].Name = %q, want %q", build.Name, "Build")
	}
	if build.Conclusion != nil {
		t.Errorf("CheckRuns[2].Conclusion = %v, want nil", build.Conclusion)
	}
}

func TestAnnotationsParsing(t *testing.T) {
	annots := loadAnnotationsFixture(t, "../../testdata/rest/annotations.json")

	if len(annots) != 2 {
		t.Fatalf("len(annotations) = %d, want 2", len(annots))
	}

	want := []annotationNode{
		{
			Path:            "internal/api/handler.go",
			StartLine:       42,
			EndLine:         42,
			AnnotationLevel: "failure",
			Title:           "golangci-lint",
			Message:         "unused variable 'x' (deadcode)",
		},
		{
			Path:            "internal/api/handler.go",
			StartLine:       55,
			EndLine:         57,
			AnnotationLevel: "warning",
			Title:           "golangci-lint",
			Message:         "function 'processData' is too complex (cyclomatic complexity 15)",
		},
	}

	if diff := cmp.Diff(want, annots); diff != "" {
		t.Errorf("annotations mismatch (-want +got):\n%s", diff)
	}
}

func TestClassifyCheckStatus(t *testing.T) {
	tests := []struct {
		name       string
		status     string
		conclusion string
		want       domain.OverallStatus
	}{
		{name: "completed success", status: "completed", conclusion: "success", want: domain.StatusPass},
		{name: "completed failure", status: "completed", conclusion: "failure", want: domain.StatusFail},
		{name: "completed neutral", status: "completed", conclusion: "neutral", want: domain.StatusPass},
		{name: "completed skipped", status: "completed", conclusion: "skipped", want: domain.StatusPass},
		{name: "completed cancelled", status: "completed", conclusion: "cancelled", want: domain.StatusFail},
		{name: "completed timed_out", status: "completed", conclusion: "timed_out", want: domain.StatusFail},
		{name: "completed action_required", status: "completed", conclusion: "action_required", want: domain.StatusFail},
		{name: "completed startup_failure", status: "completed", conclusion: "startup_failure", want: domain.StatusFail},
		{name: "completed stale", status: "completed", conclusion: "stale", want: domain.StatusFail},
		{name: "in_progress", status: "in_progress", conclusion: "", want: domain.StatusPending},
		{name: "queued", status: "queued", conclusion: "", want: domain.StatusPending},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyCheckStatus(tt.status, tt.conclusion)
			if got != tt.want {
				t.Errorf("classifyCheckStatus(%q, %q) = %q, want %q", tt.status, tt.conclusion, got, tt.want)
			}
		})
	}
}

func TestAggregateStatus(t *testing.T) {
	tests := []struct {
		name     string
		statuses []domain.OverallStatus
		want     domain.OverallStatus
	}{
		{name: "all pass", statuses: []domain.OverallStatus{domain.StatusPass, domain.StatusPass}, want: domain.StatusPass},
		{name: "one fail", statuses: []domain.OverallStatus{domain.StatusPass, domain.StatusFail}, want: domain.StatusFail},
		{name: "one pending", statuses: []domain.OverallStatus{domain.StatusPass, domain.StatusPending}, want: domain.StatusPending},
		{name: "fail beats pending", statuses: []domain.OverallStatus{domain.StatusPending, domain.StatusFail}, want: domain.StatusFail},
		{name: "empty", statuses: nil, want: domain.StatusPass},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := domain.AggregateStatus(tt.statuses)
			if got != tt.want {
				t.Errorf("AggregateStatus = %q, want %q", got, tt.want)
			}
		})
	}
}
