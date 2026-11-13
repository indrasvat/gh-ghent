package github

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/indrasvat/gh-ghent/internal/domain"
)

const (
	fbSHA     = "6a8d6cb53897062db479990c5ad0f3b731d9bea0"
	fbPR      = "repos/o/r/pulls/1"
	fbChecks  = "repos/o/r/commits/" + fbSHA + "/check-runs?per_page=100&page=1"
	fbRuns    = "repos/o/r/actions/runs?head_sha=" + fbSHA + "&per_page=100&page=1"
	fbJobs    = "repos/o/r/actions/runs/43348095280/jobs?per_page=100&page=1"
	fbStatus  = "repos/o/r/commits/" + fbSHA + "/status?per_page=100&page=1"
	denied403 = `{"message":"Resource not accessible by integration","status":"403"}`
)

func fixture(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile("../../testdata/rest/fallback/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func fallbackRoutes(t *testing.T) map[string]fakeResponse {
	return map[string]fakeResponse{
		fbPR:     {200, `{"head":{"sha":"` + fbSHA + `"}}`},
		fbChecks: {403, denied403},
		fbRuns:   {200, fixture(t, "workflow_runs.json")},
		fbJobs:   {200, fixture(t, "jobs.json")},
		fbStatus: {200, fixture(t, "combined_status.json")},
	}
}

// Responses recorded from a private repo whose Actions token had no `checks`
// permission: check-runs is a 403, Actions jobs and commit statuses are readable.
func TestFetchChecksFallsBackWhenCheckRunsDenied(t *testing.T) {
	c, _ := newFakeClient(t, fallbackRoutes(t))

	got, err := c.FetchChecks(context.Background(), "o", "r", 1)
	if err != nil {
		t.Fatalf("FetchChecks: %v", err)
	}
	if !strings.Contains(got.Fallback, "Resource not accessible by integration") ||
		!strings.Contains(got.Fallback, "Actions jobs and commit statuses") {
		t.Errorf("Fallback = %q, want the 403 reason and what replaced check runs", got.Fallback)
	}

	type row struct {
		ID         int64
		Name       string
		Status     string
		Conclusion string
	}
	var rows []row
	for _, ch := range got.Checks {
		rows = append(rows, row{ch.ID, ch.Name, ch.Status, ch.Conclusion})
	}
	want := []row{
		// Job IDs double as check-run IDs, so --logs keeps working.
		{131632216881, "passes", "completed", "success"},
		{131632217242, "fails", "completed", "failure"},
		{131632266549, "probe", "in_progress", ""},
		// Real status IDs: watch mode tells checks apart by ID.
		{58799506255, "preview/deploy", "completed", "success"},
		// A status in "error" is a failure, not something to wait on.
		{58799506256, "deploy/prod", "completed", "failure"},
	}
	if diff := cmp.Diff(want, rows); diff != "" {
		t.Errorf("checks (-want +got):\n%s", diff)
	}
	if got.OverallStatus != domain.StatusFail || got.PassCount != 2 || got.FailCount != 2 || got.PendingCount != 1 {
		t.Errorf("overall=%s pass=%d fail=%d pending=%d, want failure 2/2/1",
			got.OverallStatus, got.PassCount, got.FailCount, got.PendingCount)
	}
}

func TestFetchChecksFallbackDeniedTooIsLoud(t *testing.T) {
	routes := fallbackRoutes(t)
	routes[fbRuns] = fakeResponse{403, `{"message":"Resource not accessible by personal access token"}`}
	c, _ := newFakeClient(t, routes)

	_, err := c.FetchChecks(context.Background(), "o", "r", 1)
	var perm *PermissionError
	if !errors.As(err, &perm) {
		t.Fatalf("err = %v (%T), want *PermissionError", err, err)
	}
	for _, w := range []string{"check runs", "Actions", "actions: read"} {
		if !strings.Contains(err.Error(), w) {
			t.Errorf("error %q missing %q", err, w)
		}
	}
}

// Only a permission denial means "this token can't see checks". Anything
// else must surface as-is, not be papered over by a partial fallback.
func TestFetchChecksNoFallbackOnOtherErrors(t *testing.T) {
	old := retryBackoff
	retryBackoff = 0
	t.Cleanup(func() { retryBackoff = old })

	routes := fallbackRoutes(t)
	routes[fbChecks] = fakeResponse{500, `{"message":"boom"}`}
	c, f := newFakeClient(t, routes)

	if _, err := c.FetchChecks(context.Background(), "o", "r", 1); err == nil {
		t.Fatal("want error on 500, got nil")
	}
	for _, call := range f.calls {
		if strings.Contains(call, "actions/runs") {
			t.Errorf("fell back on a 500: called %s", call)
		}
	}
}

// Watch output streams to agents too; the note must not stop at the final report.
func TestWatchStatusCarriesFallback(t *testing.T) {
	result := &domain.ChecksResult{OverallStatus: domain.StatusPass, Fallback: "check runs denied"}
	got := buildWatchStatus(time.Now(), result, map[int64]string{})
	if got.Fallback != result.Fallback {
		t.Errorf("WatchStatus.Fallback = %q, want %q", got.Fallback, result.Fallback)
	}
}

// GitHub can list a queued run before it has any jobs. Dropping it would let
// the fallback report "pass" (and --watch stop) while that workflow is unstarted.
func TestFetchChecksFallbackKeepsRunsWithoutJobs(t *testing.T) {
	routes := fallbackRoutes(t)
	routes[fbRuns] = fakeResponse{200, `{"total_count":2,"workflow_runs":[` +
		`{"id":43348095280,"name":"ci","status":"completed","conclusion":"success","html_url":"u1"},` +
		`{"id":77,"name":"deploy","status":"queued","conclusion":null,"html_url":"https://github.com/o/r/actions/runs/77"}]}`}
	routes["repos/o/r/actions/runs/77/jobs?per_page=100&page=1"] = fakeResponse{200, `{"total_count":0,"jobs":[]}`}
	routes[fbStatus] = fakeResponse{200, `{"total_count":0,"statuses":[]}`}
	routes[fbJobs] = fakeResponse{200, `{"total_count":1,"jobs":[{"id":1,"name":"build","status":"completed","conclusion":"success","started_at":"2026-11-13T01:52:39Z","completed_at":"2026-11-13T01:52:44Z"}]}`}
	c, _ := newFakeClient(t, routes)

	got, err := c.FetchChecks(context.Background(), "o", "r", 1)
	if err != nil {
		t.Fatalf("FetchChecks: %v", err)
	}
	if got.OverallStatus != domain.StatusPending || got.PendingCount != 1 {
		t.Fatalf("overall=%s pending=%d, want pending with the job-less run counted", got.OverallStatus, got.PendingCount)
	}
	var found bool
	for _, ch := range got.Checks {
		if ch.Name == "deploy" && ch.Status == "queued" && ch.HTMLURL == "https://github.com/o/r/actions/runs/77" {
			found = true
		}
	}
	if !found {
		t.Errorf("job-less queued run missing from checks: %+v", got.Checks)
	}
}
