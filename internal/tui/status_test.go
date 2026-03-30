package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/indrasvat/gh-ghent/internal/domain"
)

func TestStatusEmptyView(t *testing.T) {
	m := statusModel{}
	m.setSize(100, 30)
	view := m.View()
	if !strings.Contains(view, "No review threads") {
		t.Error("missing empty threads message")
	}
	if !strings.Contains(view, "No CI checks") {
		t.Error("missing empty checks message")
	}
	if !strings.Contains(view, "No reviews yet") {
		t.Error("missing empty reviews message")
	}
}

func TestStatusKPICards(t *testing.T) {
	m := statusModel{
		comments: &domain.CommentsResult{UnresolvedCount: 3, ResolvedCount: 1},
		checks:   &domain.ChecksResult{PassCount: 4, FailCount: 1},
		reviews: []domain.Review{
			{Author: "alice", State: domain.ReviewApproved},
		},
	}
	m.setSize(120, 30)
	view := m.View()

	// Check that card counts appear.
	if !strings.Contains(view, "3") {
		t.Error("missing unresolved count 3")
	}
	if !strings.Contains(view, "4") {
		t.Error("missing pass count 4")
	}
	if !strings.Contains(view, "UNRESOLVED") {
		t.Error("missing UNRESOLVED label")
	}
	if !strings.Contains(view, "PASSED") {
		t.Error("missing PASSED label")
	}
	if !strings.Contains(view, "FAILED") {
		t.Error("missing FAILED label")
	}
	if !strings.Contains(view, "APPROVALS") {
		t.Error("missing APPROVALS label")
	}
}

func TestStatusMergeReady(t *testing.T) {
	tests := []struct {
		name     string
		model    statusModel
		expected bool
	}{
		{
			name: "ready: no unresolved, checks pass, approved",
			model: statusModel{
				comments: &domain.CommentsResult{UnresolvedCount: 0},
				checks:   &domain.ChecksResult{OverallStatus: domain.StatusPass},
				reviews:  []domain.Review{{State: domain.ReviewApproved}},
			},
			expected: true,
		},
		{
			name: "not ready: unresolved threads",
			model: statusModel{
				comments: &domain.CommentsResult{UnresolvedCount: 2},
				checks:   &domain.ChecksResult{OverallStatus: domain.StatusPass},
				reviews:  []domain.Review{{State: domain.ReviewApproved}},
			},
			expected: false,
		},
		{
			name: "not ready: checks failing",
			model: statusModel{
				comments: &domain.CommentsResult{UnresolvedCount: 0},
				checks:   &domain.ChecksResult{OverallStatus: domain.StatusFail},
				reviews:  []domain.Review{{State: domain.ReviewApproved}},
			},
			expected: false,
		},
		{
			name: "not ready: changes requested",
			model: statusModel{
				comments: &domain.CommentsResult{UnresolvedCount: 0},
				checks:   &domain.ChecksResult{OverallStatus: domain.StatusPass},
				reviews:  []domain.Review{{State: domain.ReviewChangesRequested}},
			},
			expected: false,
		},
		{
			name: "not ready: no approval",
			model: statusModel{
				comments: &domain.CommentsResult{UnresolvedCount: 0},
				checks:   &domain.ChecksResult{OverallStatus: domain.StatusPass},
				reviews:  []domain.Review{{State: domain.ReviewCommented}},
			},
			expected: false,
		},
		{
			name: "ready: nil reviews skips approval check",
			model: statusModel{
				comments: &domain.CommentsResult{UnresolvedCount: 0},
				checks:   &domain.ChecksResult{OverallStatus: domain.StatusPass},
				reviews:  nil,
			},
			expected: true,
		},
		// Solo mode tests.
		{
			name: "solo: empty reviews is merge-ready",
			model: statusModel{
				comments: &domain.CommentsResult{UnresolvedCount: 0},
				checks:   &domain.ChecksResult{OverallStatus: domain.StatusPass},
				reviews:  []domain.Review{},
				solo:     true,
			},
			expected: true,
		},
		{
			name: "solo: commented-only reviews is merge-ready",
			model: statusModel{
				comments: &domain.CommentsResult{UnresolvedCount: 0},
				checks:   &domain.ChecksResult{OverallStatus: domain.StatusPass},
				reviews:  []domain.Review{{State: domain.ReviewCommented}},
				solo:     true,
			},
			expected: true,
		},
		{
			name: "solo: changes requested still blocks",
			model: statusModel{
				comments: &domain.CommentsResult{UnresolvedCount: 0},
				checks:   &domain.ChecksResult{OverallStatus: domain.StatusPass},
				reviews:  []domain.Review{{State: domain.ReviewChangesRequested}},
				solo:     true,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.model.isMergeReady()
			if got != tt.expected {
				t.Errorf("isMergeReady() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestStatusMergeReadyBadge(t *testing.T) {
	ready := statusModel{
		comments: &domain.CommentsResult{UnresolvedCount: 0},
		checks:   &domain.ChecksResult{OverallStatus: domain.StatusPass},
		reviews:  []domain.Review{{State: domain.ReviewApproved}},
	}
	badge, _ := ready.mergeReadyBadge()
	if badge != "READY" {
		t.Errorf("badge = %q, want READY", badge)
	}

	notReady := statusModel{
		comments: &domain.CommentsResult{UnresolvedCount: 2},
	}
	badge, _ = notReady.mergeReadyBadge()
	if badge != "NOT READY" {
		t.Errorf("badge = %q, want NOT READY", badge)
	}
}

func TestStatusThreadsSection(t *testing.T) {
	m := statusModel{
		comments: &domain.CommentsResult{
			Threads: []domain.ReviewThread{
				{Path: "main.go", Line: 10, Comments: []domain.Comment{{Author: "alice", Body: "fix"}}},
				{Path: "api.go", Line: 23, Comments: []domain.Comment{{Author: "bob", Body: "nit"}}},
			},
			UnresolvedCount: 2,
			ResolvedCount:   1,
		},
	}
	m.setSize(120, 30)
	view := m.View()

	if !strings.Contains(view, "Review Threads") {
		t.Error("missing section title 'Review Threads'")
	}
	if !strings.Contains(view, "main.go") {
		t.Error("missing thread file 'main.go'")
	}
	if !strings.Contains(view, "@alice") {
		t.Error("missing thread author '@alice'")
	}
	if !strings.Contains(view, "2 unresolved") {
		t.Error("missing '2 unresolved' in section header")
	}
}

func TestStatusThreadsTruncation(t *testing.T) {
	threads := make([]domain.ReviewThread, 5)
	for i := range threads {
		threads[i] = domain.ReviewThread{
			Path: "file.go", Line: i + 1,
			Comments: []domain.Comment{{Author: "reviewer", Body: "comment"}},
		}
	}
	m := statusModel{
		comments: &domain.CommentsResult{Threads: threads, UnresolvedCount: 5},
	}
	m.setSize(120, 30)
	view := m.View()

	if !strings.Contains(view, "... and 2 more") {
		t.Error("missing truncation '... and 2 more'")
	}
}

func TestStatusChecksSection(t *testing.T) {
	m := statusModel{
		checks: &domain.ChecksResult{
			Checks: []domain.CheckRun{
				{
					Name: "lint", Status: "completed", Conclusion: "failure",
					Annotations: []domain.Annotation{{Path: "api.go", StartLine: 5, Title: "errcheck"}},
				},
				{Name: "build", Status: "completed", Conclusion: "success"},
				{Name: "test", Status: "completed", Conclusion: "success"},
			},
			PassCount: 2,
			FailCount: 1,
		},
	}
	m.setSize(120, 30)
	view := m.View()

	if !strings.Contains(view, "CI Checks") {
		t.Error("missing section title 'CI Checks'")
	}
	if !strings.Contains(view, "lint") {
		t.Error("missing failed check 'lint'")
	}
	if !strings.Contains(view, "errcheck") {
		t.Error("missing annotation title 'errcheck'")
	}
	if !strings.Contains(view, "2 checks passed") {
		t.Error("missing '2 checks passed'")
	}
}

func TestStatusApprovalsSection(t *testing.T) {
	m := statusModel{
		reviews: []domain.Review{
			{Author: "alice", State: domain.ReviewApproved, SubmittedAt: time.Now()},
			{Author: "bob", State: domain.ReviewChangesRequested, IsStale: true, SubmittedAt: time.Now()},
		},
	}
	m.setSize(120, 30)
	view := m.View()

	if !strings.Contains(view, "Approvals") {
		t.Error("missing section title 'Approvals'")
	}
	if !strings.Contains(view, "@alice") {
		t.Error("missing reviewer '@alice'")
	}
	if !strings.Contains(view, "approved") {
		t.Error("missing 'approved' status")
	}
	if !strings.Contains(view, "@bob") {
		t.Error("missing reviewer '@bob'")
	}
	if !strings.Contains(view, "changes requested") {
		t.Error("missing 'changes requested' status")
	}
	if !strings.Contains(view, "(stale)") {
		t.Error("missing stale marker for blocking review")
	}
	if !strings.Contains(view, "2 reviews · 1 stale") {
		t.Error("missing stale count in approvals header")
	}
}

func TestStatusReviewIcons(t *testing.T) {
	tests := []struct {
		review   domain.Review
		wantIcon string
		wantText string
	}{
		{review: domain.Review{State: domain.ReviewApproved}, wantIcon: "✓", wantText: "approved"},
		{review: domain.Review{State: domain.ReviewChangesRequested}, wantIcon: "✗", wantText: "changes requested"},
		{review: domain.Review{State: domain.ReviewChangesRequested, IsStale: true}, wantIcon: "✗", wantText: "stale"},
		{review: domain.Review{State: domain.ReviewCommented}, wantIcon: "○", wantText: "commented"},
		{review: domain.Review{State: domain.ReviewDismissed}, wantIcon: "—", wantText: "dismissed"},
		{review: domain.Review{State: domain.ReviewPending}, wantIcon: "◌", wantText: "pending"},
	}
	for _, tt := range tests {
		icon, text := reviewIcon(tt.review)
		if !strings.Contains(icon, tt.wantIcon) {
			t.Errorf("reviewIcon(%s) icon = %q, want contains %q", tt.review.State, icon, tt.wantIcon)
		}
		if !strings.Contains(text, tt.wantText) {
			t.Errorf("reviewIcon(%s) text = %q, want contains %q", tt.review.State, text, tt.wantText)
		}
	}
}

func TestStatusCardColorForCount(t *testing.T) {
	green := cardColorForCount(0, true)
	red := cardColorForCount(3, true)
	if green == red {
		t.Error("expected different colors for 0 and 3")
	}
}

func TestStatusCheckNames(t *testing.T) {
	checks := []domain.CheckRun{
		{Name: "build", Status: "completed", Conclusion: "success"},
		{Name: "test", Status: "completed", Conclusion: "success"},
		{Name: "lint", Status: "completed", Conclusion: "failure"},
	}
	passing := checkNames(checks, false)
	if !strings.Contains(passing, "build") || !strings.Contains(passing, "test") {
		t.Errorf("checkNames(false) = %q, expected build and test", passing)
	}
	failing := checkNames(checks, true)
	if !strings.Contains(failing, "lint") {
		t.Errorf("checkNames(true) = %q, expected lint", failing)
	}
}

func TestStatusAppIntegration(t *testing.T) {
	app := NewApp("owner/repo", 42, ViewStatus)
	app.SetComments(&domain.CommentsResult{
		Threads: []domain.ReviewThread{
			{Path: "main.go", Line: 10, Comments: []domain.Comment{{Author: "alice", Body: "fix"}}},
		},
		UnresolvedCount: 1,
	})
	app.SetChecks(&domain.ChecksResult{
		OverallStatus: domain.StatusPass,
		PassCount:     3,
	})
	app.SetReviews([]domain.Review{
		{Author: "bob", State: domain.ReviewApproved},
	})
	app = sendWindowSize(app, 120, 30)

	view := app.View()

	// Status bar should show NOT READY (1 unresolved).
	if !strings.Contains(view, "NOT READY") {
		t.Error("missing 'NOT READY' badge in status bar")
	}

	// Section headers should appear.
	if !strings.Contains(view, "Review Threads") {
		t.Error("missing 'Review Threads' section")
	}
	if !strings.Contains(view, "CI Checks") {
		t.Error("missing 'CI Checks' section")
	}
	if !strings.Contains(view, "Approvals") {
		t.Error("missing 'Approvals' section")
	}

	// Help bar should show status-specific bindings.
	if !strings.Contains(view, "comments") || !strings.Contains(view, "checks") {
		t.Error("missing status key bindings in help bar")
	}
}

func TestStatusReadyAppIntegration(t *testing.T) {
	app := NewApp("owner/repo", 42, ViewStatus)
	app.SetComments(&domain.CommentsResult{UnresolvedCount: 0})
	app.SetChecks(&domain.ChecksResult{OverallStatus: domain.StatusPass, PassCount: 5})
	app.SetReviews([]domain.Review{{Author: "alice", State: domain.ReviewApproved}})
	app = sendWindowSize(app, 120, 30)

	view := app.View()
	if !strings.Contains(view, "READY") {
		t.Error("missing 'READY' badge for merge-ready PR")
	}
	// Make sure it's "READY" not "NOT READY"
	// Count occurrences: if "NOT READY" is present, that's wrong for this test.
	if strings.Contains(view, "NOT READY") {
		t.Error("should show 'READY', not 'NOT READY'")
	}
}

func TestStatusQuickNav(t *testing.T) {
	app := NewApp("owner/repo", 42, ViewStatus)
	app = sendWindowSize(app, 120, 30)

	// 'c' should jump to comments.
	appC := sendKey(app, "c")
	if appC.ActiveView() != ViewCommentsList {
		t.Errorf("'c' from status: expected ViewCommentsList, got %v", appC.ActiveView())
	}

	// 'k' should jump to checks.
	appK := sendKey(app, "k")
	if appK.ActiveView() != ViewChecksList {
		t.Errorf("'k' from status: expected ViewChecksList, got %v", appK.ActiveView())
	}

	// 'r' should jump to resolve.
	appR := sendKey(app, "r")
	if appR.ActiveView() != ViewResolve {
		t.Errorf("'r' from status: expected ViewResolve, got %v", appR.ActiveView())
	}
}

func TestStatusZeroWidth(t *testing.T) {
	m := statusModel{}
	view := m.View()
	if view != "" {
		t.Errorf("expected empty view at zero width, got %q", view)
	}
}

func TestStatusApprovalsCapped(t *testing.T) {
	reviews := make([]domain.Review, 20)
	for i := range reviews {
		reviews[i] = domain.Review{
			Author:      "user" + string(rune('A'+i)),
			State:       domain.ReviewCommented,
			SubmittedAt: time.Now(),
		}
	}
	m := statusModel{reviews: reviews}
	m.setSize(120, 30)
	view := m.View()

	if !strings.Contains(view, "... and 15 more") {
		t.Errorf("missing overflow indicator, expected '... and 15 more' in:\n%s", view)
	}
	// Should NOT contain the 6th reviewer's author name fully rendered in the list.
	// (maxReviewsShow = 5, so only first 5 are shown)
	if !strings.Contains(view, "20 reviews") {
		t.Error("missing total review count in section header")
	}
}

func TestStatusApprovalsPriorityOrder(t *testing.T) {
	reviews := []domain.Review{
		{Author: "commenter1", State: domain.ReviewCommented, SubmittedAt: time.Now()},
		{Author: "approver1", State: domain.ReviewApproved, SubmittedAt: time.Now()},
		{Author: "requester1", State: domain.ReviewChangesRequested, SubmittedAt: time.Now()},
		{Author: "commenter2", State: domain.ReviewCommented, SubmittedAt: time.Now()},
		{Author: "approver2", State: domain.ReviewApproved, SubmittedAt: time.Now()},
	}
	m := statusModel{reviews: reviews}
	m.setSize(120, 30)
	section := m.renderApprovalsSection()
	lines := strings.Split(section, "\n")

	// Find order: CHANGES_REQUESTED should be first, then APPROVED, then COMMENTED.
	var order []string
	for _, line := range lines {
		switch {
		case strings.Contains(line, "changes requested"):
			order = append(order, "changes_requested")
		case strings.Contains(line, "approved"):
			order = append(order, "approved")
		case strings.Contains(line, "commented"):
			order = append(order, "commented")
		}
	}

	if len(order) < 3 {
		t.Fatalf("expected at least 3 review state lines, got %d: %v", len(order), order)
	}
	if order[0] != "changes_requested" {
		t.Errorf("first review should be changes_requested, got %s", order[0])
	}
	if order[1] != "approved" {
		t.Errorf("second review should be approved, got %s", order[1])
	}
}

func TestStatusApprovalsSmallList(t *testing.T) {
	reviews := []domain.Review{
		{Author: "alice", State: domain.ReviewApproved, SubmittedAt: time.Now()},
		{Author: "bob", State: domain.ReviewCommented, SubmittedAt: time.Now()},
	}
	m := statusModel{reviews: reviews}
	m.setSize(120, 30)
	view := m.View()

	// Small list: all reviews shown, no overflow indicator.
	if strings.Contains(view, "... and") {
		t.Error("should not show overflow for 2 reviews")
	}
	if !strings.Contains(view, "@alice") {
		t.Error("missing @alice")
	}
	if !strings.Contains(view, "@bob") {
		t.Error("missing @bob")
	}
}

func TestStatusSoloApprovalsSection(t *testing.T) {
	// Solo mode with no reviews should show solo message.
	m := statusModel{solo: true}
	m.setSize(120, 30)
	view := m.View()

	if !strings.Contains(view, "Solo mode") {
		t.Error("missing 'Solo mode' text in solo approvals section")
	}
	if !strings.Contains(view, "approval not required") {
		t.Error("missing 'approval not required' text in solo approvals section")
	}
	// Should NOT show "No reviews yet".
	if strings.Contains(view, "No reviews yet") {
		t.Error("should show solo message, not 'No reviews yet'")
	}
}

func TestStatusSoloApprovalsWithReviews(t *testing.T) {
	// Solo mode with existing reviews should still render them normally.
	m := statusModel{
		solo: true,
		reviews: []domain.Review{
			{Author: "alice", State: domain.ReviewCommented},
		},
	}
	m.setSize(120, 30)
	view := m.View()

	if !strings.Contains(view, "@alice") {
		t.Error("missing @alice in solo mode with reviews")
	}
	// Should not show solo message when reviews exist.
	if strings.Contains(view, "Solo mode") {
		t.Error("should not show solo message when reviews exist")
	}
}

func TestStatusSoloKPICard(t *testing.T) {
	// Solo mode with no approvals should show "—" not "0".
	m := statusModel{
		solo:     true,
		comments: &domain.CommentsResult{UnresolvedCount: 0},
		checks:   &domain.ChecksResult{OverallStatus: domain.StatusPass, PassCount: 3},
	}
	m.setSize(120, 30)
	view := m.View()

	if !strings.Contains(view, "—") {
		t.Error("solo mode with no approvals should show '—' in KPI card")
	}
}

func TestStatusSoloWithErrorsNotGreen(t *testing.T) {
	// Solo + hasErrors (review fetch failed) should NOT show green approvals.
	m := statusModel{
		solo:      true,
		hasErrors: true,
		comments:  &domain.CommentsResult{UnresolvedCount: 0},
		checks:    &domain.ChecksResult{OverallStatus: domain.StatusPass, PassCount: 3},
	}
	if m.isMergeReady() {
		t.Error("solo + hasErrors should NOT be merge-ready")
	}
	badge, _ := m.mergeReadyBadge()
	if badge != "NOT READY" {
		t.Errorf("badge = %q, want NOT READY when hasErrors", badge)
	}
}

func TestStatusSoloBadge(t *testing.T) {
	// Solo + clean checks + no threads + no reviews = READY.
	app := NewApp("owner/repo", 42, ViewStatus)
	app.SetSolo(true)
	app.SetComments(&domain.CommentsResult{UnresolvedCount: 0})
	app.SetChecks(&domain.ChecksResult{OverallStatus: domain.StatusPass, PassCount: 3})
	app.SetReviews([]domain.Review{})
	app = sendWindowSize(app, 120, 30)

	view := app.View()
	if strings.Contains(view, "NOT READY") {
		t.Error("solo mode should show READY, not NOT READY")
	}
	if !strings.Contains(view, "READY") {
		t.Error("missing READY badge in solo mode")
	}
}

func TestStatusScrolling(t *testing.T) {
	m := statusModel{
		comments: &domain.CommentsResult{UnresolvedCount: 3},
		checks:   &domain.ChecksResult{PassCount: 5},
	}
	// Set height very small to force scroll.
	m.setSize(120, 5)
	view1 := m.View()
	lines1 := strings.Split(view1, "\n")
	if len(lines1) != 5 {
		t.Errorf("expected 5 visible lines, got %d", len(lines1))
	}

	// Scroll down.
	m.scrollDown()
	view2 := m.View()
	if view1 == view2 {
		t.Error("scroll down should change visible content")
	}

	// Scroll back up.
	m.scrollUp()
	view3 := m.View()
	if view1 != view3 {
		t.Error("scroll up should restore original content")
	}

	// Scroll up past 0 should stay at 0.
	m.scrollUp()
	view4 := m.View()
	if view1 != view4 {
		t.Error("scroll up past 0 should stay at 0")
	}
}

func TestStatusScrollClamp(t *testing.T) {
	m := statusModel{}
	m.setSize(120, 100) // Very tall — content fits without scroll.
	m.scrollOffset = 999
	view := m.View()
	// Should clamp and render without panic.
	if view == "" {
		t.Error("expected non-empty view even with large scroll offset")
	}
}

func TestStatusLoadingView(t *testing.T) {
	m := statusModel{loading: true}
	m.setSize(120, 30)
	view := m.View()
	if !strings.Contains(view, "Loading PR data") {
		t.Errorf("expected loading message, got %q", view)
	}
}

func TestStatusLoadingClearsOnData(t *testing.T) {
	m := statusModel{loading: true}
	m.setSize(120, 30)

	// Once comments arrive, loading view should not show.
	m.comments = &domain.CommentsResult{UnresolvedCount: 1}
	m.loading = false
	view := m.View()
	if strings.Contains(view, "Loading PR data") {
		t.Error("loading message should not appear after data arrives")
	}
	if !strings.Contains(view, "Review Threads") {
		t.Error("expected Review Threads section after data arrives")
	}
}

func TestStatusHasErrorsBlocksMergeReady(t *testing.T) {
	// Even with perfect data, hasErrors should block merge readiness.
	m := statusModel{
		comments:  &domain.CommentsResult{UnresolvedCount: 0},
		checks:    &domain.ChecksResult{OverallStatus: domain.StatusPass},
		reviews:   []domain.Review{{State: domain.ReviewApproved}},
		hasErrors: true,
	}
	if m.isMergeReady() {
		t.Error("isMergeReady() should return false when hasErrors is true")
	}
	badge, _ := m.mergeReadyBadge()
	if badge != "NOT READY" {
		t.Errorf("badge = %q, want NOT READY when hasErrors", badge)
	}
}

func TestStatusScrollDownClampsToMaxScroll(t *testing.T) {
	m := statusModel{
		comments: &domain.CommentsResult{UnresolvedCount: 1},
		checks:   &domain.ChecksResult{PassCount: 2},
	}
	m.setSize(120, 5) // Small height forces scroll

	// Scroll down many times — should not exceed maxScroll.
	for range 200 {
		m.scrollDown()
	}
	if m.scrollOffset > m.maxScroll {
		t.Errorf("scrollOffset %d exceeds maxScroll %d", m.scrollOffset, m.maxScroll)
	}
	if m.scrollOffset != m.maxScroll {
		t.Errorf("scrollOffset %d should equal maxScroll %d after many scrolls", m.scrollOffset, m.maxScroll)
	}
}

func TestStatusRecomputeMaxScrollOnDataChange(t *testing.T) {
	m := statusModel{}
	m.setSize(120, 5)

	// Initially no data — maxScroll should be computed.
	initialMax := m.maxScroll

	// Add data — more content increases maxScroll.
	m.comments = &domain.CommentsResult{
		UnresolvedCount: 3,
		Threads: []domain.ReviewThread{
			{Path: "a.go", Line: 1, Comments: []domain.Comment{{Author: "x"}}},
			{Path: "b.go", Line: 2, Comments: []domain.Comment{{Author: "y"}}},
			{Path: "c.go", Line: 3, Comments: []domain.Comment{{Author: "z"}}},
		},
	}
	m.recomputeMaxScroll()

	if m.maxScroll <= initialMax {
		t.Errorf("maxScroll should increase with more data: was %d, now %d", initialMax, m.maxScroll)
	}
}

func TestReviewPriority(t *testing.T) {
	tests := []struct {
		state    domain.ReviewState
		expected int
	}{
		{domain.ReviewChangesRequested, 0},
		{domain.ReviewApproved, 1},
		{domain.ReviewCommented, 2},
		{domain.ReviewPending, 3},
		{domain.ReviewDismissed, 3},
	}
	for _, tt := range tests {
		got := reviewPriority(tt.state)
		if got != tt.expected {
			t.Errorf("reviewPriority(%s) = %d, want %d", tt.state, got, tt.expected)
		}
	}
}
