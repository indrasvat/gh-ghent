package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/indrasvat/gh-ghent/internal/domain"
)

func sendKey(app App, key string) App {
	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
	return model.(App)
}

func sendSpecialKey(app App, keyType tea.KeyType) App {
	model, _ := app.Update(tea.KeyMsg{Type: keyType})
	return model.(App)
}

func sendWindowSize(app App, w, h int) App {
	model, _ := app.Update(tea.WindowSizeMsg{Width: w, Height: h})
	return model.(App)
}

func TestNewApp(t *testing.T) {
	app := NewApp("owner/repo", 42, ViewCommentsList)
	if app.ActiveView() != ViewCommentsList {
		t.Errorf("expected ViewCommentsList, got %v", app.ActiveView())
	}
	if app.Width() != 0 || app.Height() != 0 {
		t.Error("expected zero dimensions before WindowSizeMsg")
	}
}

func TestWindowSizeMsg(t *testing.T) {
	app := NewApp("owner/repo", 42, ViewCommentsList)
	app = sendWindowSize(app, 120, 40)
	if app.Width() != 120 {
		t.Errorf("expected width 120, got %d", app.Width())
	}
	if app.Height() != 40 {
		t.Errorf("expected height 40, got %d", app.Height())
	}
}

func TestWindowSizeMsgPropagatesAllSubModels(t *testing.T) {
	// When sub-models are added, this test should verify propagation.
	// For now, verify dimensions are stored on the root model from any view.
	views := []View{ViewCommentsList, ViewChecksList, ViewResolve, ViewSummary}
	for _, v := range views {
		app := NewApp("owner/repo", 42, v)
		app = sendWindowSize(app, 80, 24)
		if app.Width() != 80 || app.Height() != 24 {
			t.Errorf("view %v: expected 80x24, got %dx%d", v, app.Width(), app.Height())
		}
	}
}

func TestTabCyclesViews(t *testing.T) {
	app := NewApp("owner/repo", 42, ViewCommentsList)

	// Tab: comments → checks
	app = sendSpecialKey(app, tea.KeyTab)
	if app.ActiveView() != ViewChecksList {
		t.Errorf("expected ViewChecksList after Tab, got %v", app.ActiveView())
	}

	// Tab: checks → comments (wraps)
	app = sendSpecialKey(app, tea.KeyTab)
	if app.ActiveView() != ViewCommentsList {
		t.Errorf("expected ViewCommentsList after second Tab, got %v", app.ActiveView())
	}
}

func TestShiftTabCyclesReverse(t *testing.T) {
	app := NewApp("owner/repo", 42, ViewCommentsList)

	// Shift+Tab: comments → checks (reverse wraps)
	app = sendSpecialKey(app, tea.KeyShiftTab)
	if app.ActiveView() != ViewChecksList {
		t.Errorf("expected ViewChecksList after Shift+Tab, got %v", app.ActiveView())
	}

	// Shift+Tab: checks → comments
	app = sendSpecialKey(app, tea.KeyShiftTab)
	if app.ActiveView() != ViewCommentsList {
		t.Errorf("expected ViewCommentsList after second Shift+Tab, got %v", app.ActiveView())
	}
}

func TestEnterDrillsIntoDetail(t *testing.T) {
	// Comments list: Enter goes through sub-model → selectThreadMsg.
	// Need thread data so the sub-model has something to select.
	app := NewApp("owner/repo", 42, ViewCommentsList)
	app.SetComments(&domain.CommentsResult{
		Threads: []domain.ReviewThread{
			{ID: "t1", Path: "file.go", Line: 10, Comments: []domain.Comment{{Author: "alice", Body: "fix this"}}},
		},
		UnresolvedCount: 1,
	})
	app = sendWindowSize(app, 80, 24)

	// Enter on the comments list sub-model returns a selectThreadMsg cmd.
	model, cmd := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = model.(App)
	if cmd != nil {
		// Execute the command to get the selectThreadMsg.
		msg := cmd()
		model, _ = app.Update(msg)
		app = model.(App)
	}
	if app.ActiveView() != ViewCommentsExpand {
		t.Errorf("Enter from comments list: expected ViewCommentsExpand, got %v", app.ActiveView())
	}

	// Checks list: Enter goes through sub-model → selectCheckMsg.
	app2 := NewApp("owner/repo", 42, ViewChecksList)
	app2.SetChecks(&domain.ChecksResult{
		Checks: []domain.CheckRun{
			{ID: 1, Name: "build", Status: "completed", Conclusion: "success"},
		},
		PassCount: 1,
	})
	app2 = sendWindowSize(app2, 80, 24)
	model2, cmd2 := app2.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app2 = model2.(App)
	if cmd2 != nil {
		msg2 := cmd2()
		model2, _ = app2.Update(msg2)
		app2 = model2.(App)
	}
	if app2.ActiveView() != ViewChecksLog {
		t.Errorf("Enter from checks: expected ViewChecksLog, got %v", app2.ActiveView())
	}
}

func TestEscReturnsToList(t *testing.T) {
	tests := []struct {
		start    View
		expected View
	}{
		{ViewCommentsExpand, ViewCommentsList},
		{ViewChecksLog, ViewChecksList},
	}
	for _, tt := range tests {
		app := NewApp("owner/repo", 42, tt.start)
		app = sendSpecialKey(app, tea.KeyEscape)
		if app.ActiveView() != tt.expected {
			t.Errorf("Esc from %v: expected %v, got %v", tt.start, tt.expected, app.ActiveView())
		}
	}
}

func TestQuitSendsQuitCmd(t *testing.T) {
	app := NewApp("owner/repo", 42, ViewCommentsList)
	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if cmd == nil {
		t.Error("expected tea.Quit command on 'q', got nil")
	}
}

func TestSummaryShortcuts(t *testing.T) {
	tests := []struct {
		key      string
		expected View
	}{
		{"c", ViewCommentsList},
		{"k", ViewChecksList},
		{"r", ViewResolve},
	}
	for _, tt := range tests {
		app := NewApp("owner/repo", 42, ViewSummary)
		app = sendKey(app, tt.key)
		if app.ActiveView() != tt.expected {
			t.Errorf("key %q from Summary: expected %v, got %v", tt.key, tt.expected, app.ActiveView())
		}
	}
}

func TestEscFromSummaryReturnsToPrevView(t *testing.T) {
	app := NewApp("owner/repo", 42, ViewCommentsList)
	// Simulate navigating to summary via c from summary... instead, set directly
	app.activeView = ViewSummary
	app.prevView = ViewCommentsList
	app = sendSpecialKey(app, tea.KeyEscape)
	if app.ActiveView() != ViewCommentsList {
		t.Errorf("expected ViewCommentsList after Esc from Summary, got %v", app.ActiveView())
	}
}

func TestEscFromResolveReturnsToPrevView(t *testing.T) {
	app := NewApp("owner/repo", 42, ViewCommentsList)
	app.activeView = ViewResolve
	app.prevView = ViewChecksList
	app = sendSpecialKey(app, tea.KeyEscape)
	if app.ActiveView() != ViewChecksList {
		t.Errorf("expected ViewChecksList after Esc from Resolve, got %v", app.ActiveView())
	}
}

func TestTabFromDetailGoesToNextTopLevel(t *testing.T) {
	// If you're in comments-expand and press Tab, should go to checks list.
	// Start directly in CommentsExpand to avoid needing sub-model Enter flow.
	app := NewApp("owner/repo", 42, ViewCommentsList)
	app.activeView = ViewCommentsExpand
	if app.ActiveView() != ViewCommentsExpand {
		t.Fatalf("expected ViewCommentsExpand, got %v", app.ActiveView())
	}
	app = sendSpecialKey(app, tea.KeyTab)
	if app.ActiveView() != ViewChecksList {
		t.Errorf("Tab from CommentsExpand: expected ViewChecksList, got %v", app.ActiveView())
	}
}

func TestViewRenders(t *testing.T) {
	app := NewApp("owner/repo", 42, ViewCommentsList)
	app = sendWindowSize(app, 80, 24)

	output := app.View()
	if output == "" {
		t.Error("expected non-empty View output")
	}
	// Status bar should contain "ghent"
	if !strings.Contains(output, "ghent") {
		t.Error("missing 'ghent' in status bar")
	}
	// Should contain comments list content (empty list message or threads)
	if !strings.Contains(output, "No review threads") {
		t.Error("missing empty comments list message")
	}
}

func TestViewRendersEmpty(t *testing.T) {
	// Before WindowSizeMsg, View should return empty (no crash).
	app := NewApp("owner/repo", 42, ViewCommentsList)
	output := app.View()
	if output != "" {
		t.Errorf("expected empty View before WindowSizeMsg, got %q", output)
	}
}

func TestStatusBarShowsCommentCounts(t *testing.T) {
	app := NewApp("owner/repo", 42, ViewCommentsList)
	app.SetComments(&domain.CommentsResult{
		UnresolvedCount: 5,
		ResolvedCount:   2,
	})
	app = sendWindowSize(app, 120, 40)
	output := app.View()
	if !strings.Contains(output, "5 unresolved") {
		t.Error("missing unresolved count in status bar")
	}
	if !strings.Contains(output, "2 resolved") {
		t.Error("missing resolved count in status bar")
	}
}

func TestStatusBarShowsCheckCounts(t *testing.T) {
	app := NewApp("owner/repo", 42, ViewChecksList)
	app.SetChecks(&domain.ChecksResult{
		HeadSHA:   "abc1234567890",
		PassCount: 4,
		FailCount: 1,
	})
	app = sendWindowSize(app, 120, 40)
	output := app.View()
	if !strings.Contains(output, "abc1234") {
		t.Error("missing HEAD SHA in status bar")
	}
	if !strings.Contains(output, "4 passed") {
		t.Error("missing pass count in status bar")
	}
	if !strings.Contains(output, "1 failed") {
		t.Error("missing fail count in status bar")
	}
}

func TestHelpBarChangesPerView(t *testing.T) {
	app := NewApp("owner/repo", 42, ViewCommentsList)
	app = sendWindowSize(app, 120, 40)

	commentsOutput := app.View()

	app = sendSpecialKey(app, tea.KeyTab)
	checksOutput := app.View()

	// Comments help should contain "expand" (enter → expand)
	if !strings.Contains(commentsOutput, "expand") {
		t.Error("missing 'expand' in comments help bar")
	}
	// Checks help should contain "view logs"
	if !strings.Contains(checksOutput, "view logs") {
		t.Error("missing 'view logs' in checks help bar")
	}
}

func TestViewStringValues(t *testing.T) {
	tests := []struct {
		view     View
		expected string
	}{
		{ViewCommentsList, "comments"},
		{ViewCommentsExpand, "comments-expand"},
		{ViewChecksList, "checks"},
		{ViewChecksLog, "checks-log"},
		{ViewResolve, "resolve"},
		{ViewSummary, "summary"},
		{ViewWatch, "watch"},
		{View(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.view.String(); got != tt.expected {
			t.Errorf("View(%d).String() = %q, want %q", tt.view, got, tt.expected)
		}
	}
}

func TestSetData(t *testing.T) {
	app := NewApp("owner/repo", 42, ViewCommentsList)

	app.SetComments(&domain.CommentsResult{UnresolvedCount: 3})
	if app.comments == nil || app.comments.UnresolvedCount != 3 {
		t.Error("SetComments failed")
	}

	app.SetChecks(&domain.ChecksResult{PassCount: 5})
	if app.checks == nil || app.checks.PassCount != 5 {
		t.Error("SetChecks failed")
	}

	app.SetReviews([]domain.Review{{Author: "alice"}})
	if len(app.reviews) != 1 || app.reviews[0].Author != "alice" {
		t.Error("SetReviews failed")
	}
}

func TestEnterFromNonListViewIsNoOp(t *testing.T) {
	// Enter from summary/resolve should not change view.
	for _, v := range []View{ViewSummary, ViewResolve, ViewWatch} {
		app := NewApp("owner/repo", 42, v)
		app = sendSpecialKey(app, tea.KeyEnter)
		if app.ActiveView() != v {
			t.Errorf("Enter from %v should be no-op, got %v", v, app.ActiveView())
		}
	}
}

func TestEscFromTopLevelIsNoOp(t *testing.T) {
	for _, v := range []View{ViewCommentsList, ViewChecksList} {
		app := NewApp("owner/repo", 42, v)
		app = sendSpecialKey(app, tea.KeyEscape)
		if app.ActiveView() != v {
			t.Errorf("Esc from %v should be no-op, got %v", v, app.ActiveView())
		}
	}
}

func TestEscFromCommentsListReturnsToPrevView(t *testing.T) {
	// Simulate: summary → 'c' → comments list → Esc → back to summary.
	app := NewApp("owner/repo", 42, ViewSummary)
	app = sendKey(app, "c")
	if app.ActiveView() != ViewCommentsList {
		t.Fatalf("expected ViewCommentsList after 'c', got %v", app.ActiveView())
	}
	app = sendSpecialKey(app, tea.KeyEscape)
	if app.ActiveView() != ViewSummary {
		t.Errorf("Esc from comments list: expected ViewSummary, got %v", app.ActiveView())
	}
}

func TestEscFromChecksListReturnsToPrevView(t *testing.T) {
	// Simulate: summary → 'k' → checks list → Esc → back to summary.
	app := NewApp("owner/repo", 42, ViewSummary)
	app = sendKey(app, "k")
	if app.ActiveView() != ViewChecksList {
		t.Fatalf("expected ViewChecksList after 'k', got %v", app.ActiveView())
	}
	app = sendSpecialKey(app, tea.KeyEscape)
	if app.ActiveView() != ViewSummary {
		t.Errorf("Esc from checks list: expected ViewSummary, got %v", app.ActiveView())
	}
}

func TestEscRoundTripFromSummary(t *testing.T) {
	// Full round-trip: summary → c → esc → summary → k → esc → summary.
	app := NewApp("owner/repo", 42, ViewSummary)

	// summary → comments → back
	app = sendKey(app, "c")
	app = sendSpecialKey(app, tea.KeyEscape)
	if app.ActiveView() != ViewSummary {
		t.Fatalf("first round-trip: expected ViewSummary, got %v", app.ActiveView())
	}

	// summary → checks → back
	app = sendKey(app, "k")
	app = sendSpecialKey(app, tea.KeyEscape)
	if app.ActiveView() != ViewSummary {
		t.Errorf("second round-trip: expected ViewSummary, got %v", app.ActiveView())
	}
}

func TestFormatCount(t *testing.T) {
	tests := []struct {
		n     int
		label string
		want  string
	}{
		{5, "unresolved", "5 unresolved"},
		{1, "passed", "1 passed"},
		{0, "failed", "0 failed"},
		{12, "checks", "12 checks"},
	}
	for _, tt := range tests {
		got := formatCount(tt.n, tt.label)
		if got != tt.want {
			t.Errorf("formatCount(%d, %q) = %q, want %q", tt.n, tt.label, got, tt.want)
		}
	}
}

func TestStatusBarShowsThreadCount(t *testing.T) {
	app := NewApp("owner/repo", 42, ViewCommentsList)
	app.SetComments(&domain.CommentsResult{
		Threads: []domain.ReviewThread{
			{ID: "t1", Path: "a.go", Line: 10, Comments: []domain.Comment{{Author: "alice", Body: "fix"}}},
			{ID: "t2", Path: "b.go", Line: 20, Comments: []domain.Comment{{Author: "bob", Body: "nit"}}},
		},
		UnresolvedCount: 2,
	})
	app = sendWindowSize(app, 120, 40)

	// Enter to expand thread 0.
	model, cmd := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = model.(App)
	if cmd != nil {
		msg := cmd()
		model, _ = app.Update(msg)
		app = model.(App)
	}
	if app.ActiveView() != ViewCommentsExpand {
		t.Fatalf("expected ViewCommentsExpand, got %v", app.ActiveView())
	}

	output := app.View()
	if !strings.Contains(output, "Thread 1 of 2") {
		t.Error("missing 'Thread 1 of 2' in expanded view status bar")
	}
}

func TestExpandedViewRendersContent(t *testing.T) {
	app := NewApp("owner/repo", 42, ViewCommentsList)
	app.SetComments(&domain.CommentsResult{
		Threads: []domain.ReviewThread{
			{
				ID: "t1", Path: "internal/api.go", Line: 42,
				Comments: []domain.Comment{
					{Author: "reviewer1", Body: "This needs error wrapping."},
					{Author: "you", Body: "Done!"},
				},
			},
		},
		UnresolvedCount: 1,
	})
	app = sendWindowSize(app, 120, 40)

	// Enter to expand.
	model, cmd := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = model.(App)
	if cmd != nil {
		msg := cmd()
		model, _ = app.Update(msg)
		app = model.(App)
	}

	output := app.View()
	if !strings.Contains(output, "internal/api.go") {
		t.Error("missing file path in expanded view")
	}
	if !strings.Contains(output, ":42") {
		t.Error("missing line number in expanded view")
	}
	if !strings.Contains(output, "@reviewer1") {
		t.Error("missing author in expanded view")
	}
	if !strings.Contains(output, "error wrapping") {
		t.Error("missing comment body in expanded view")
	}
}

func TestEscFromExpandedReturnsToList(t *testing.T) {
	app := NewApp("owner/repo", 42, ViewCommentsList)
	app.SetComments(&domain.CommentsResult{
		Threads: []domain.ReviewThread{
			{ID: "t1", Path: "a.go", Line: 10, Comments: []domain.Comment{{Author: "alice", Body: "fix"}}},
		},
		UnresolvedCount: 1,
	})
	app = sendWindowSize(app, 120, 40)

	// Enter to expand.
	model, cmd := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = model.(App)
	if cmd != nil {
		msg := cmd()
		model, _ = app.Update(msg)
		app = model.(App)
	}
	if app.ActiveView() != ViewCommentsExpand {
		t.Fatalf("expected expanded, got %v", app.ActiveView())
	}

	// Esc to return.
	app = sendSpecialKey(app, tea.KeyEscape)
	if app.ActiveView() != ViewCommentsList {
		t.Errorf("expected ViewCommentsList after Esc, got %v", app.ActiveView())
	}
}

func TestEscFromResolveConfirmingCancelsNotSwitchesView(t *testing.T) {
	// Enter resolve from summary (r key), so prevView = ViewSummary.
	app := NewApp("owner/repo", 42, ViewSummary)
	app.SetComments(&domain.CommentsResult{
		Threads: []domain.ReviewThread{
			makeThread("PRRT_aaa", "main.go", 10, true),
		},
		UnresolvedCount: 1,
	})
	app = sendWindowSize(app, 100, 30)

	// Navigate to resolve via 'r' from summary.
	app = sendKey(app, "r")
	if app.ActiveView() != ViewResolve {
		t.Fatalf("expected ViewResolve after 'r', got %v", app.ActiveView())
	}

	// Space to select thread.
	app = sendKey(app, " ")
	// Enter to start confirmation.
	app = sendSpecialKey(app, tea.KeyEnter)
	if app.resolve.state != resolveStateConfirming {
		t.Fatalf("expected confirming state, got %d", app.resolve.state)
	}

	// Esc should cancel confirmation, NOT switch view.
	app = sendSpecialKey(app, tea.KeyEscape)
	if app.ActiveView() != ViewResolve {
		t.Errorf("Esc from confirming: expected ViewResolve, got %v", app.ActiveView())
	}
	if app.resolve.state != resolveStateBrowsing {
		t.Errorf("Esc from confirming: expected browsing state, got %d", app.resolve.state)
	}

	// Esc again (from browsing) should switch back to summary.
	app = sendSpecialKey(app, tea.KeyEscape)
	if app.ActiveView() != ViewSummary {
		t.Errorf("Esc from browsing: expected ViewSummary, got %v", app.ActiveView())
	}
}

func TestSetAsyncFetchMarksLoading(t *testing.T) {
	app := NewApp("owner/repo", 42, ViewSummary)
	app.SetAsyncFetch(
		func() (*domain.CommentsResult, error) { return nil, nil },
		func() (*domain.ChecksResult, error) { return nil, nil },
		func() ([]domain.Review, error) { return nil, nil },
	)
	if !app.commentsLoading || !app.checksLoading || !app.reviewsLoading {
		t.Error("expected all loading flags to be true after SetAsyncFetch")
	}
	if !app.isLoading() {
		t.Error("expected isLoading() to be true")
	}
	if !app.summary.loading {
		t.Error("expected summary.loading to be true")
	}
}

func TestAsyncInitReturnsCommands(t *testing.T) {
	app := NewApp("owner/repo", 42, ViewSummary)
	app.SetAsyncFetch(
		func() (*domain.CommentsResult, error) {
			return &domain.CommentsResult{UnresolvedCount: 3}, nil
		},
		func() (*domain.ChecksResult, error) {
			return &domain.ChecksResult{PassCount: 5}, nil
		},
		func() ([]domain.Review, error) {
			return []domain.Review{{Author: "alice", State: domain.ReviewApproved}}, nil
		},
	)

	cmd := app.Init()
	if cmd == nil {
		t.Fatal("expected Init() to return a command for async fetches")
	}
}

func TestAsyncLoadedMsgUpdatesData(t *testing.T) {
	app := NewApp("owner/repo", 42, ViewSummary)
	app.commentsLoading = true
	app.checksLoading = true
	app.reviewsLoading = true
	app.summary.loading = true
	app = sendWindowSize(app, 80, 24)

	// Simulate commentsLoadedMsg.
	model, _ := app.Update(commentsLoadedMsg{
		result: &domain.CommentsResult{UnresolvedCount: 2},
	})
	app = model.(App)
	if app.commentsLoading {
		t.Error("commentsLoading should be false after commentsLoadedMsg")
	}
	if app.comments == nil || app.comments.UnresolvedCount != 2 {
		t.Error("comments data not set after commentsLoadedMsg")
	}

	// Simulate checksLoadedMsg.
	model, _ = app.Update(checksLoadedMsg{
		result: &domain.ChecksResult{PassCount: 4},
	})
	app = model.(App)
	if app.checksLoading {
		t.Error("checksLoading should be false after checksLoadedMsg")
	}
	if app.checks == nil || app.checks.PassCount != 4 {
		t.Error("checks data not set after checksLoadedMsg")
	}

	// Simulate reviewsLoadedMsg.
	model, _ = app.Update(reviewsLoadedMsg{
		reviews: []domain.Review{{Author: "bob", State: domain.ReviewApproved}},
	})
	app = model.(App)
	if app.reviewsLoading {
		t.Error("reviewsLoading should be false after reviewsLoadedMsg")
	}
	if len(app.reviews) != 1 || app.reviews[0].Author != "bob" {
		t.Error("reviews data not set after reviewsLoadedMsg")
	}
	if app.isLoading() {
		t.Error("expected isLoading() to be false after all data loaded")
	}
	if app.summary.loading {
		t.Error("expected summary.loading to be false after all data loaded")
	}
}

func TestAsyncLoadErrorIsRecorded(t *testing.T) {
	app := NewApp("owner/repo", 42, ViewSummary)
	app.commentsLoading = true
	app.summary.loading = true

	model, _ := app.Update(commentsLoadedMsg{
		err: fmt.Errorf("network error"),
	})
	app = model.(App)
	if len(app.loadErrors) == 0 {
		t.Fatal("expected load error to be recorded")
	}
	if !strings.Contains(app.loadErrors[0], "network error") {
		t.Errorf("expected error message to contain 'network error', got %q", app.loadErrors[0])
	}
}

func TestTruncateSHA(t *testing.T) {
	tests := []struct {
		sha  string
		want string
	}{
		{"abc1234567890", "abc1234"},
		{"short", "short"},
		{"", ""},
	}
	for _, tt := range tests {
		got := truncateSHA(tt.sha)
		if got != tt.want {
			t.Errorf("truncateSHA(%q) = %q, want %q", tt.sha, got, tt.want)
		}
	}
}
