package tui

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/indrasvat/gh-ghent/internal/domain"
)

func makeChecksResult(checks []domain.CheckRun, overall domain.OverallStatus) *domain.ChecksResult {
	pass, fail := 0, 0
	for _, ch := range checks {
		if ch.Status == "completed" && ch.Conclusion == "success" {
			pass++
		}
		if ch.Status == "completed" && (ch.Conclusion == "failure" || ch.Conclusion == "timed_out") {
			fail++
		}
	}
	return &domain.ChecksResult{
		Checks:        checks,
		OverallStatus: overall,
		PassCount:     pass,
		FailCount:     fail,
		PendingCount:  len(checks) - pass - fail,
	}
}

func TestWatcherEmptyView(t *testing.T) {
	m := watcherModel{}
	view := m.View()
	if view != "" {
		t.Errorf("expected empty view at zero width, got %q", view)
	}
}

func TestWatcherInitialView(t *testing.T) {
	m := newWatcherModel(10 * time.Second)
	m.setSize(100, 30)

	view := m.View()
	if !strings.Contains(view, "watching") {
		t.Error("missing 'watching' in initial view")
	}
	if !strings.Contains(view, "Waiting for first poll") {
		t.Error("missing 'Waiting for first poll' in initial view")
	}
	if !strings.Contains(view, "Event Log") {
		t.Error("missing 'Event Log' header")
	}
	if !strings.Contains(view, "poll: 10s") {
		t.Error("missing poll interval")
	}
}

func TestWatcherPollResult(t *testing.T) {
	m := newWatcherModel(10 * time.Second)
	m.setSize(100, 30)

	checks := makeChecksResult([]domain.CheckRun{
		{ID: 1, Name: "lint", Status: "completed", Conclusion: "success"},
		{ID: 2, Name: "build", Status: "in_progress"},
	}, domain.StatusPending)

	m, _ = m.handlePollResult(watchResultMsg{checks: checks})

	if m.completed != 1 {
		t.Errorf("completed = %d, want 1", m.completed)
	}
	if m.total != 2 {
		t.Errorf("total = %d, want 2", m.total)
	}
	if len(m.events) != 1 {
		t.Errorf("events = %d, want 1", len(m.events))
	}
	if m.state != watchStatePolling {
		t.Error("expected polling state")
	}

	view := m.View()
	if !strings.Contains(view, "lint") {
		t.Error("missing check name 'lint' in view")
	}
	if !strings.Contains(view, "build") {
		t.Error("missing check name 'build' in view")
	}
}

func TestWatcherAllPass(t *testing.T) {
	m := newWatcherModel(10 * time.Second)
	m.setSize(100, 30)

	checks := makeChecksResult([]domain.CheckRun{
		{ID: 1, Name: "lint", Status: "completed", Conclusion: "success"},
		{ID: 2, Name: "build", Status: "completed", Conclusion: "success"},
	}, domain.StatusPass)

	m, cmd := m.handlePollResult(watchResultMsg{checks: checks})

	if m.state != watchStateDone {
		t.Errorf("state = %d, want watchStateDone", m.state)
	}
	if cmd != nil {
		t.Error("expected nil cmd on terminal state")
	}

	view := m.View()
	if !strings.Contains(view, "all checks passed") {
		t.Error("missing 'all checks passed' in done view")
	}
}

func TestWatcherFailFast(t *testing.T) {
	m := newWatcherModel(10 * time.Second)
	m.setSize(100, 30)

	checks := makeChecksResult([]domain.CheckRun{
		{ID: 1, Name: "lint", Status: "completed", Conclusion: "failure"},
		{ID: 2, Name: "build", Status: "in_progress"},
	}, domain.StatusFail)

	m, cmd := m.handlePollResult(watchResultMsg{checks: checks})

	if m.state != watchStateFailed {
		t.Errorf("state = %d, want watchStateFailed", m.state)
	}
	if cmd != nil {
		t.Error("expected nil cmd on terminal state")
	}

	view := m.View()
	if !strings.Contains(view, "failure detected") {
		t.Error("missing 'failure detected' in failed view")
	}
	if !strings.Contains(view, "fail-fast") {
		t.Error("missing 'fail-fast' in event log")
	}
}

func TestWatcherPollError(t *testing.T) {
	m := newWatcherModel(10 * time.Second)
	m.setSize(100, 30)

	m, cmd := m.handlePollResult(watchResultMsg{err: errors.New("network timeout")})

	if m.state != watchStatePolling {
		t.Error("should stay polling after error")
	}
	if cmd == nil {
		t.Error("expected schedule next poll cmd")
	}
	if len(m.events) != 1 {
		t.Errorf("events = %d, want 1", len(m.events))
	}
	if m.events[0].name != "poll error" {
		t.Errorf("event name = %q, want 'poll error'", m.events[0].name)
	}
}

func TestWatcherSeenDedup(t *testing.T) {
	m := newWatcherModel(10 * time.Second)
	m.setSize(100, 30)

	checks := makeChecksResult([]domain.CheckRun{
		{ID: 1, Name: "lint", Status: "completed", Conclusion: "success"},
	}, domain.StatusPending)

	m, _ = m.handlePollResult(watchResultMsg{checks: checks})
	if len(m.events) != 1 {
		t.Fatalf("events after first poll = %d, want 1", len(m.events))
	}

	// Second poll with same check — should NOT add duplicate event.
	m, _ = m.handlePollResult(watchResultMsg{checks: checks})
	// events = 1 (lint) — NOT duplicated
	lintEvents := 0
	for _, e := range m.events {
		if e.name == "lint" {
			lintEvents++
		}
	}
	if lintEvents != 1 {
		t.Errorf("lint events = %d, want 1 (dedup failed)", lintEvents)
	}
}

func TestWatcherTickIgnoredWhenDone(t *testing.T) {
	m := newWatcherModel(10 * time.Second)
	m.state = watchStateDone

	m, cmd := m.Update(watchTickMsg(time.Now()))
	if cmd != nil {
		t.Error("tick should be ignored when done")
	}
}

func TestWatcherPollCmdNilWithoutFetchFn(t *testing.T) {
	m := newWatcherModel(10 * time.Second)
	m.fetchFn = nil
	cmd := m.pollCmd()
	if cmd != nil {
		t.Error("pollCmd should return nil without fetchFn")
	}
}

func TestWatcherEventLogScroll(t *testing.T) {
	m := newWatcherModel(10 * time.Second)
	m.setSize(100, 15) // Small height to trigger scrolling.

	// Add many events.
	for range 20 {
		m.events = append(m.events, watchEvent{
			timestamp: time.Now(),
			icon:      "✓",
			name:      "check",
		})
	}
	m.logOffset = max(len(m.events)-m.eventLogHeight(), 0)

	view := m.View()
	// Should render without panic.
	if view == "" {
		t.Error("expected non-empty view with events")
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{0, "0s"},
		{5 * time.Second, "5s"},
		{59 * time.Second, "59s"},
		{60 * time.Second, "1m"},
		{90 * time.Second, "1m 30s"},
		{125 * time.Second, "2m 5s"},
	}
	for _, tt := range tests {
		got := formatDuration(tt.d)
		if got != tt.want {
			t.Errorf("formatDuration(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}

func TestWatcherMakeEvent(t *testing.T) {
	m := newWatcherModel(10 * time.Second)

	// Success event.
	ev := m.makeEvent(domain.CheckRun{
		Name:        "lint",
		Status:      "completed",
		Conclusion:  "success",
		StartedAt:   time.Now().Add(-10 * time.Second),
		CompletedAt: time.Now(),
	})
	if ev.name != "lint" {
		t.Errorf("name = %q, want 'lint'", ev.name)
	}
	if !strings.Contains(ev.icon, "✓") {
		t.Error("success event should have ✓ icon")
	}
	if ev.detail == "" {
		t.Error("expected duration detail for completed check")
	}

	// Failure event.
	ev = m.makeEvent(domain.CheckRun{
		Name:       "build",
		Status:     "completed",
		Conclusion: "failure",
	})
	if !strings.Contains(ev.icon, "✗") {
		t.Error("failure event should have ✗ icon")
	}

	// Skipped event.
	ev = m.makeEvent(domain.CheckRun{
		Name:       "deploy",
		Status:     "completed",
		Conclusion: "skipped",
	})
	if !strings.Contains(ev.icon, "—") {
		t.Error("skipped event should have — icon")
	}
}

func TestWatcherAppIntegration(t *testing.T) {
	fetchCalled := false
	fetchFn := func() (*domain.ChecksResult, error) {
		fetchCalled = true
		return makeChecksResult([]domain.CheckRun{
			{ID: 1, Name: "test", Status: "completed", Conclusion: "success"},
		}, domain.StatusPass), nil
	}

	app := NewApp("owner/repo", 42, ViewWatch)
	app.SetWatchFetch(fetchFn, 10*time.Second)
	app = sendWindowSize(app, 100, 30)

	// Init should return commands (spinner + poll).
	initCmd := app.Init()
	if initCmd == nil {
		t.Error("Init should return commands for ViewWatch")
	}

	// The fetch function should be configured.
	if app.watcher.fetchFn == nil {
		t.Error("fetchFn not set on watcher")
	}

	// Simulate a poll result.
	checks := makeChecksResult([]domain.CheckRun{
		{ID: 1, Name: "test", Status: "completed", Conclusion: "success"},
	}, domain.StatusPass)
	var model tea.Model
	model, _ = app.Update(watchResultMsg{checks: checks})
	app = model.(App)

	if app.watcher.state != watchStateDone {
		t.Errorf("watcher state = %d, want done", app.watcher.state)
	}

	view := app.View()
	if !strings.Contains(view, "all checks passed") {
		t.Error("missing 'all checks passed' in app view")
	}

	// Help bar should show watch keys.
	if !strings.Contains(view, "quit") {
		t.Error("missing watch key bindings in help bar")
	}

	_ = fetchCalled // fetchFn is passed but poll is driven by messages
}

func TestWatcherStatusBar(t *testing.T) {
	app := NewApp("owner/repo", 42, ViewWatch)
	app.SetWatchFetch(func() (*domain.ChecksResult, error) {
		return nil, nil
	}, 10*time.Second)

	checks := &domain.ChecksResult{
		PassCount: 2,
		FailCount: 1,
		HeadSHA:   "abc1234567890",
	}
	app.SetChecks(checks)
	app = sendWindowSize(app, 100, 30)

	view := app.View()
	// Status bar should show HEAD SHA and pass/fail counts.
	if !strings.Contains(view, "abc1234") {
		t.Error("missing HEAD SHA in status bar")
	}
	if !strings.Contains(view, "2 passed") {
		t.Error("missing pass count in status bar")
	}
	if !strings.Contains(view, "1 failed") {
		t.Error("missing fail count in status bar")
	}
}
