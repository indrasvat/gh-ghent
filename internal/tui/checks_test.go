package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/indrasvat/gh-ghent/internal/domain"
)

func makeCheck(name, status, conclusion string, annotations []domain.Annotation) domain.CheckRun {
	started := time.Date(2026, 2, 23, 14, 30, 0, 0, time.UTC)
	completed := time.Time{}
	if status == "completed" {
		completed = started.Add(42 * time.Second)
	}
	return domain.CheckRun{
		ID:          1,
		Name:        name,
		Status:      status,
		Conclusion:  conclusion,
		StartedAt:   started,
		CompletedAt: completed,
		HTMLURL:     "https://github.com/test/repo/runs/1",
		Annotations: annotations,
	}
}

func TestChecksListRenders(t *testing.T) {
	checks := []domain.CheckRun{
		makeCheck("build", "completed", "success", nil),
		makeCheck("lint", "completed", "failure", []domain.Annotation{
			{Path: "main.go", StartLine: 10, Message: "unused var", Title: "govet"},
		}),
		makeCheck("test", "in_progress", "", nil),
		makeCheck("security", "queued", "", nil),
	}

	m := newChecksListModel(checks)
	m.setSize(100, 20)

	view := m.View()

	// Pass icon should be present.
	if !strings.Contains(view, "✓") {
		t.Error("missing pass icon ✓")
	}
	// Fail icon should be present.
	if !strings.Contains(view, "✗") {
		t.Error("missing fail icon ✗")
	}
	// Running icon should be present.
	if !strings.Contains(view, "⟳") {
		t.Error("missing running icon ⟳")
	}
	// Pending icon should be present.
	if !strings.Contains(view, "◌") {
		t.Error("missing pending icon ◌")
	}
	// Check names should appear.
	if !strings.Contains(view, "build") {
		t.Error("missing check name 'build'")
	}
	if !strings.Contains(view, "lint") {
		t.Error("missing check name 'lint'")
	}
}

func TestChecksListAnnotationsAutoExpand(t *testing.T) {
	checks := []domain.CheckRun{
		makeCheck("lint", "completed", "failure", []domain.Annotation{
			{Path: "main.go", StartLine: 10, Message: "unused var", Title: "govet"},
			{Path: "api.go", StartLine: 23, Message: "unchecked error", Title: "errcheck"},
		}),
	}

	m := newChecksListModel(checks)
	m.setSize(100, 20)

	view := m.View()

	// Annotations should auto-expand for failed check.
	if !strings.Contains(view, "2 errors") {
		t.Error("missing annotation count header '2 errors'")
	}
	if !strings.Contains(view, "main.go") {
		t.Error("missing annotation file path 'main.go'")
	}
	if !strings.Contains(view, ":10") {
		t.Error("missing annotation line ':10'")
	}
	if !strings.Contains(view, "api.go") {
		t.Error("missing annotation file path 'api.go'")
	}
}

func TestChecksListCursorNavigation(t *testing.T) {
	checks := []domain.CheckRun{
		makeCheck("build", "completed", "success", nil),
		makeCheck("lint", "completed", "failure", nil),
		makeCheck("test", "completed", "success", nil),
	}

	m := newChecksListModel(checks)
	m.setSize(100, 20)

	// Initial cursor at 0.
	if m.selectedCheckIdx() != 0 {
		t.Errorf("initial cursor expected 0, got %d", m.selectedCheckIdx())
	}

	// j moves down.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.selectedCheckIdx() != 1 {
		t.Errorf("after j: expected cursor 1, got %d", m.selectedCheckIdx())
	}

	// j again.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.selectedCheckIdx() != 2 {
		t.Errorf("after j+j: expected cursor 2, got %d", m.selectedCheckIdx())
	}

	// j at end stays.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.selectedCheckIdx() != 2 {
		t.Errorf("j at end: expected cursor 2, got %d", m.selectedCheckIdx())
	}

	// k moves up.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if m.selectedCheckIdx() != 1 {
		t.Errorf("after k: expected cursor 1, got %d", m.selectedCheckIdx())
	}
}

func TestChecksListEnterEmitsSelectCheckMsg(t *testing.T) {
	checks := []domain.CheckRun{
		makeCheck("build", "completed", "success", nil),
	}

	m := newChecksListModel(checks)
	m.setSize(100, 20)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected command from Enter, got nil")
	}

	msg := cmd()
	scm, ok := msg.(selectCheckMsg)
	if !ok {
		t.Fatalf("expected selectCheckMsg, got %T", msg)
	}
	if scm.checkIdx != 0 {
		t.Errorf("expected checkIdx 0, got %d", scm.checkIdx)
	}
}

func TestChecksListViewLogAlias(t *testing.T) {
	checks := []domain.CheckRun{
		makeCheck("build", "completed", "success", nil),
	}

	m := newChecksListModel(checks)
	m.setSize(100, 20)

	// 'l' should also emit selectCheckMsg (view full log).
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	if cmd == nil {
		t.Fatal("expected command from 'l' key, got nil")
	}

	msg := cmd()
	if _, ok := msg.(selectCheckMsg); !ok {
		t.Fatalf("expected selectCheckMsg from 'l', got %T", msg)
	}
}

func TestChecksListEmptyView(t *testing.T) {
	m := newChecksListModel(nil)
	m.setSize(80, 20)

	view := m.View()
	if !strings.Contains(view, "No check runs found") {
		t.Error("missing empty state message")
	}
}

func TestChecksLogModelRenders(t *testing.T) {
	check := makeCheck("lint", "completed", "failure", []domain.Annotation{
		{Path: "main.go", StartLine: 10, Message: "unused var", Title: "govet"},
	})
	check.LogExcerpt = "Step 5: Run lint\nError: unused variable 'x'\n..."

	m := newChecksLogModel(&check)
	m.setSize(100, 30)

	view := m.View()

	// Should show check name.
	if !strings.Contains(view, "lint") {
		t.Error("missing check name in log view")
	}
	// Should show annotation.
	if !strings.Contains(view, "main.go") {
		t.Error("missing annotation file in log view")
	}
	// Should show log excerpt.
	if !strings.Contains(view, "unused variable") {
		t.Error("missing log content in log view")
	}
}

func TestChecksLogModelScrolls(t *testing.T) {
	check := makeCheck("test", "completed", "failure", nil)
	// Create a long log.
	var logLines []string
	for i := range 50 {
		logLines = append(logLines, strings.Repeat("x", 80-4)+" line "+string(rune('0'+i%10)))
	}
	check.LogExcerpt = strings.Join(logLines, "\n")

	m := newChecksLogModel(&check)
	m.setSize(80, 10)

	initialView := m.View()

	// Scroll down.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	afterScroll := m.View()

	// Content should change after scroll.
	if initialView == afterScroll {
		t.Error("expected view to change after scroll down")
	}

	// Scroll back up.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	afterUp := m.View()

	// Should be back to initial.
	if afterUp != initialView {
		t.Error("expected view to match initial after scroll up")
	}
}

func TestChecksLogModelNoLogExcerpt(t *testing.T) {
	check := makeCheck("lint", "completed", "failure", nil)
	// No LogExcerpt set.

	m := newChecksLogModel(&check)
	m.setSize(80, 20)

	view := m.View()

	if !strings.Contains(view, "No log excerpt available") {
		t.Error("missing 'No log excerpt available' message for failed check without logs")
	}
}

func TestCheckIsFailed(t *testing.T) {
	tests := []struct {
		status     string
		conclusion string
		want       bool
	}{
		{"completed", "failure", true},
		{"completed", "cancelled", true},
		{"completed", "timed_out", true},
		{"completed", "success", false},
		{"completed", "skipped", false},
		{"in_progress", "", false},
		{"queued", "", false},
	}
	for _, tt := range tests {
		ch := domain.CheckRun{Status: tt.status, Conclusion: tt.conclusion}
		if got := checkIsFailed(ch); got != tt.want {
			t.Errorf("checkIsFailed(%s/%s) = %v, want %v", tt.status, tt.conclusion, got, tt.want)
		}
	}
}

func TestCheckStatusIcon(t *testing.T) {
	tests := []struct {
		status     string
		conclusion string
		wantIcon   string
	}{
		{"completed", "success", "✓"},
		{"completed", "failure", "✗"},
		{"in_progress", "", "⟳"},
		{"queued", "", "◌"},
		{"completed", "skipped", "✓"},
		{"completed", "neutral", "✓"},
		{"completed", "cancelled", "✗"},
	}
	for _, tt := range tests {
		ch := domain.CheckRun{Status: tt.status, Conclusion: tt.conclusion}
		icon := checkStatusIcon(ch)
		if !strings.Contains(icon, tt.wantIcon) {
			t.Errorf("checkStatusIcon(%s/%s) = %q, want %q", tt.status, tt.conclusion, icon, tt.wantIcon)
		}
	}
}

func TestFormatCheckDuration(t *testing.T) {
	base := time.Date(2026, 2, 23, 14, 30, 0, 0, time.UTC)
	tests := []struct {
		name      string
		started   time.Time
		completed time.Time
		status    string
		want      string
	}{
		{"42 seconds", base, base.Add(42 * time.Second), "completed", "42s"},
		{"1 min 12 sec", base, base.Add(72 * time.Second), "completed", "1m 12s"},
		{"exactly 2 min", base, base.Add(120 * time.Second), "completed", "2m"},
		{"running", base, time.Time{}, "in_progress", "running..."},
		{"no times", time.Time{}, time.Time{}, "queued", "—"},
	}
	for _, tt := range tests {
		ch := domain.CheckRun{
			StartedAt:   tt.started,
			CompletedAt: tt.completed,
			Status:      tt.status,
		}
		got := formatCheckDuration(ch)
		if got != tt.want {
			t.Errorf("%s: formatCheckDuration = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestRenderCheckStatusText(t *testing.T) {
	tests := []struct {
		status     string
		conclusion string
		want       string
	}{
		{"completed", "success", "passed"},
		{"completed", "failure", "failed"},
		{"completed", "cancelled", "cancelled"},
		{"completed", "skipped", "skipped"},
		{"in_progress", "", "running"},
		{"queued", "", "queued"},
	}
	for _, tt := range tests {
		ch := domain.CheckRun{Status: tt.status, Conclusion: tt.conclusion}
		got := renderCheckStatusText(ch)
		if !strings.Contains(got, tt.want) {
			t.Errorf("renderCheckStatusText(%s/%s) = %q, want to contain %q",
				tt.status, tt.conclusion, got, tt.want)
		}
	}
}

func TestScreenLinesForCheck(t *testing.T) {
	checks := []domain.CheckRun{
		makeCheck("pass", "completed", "success", nil),
		makeCheck("fail", "completed", "failure", []domain.Annotation{
			{Path: "a.go", StartLine: 1, Message: "error 1"},
			{Path: "b.go", StartLine: 2, Message: "error 2"},
			{Path: "c.go", StartLine: 3, Message: "error 3"},
		}),
	}

	m := newChecksListModel(checks)

	// Passing check: 1 line.
	if got := m.screenLinesForCheck(0); got != 1 {
		t.Errorf("pass check: expected 1 line, got %d", got)
	}
	// Failed check with 3 annotations: 1 + 1 header + 3 annotations = 5 lines.
	if got := m.screenLinesForCheck(1); got != 5 {
		t.Errorf("fail check with 3 annotations: expected 5 lines, got %d", got)
	}
}

// ── Task 033: New keybinding tests ──────────────────────────────

func TestExtractRunID(t *testing.T) {
	tests := []struct {
		name    string
		htmlURL string
		want    string
	}{
		{
			"standard GitHub Actions URL",
			"https://github.com/owner/repo/actions/runs/12345/job/67890",
			"12345",
		},
		{
			"URL without job suffix",
			"https://github.com/owner/repo/actions/runs/99999",
			"99999",
		},
		{
			"external CI URL (no actions/runs)",
			"https://ci.example.com/builds/123",
			"",
		},
		{
			"empty URL",
			"",
			"",
		},
		{
			"URL with long numeric run ID",
			"https://github.com/indrasvat/peek-it/actions/runs/13579246801/job/37890",
			"13579246801",
		},
		{
			"check run /runs/ format (REST API)",
			"https://github.com/owner/repo/runs/100001",
			"100001",
		},
		{
			"check run /runs/ with query params",
			"https://github.com/owner/repo/runs/100001?check_suite_focus=true",
			"100001",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractRunID(tt.htmlURL)
			if got != tt.want {
				t.Errorf("extractRunID(%q) = %q, want %q", tt.htmlURL, got, tt.want)
			}
		})
	}
}

func TestChecksAppIntegration(t *testing.T) {
	app := NewApp("owner/repo", 42, ViewChecksList)
	app.SetChecks(&domain.ChecksResult{
		HeadSHA: "abc1234567890",
		Checks: []domain.CheckRun{
			makeCheck("build", "completed", "success", nil),
			makeCheck("lint", "completed", "failure", []domain.Annotation{
				{Path: "main.go", StartLine: 10, Message: "unused", Title: "govet"},
			}),
		},
		PassCount: 1,
		FailCount: 1,
	})
	app = sendWindowSize(app, 100, 30)

	view := app.View()

	// Status bar should show pass/fail counts.
	if !strings.Contains(view, "1 passed") {
		t.Error("missing '1 passed' in status bar")
	}
	if !strings.Contains(view, "1 failed") {
		t.Error("missing '1 failed' in status bar")
	}

	// Check names should appear.
	if !strings.Contains(view, "build") {
		t.Error("missing 'build' in checks view")
	}
	if !strings.Contains(view, "lint") {
		t.Error("missing 'lint' in checks view")
	}

	// Enter to open log viewer.
	model, cmd := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = model.(App)
	if cmd != nil {
		msg := cmd()
		model, _ = app.Update(msg)
		app = model.(App)
	}
	if app.ActiveView() != ViewChecksLog {
		t.Errorf("expected ViewChecksLog, got %v", app.ActiveView())
	}

	// Log view should show the check name.
	logView := app.View()
	if !strings.Contains(logView, "build") {
		t.Error("missing check name in log view")
	}

	// Esc returns to list.
	app = sendSpecialKey(app, tea.KeyEscape)
	if app.ActiveView() != ViewChecksList {
		t.Errorf("expected ViewChecksList after Esc, got %v", app.ActiveView())
	}
}
