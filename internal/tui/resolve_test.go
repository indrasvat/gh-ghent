package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/indrasvat/gh-ghent/internal/domain"
)

func makeThread(id, path string, line int, canResolve bool) domain.ReviewThread {
	return domain.ReviewThread{
		ID:               id,
		Path:             path,
		Line:             line,
		ViewerCanResolve: canResolve,
		Comments: []domain.Comment{
			{Author: "reviewer1", Body: "Please fix this"},
		},
	}
}

func TestResolveEmptyView(t *testing.T) {
	m := newResolveModel(nil)
	m.setSize(80, 20)
	view := m.View()
	if !strings.Contains(view, "No review threads") {
		t.Error("missing empty state message")
	}
}

func TestResolveRenderThreads(t *testing.T) {
	threads := []domain.ReviewThread{
		makeThread("PRRT_aaa", "main.go", 10, true),
		makeThread("PRRT_bbb", "api.go", 23, true),
	}
	m := newResolveModel(threads)
	m.setSize(100, 20)
	view := m.View()

	if !strings.Contains(view, "main.go") {
		t.Error("missing file path 'main.go'")
	}
	if !strings.Contains(view, "api.go") {
		t.Error("missing file path 'api.go'")
	}
	if !strings.Contains(view, "[ ]") {
		t.Error("missing unchecked checkbox")
	}
	if !strings.Contains(view, "▶") {
		t.Error("missing cursor marker")
	}
}

func TestResolveSpaceTogglesSelection(t *testing.T) {
	threads := []domain.ReviewThread{
		makeThread("PRRT_aaa", "main.go", 10, true),
		makeThread("PRRT_bbb", "api.go", 23, true),
	}
	m := newResolveModel(threads)
	m.setSize(100, 20)

	// Initially no selections.
	if m.selectedCount() != 0 {
		t.Errorf("initial selectedCount = %d, want 0", m.selectedCount())
	}

	// Space selects first thread.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	if m.selectedCount() != 1 {
		t.Errorf("after space: selectedCount = %d, want 1", m.selectedCount())
	}
	if !m.selected[0] {
		t.Error("expected thread 0 to be selected")
	}

	// Space again deselects.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	if m.selectedCount() != 0 {
		t.Errorf("after second space: selectedCount = %d, want 0", m.selectedCount())
	}
}

func TestResolveSelectAll(t *testing.T) {
	threads := []domain.ReviewThread{
		makeThread("PRRT_aaa", "main.go", 10, true),
		makeThread("PRRT_bbb", "api.go", 23, true),
		makeThread("PRRT_ccc", "util.go", 5, false), // no permission
	}
	m := newResolveModel(threads)
	m.setSize(100, 20)

	// 'a' selects all eligible.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	if m.selectedCount() != 2 {
		t.Errorf("after 'a': selectedCount = %d, want 2", m.selectedCount())
	}
	if m.selected[2] {
		t.Error("no-permission thread should not be selected")
	}

	// 'a' again deselects all.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	if m.selectedCount() != 0 {
		t.Errorf("after second 'a': selectedCount = %d, want 0", m.selectedCount())
	}
}

func TestResolveNoPermissionCannotSelect(t *testing.T) {
	threads := []domain.ReviewThread{
		makeThread("PRRT_aaa", "main.go", 10, false),
	}
	m := newResolveModel(threads)
	m.setSize(100, 20)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	if m.selectedCount() != 0 {
		t.Errorf("no-permission thread selectedCount = %d, want 0", m.selectedCount())
	}

	// View should show "(no permission)".
	view := m.View()
	if !strings.Contains(view, "no permission") {
		t.Error("missing '(no permission)' label")
	}
}

func TestResolveCursorNavigation(t *testing.T) {
	threads := []domain.ReviewThread{
		makeThread("PRRT_aaa", "main.go", 10, true),
		makeThread("PRRT_bbb", "api.go", 23, true),
		makeThread("PRRT_ccc", "util.go", 5, true),
	}
	m := newResolveModel(threads)
	m.setSize(100, 20)

	if m.cursor != 0 {
		t.Errorf("initial cursor = %d, want 0", m.cursor)
	}

	// j moves down.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.cursor != 1 {
		t.Errorf("after j: cursor = %d, want 1", m.cursor)
	}

	// j again.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.cursor != 2 {
		t.Errorf("after j+j: cursor = %d, want 2", m.cursor)
	}

	// j at end stays.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.cursor != 2 {
		t.Errorf("j at end: cursor = %d, want 2", m.cursor)
	}

	// k moves up.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if m.cursor != 1 {
		t.Errorf("after k: cursor = %d, want 1", m.cursor)
	}
}

func TestResolveEnterShowsConfirmation(t *testing.T) {
	threads := []domain.ReviewThread{
		makeThread("PRRT_aaa", "main.go", 10, true),
	}
	m := newResolveModel(threads)
	m.setSize(100, 20)

	// Select the thread.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	// Enter shows confirmation.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != resolveStateConfirming {
		t.Errorf("state = %d, want %d (confirming)", m.state, resolveStateConfirming)
	}

	// View should show confirmation bar.
	view := m.View()
	if !strings.Contains(view, "Resolve 1 thread?") {
		t.Error("missing confirmation prompt 'Resolve 1 thread?'")
	}
}

func TestResolveEnterWithoutSelectionDoesNotConfirm(t *testing.T) {
	threads := []domain.ReviewThread{
		makeThread("PRRT_aaa", "main.go", 10, true),
	}
	m := newResolveModel(threads)
	m.setSize(100, 20)

	// Enter without selection stays in browsing.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != resolveStateBrowsing {
		t.Errorf("state = %d, want %d (browsing)", m.state, resolveStateBrowsing)
	}
}

func TestResolveEscCancelsConfirmation(t *testing.T) {
	threads := []domain.ReviewThread{
		makeThread("PRRT_aaa", "main.go", 10, true),
	}
	m := newResolveModel(threads)
	m.setSize(100, 20)

	// Select → Enter → Confirming.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != resolveStateConfirming {
		t.Fatal("expected confirming state")
	}

	// Esc cancels.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if m.state != resolveStateBrowsing {
		t.Errorf("after esc: state = %d, want %d (browsing)", m.state, resolveStateBrowsing)
	}
}

func TestResolveConfirmEmitsRequest(t *testing.T) {
	threads := []domain.ReviewThread{
		makeThread("PRRT_aaa", "main.go", 10, true),
		makeThread("PRRT_bbb", "api.go", 23, true),
	}
	m := newResolveModel(threads)
	m.setSize(100, 20)

	// Select both.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	// Enter → confirming.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	// Enter again → resolving.
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != resolveStateResolving {
		t.Errorf("state = %d, want %d (resolving)", m.state, resolveStateResolving)
	}
	if cmd == nil {
		t.Fatal("expected command from confirm, got nil")
	}
	msg := cmd()
	req, ok := msg.(resolveRequestMsg)
	if !ok {
		t.Fatalf("expected resolveRequestMsg, got %T", msg)
	}
	if len(req.threadIDs) != 2 {
		t.Errorf("threadIDs count = %d, want 2", len(req.threadIDs))
	}
}

func TestResolveThreadMsgUpdatesState(t *testing.T) {
	threads := []domain.ReviewThread{
		makeThread("PRRT_aaa", "main.go", 10, true),
	}
	m := newResolveModel(threads)
	m.setSize(100, 20)
	m.selected[0] = true
	m.state = resolveStateResolving

	// Receive successful resolve.
	m, cmd := m.Update(resolveThreadMsg{threadID: "PRRT_aaa"})
	if !m.resolved[0] {
		t.Error("thread 0 should be marked resolved")
	}
	// Should emit allDone msg since all selected are done.
	if cmd == nil {
		t.Fatal("expected allDone command")
	}
	msg := cmd()
	if _, ok := msg.(resolveAllDoneMsg); !ok {
		t.Fatalf("expected resolveAllDoneMsg, got %T", msg)
	}
}

func TestResolveDoneView(t *testing.T) {
	threads := []domain.ReviewThread{
		makeThread("PRRT_aaa", "main.go", 10, true),
	}
	m := newResolveModel(threads)
	m.setSize(100, 20)
	m.resolved[0] = true
	m.state = resolveStateDone

	view := m.View()
	if !strings.Contains(view, "resolved") {
		t.Error("missing 'resolved' in done view")
	}
	// Resolved thread should show [✓].
	if !strings.Contains(view, "✓") {
		t.Error("missing resolved checkmark")
	}
}

func TestResolveCheckboxRendering(t *testing.T) {
	threads := []domain.ReviewThread{
		makeThread("PRRT_sel", "selected.go", 1, true),
		makeThread("PRRT_unsel", "unselected.go", 2, true),
		makeThread("PRRT_noperm", "noperm.go", 3, false),
	}
	m := newResolveModel(threads)
	m.setSize(100, 20)
	m.selected[0] = true

	view := m.View()
	// Selected: [✓]
	if !strings.Contains(view, "[✓]") {
		t.Error("missing selected checkbox [✓]")
	}
	// Unselected: [ ]
	if !strings.Contains(view, "[ ]") {
		t.Error("missing unselected checkbox [ ]")
	}
	// No permission: [-]
	if !strings.Contains(view, "[-]") {
		t.Error("missing no-permission checkbox [-]")
	}
}

func TestResolvePluralS(t *testing.T) {
	if pluralS(1) != "" {
		t.Errorf("pluralS(1) = %q, want empty", pluralS(1))
	}
	if pluralS(2) != "s" {
		t.Errorf("pluralS(2) = %q, want 's'", pluralS(2))
	}
	if pluralS(0) != "s" {
		t.Errorf("pluralS(0) = %q, want 's'", pluralS(0))
	}
}

func TestTruncateID(t *testing.T) {
	if got := truncateID("short"); got != "short" {
		t.Errorf("truncateID(short) = %q", got)
	}
	long := "PRRT_kwDOQQ76Ts5iIWqn"
	got := truncateID(long)
	runes := []rune(got)
	if len(runes) > 17 { // 16 chars + "…"
		t.Errorf("truncateID too long: %q (runes=%d)", got, len(runes))
	}
	if !strings.HasSuffix(got, "…") {
		t.Errorf("truncateID missing ellipsis: %q", got)
	}
}

func TestResolveAppIntegration(t *testing.T) {
	app := NewApp("owner/repo", 42, ViewResolve)
	app.SetComments(&domain.CommentsResult{
		Threads: []domain.ReviewThread{
			makeThread("PRRT_aaa", "main.go", 10, true),
			makeThread("PRRT_bbb", "api.go", 23, true),
		},
		UnresolvedCount: 2,
	})
	app = sendWindowSize(app, 100, 30)

	view := app.View()

	// Status bar should show "resolve mode" and count.
	if !strings.Contains(view, "resolve mode") {
		t.Error("missing 'resolve mode' in status bar")
	}
	if !strings.Contains(view, "2 unresolved") {
		t.Error("missing '2 unresolved' in status bar")
	}

	// Thread paths should appear.
	if !strings.Contains(view, "main.go") {
		t.Error("missing 'main.go' in resolve view")
	}
	if !strings.Contains(view, "api.go") {
		t.Error("missing 'api.go' in resolve view")
	}

	// Help bar should show resolve-specific bindings.
	if !strings.Contains(view, "toggle select") || !strings.Contains(view, "select all") {
		t.Error("missing resolve key bindings in help bar")
	}
}
