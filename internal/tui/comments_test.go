package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/indrasvat/gh-ghent/internal/domain"
)

func testThreads() []domain.ReviewThread {
	return []domain.ReviewThread{
		{
			ID: "PRRT_1", Path: "internal/api/graphql.go", Line: 47,
			Comments: []domain.Comment{
				{Author: "reviewer1", Body: "Wrap this error with context."},
				{Author: "you", Body: "Fixed, thanks!"},
			},
		},
		{
			ID: "PRRT_2", Path: "internal/api/graphql.go", Line: 102,
			Comments: []domain.Comment{
				{Author: "reviewer2", Body: "This should use a context parameter."},
			},
		},
		{
			ID: "PRRT_3", Path: "internal/cli/comments.go", Line: 23,
			Comments: []domain.Comment{
				{Author: "reviewer1", Body: "Consider using tableprinter."},
			},
		},
	}
}

func TestCommentsListModelItemCount(t *testing.T) {
	m := newCommentsListModel(testThreads())
	// 2 file headers + 3 thread items = 5
	if len(m.items) != 5 {
		t.Errorf("expected 5 items, got %d", len(m.items))
	}
}

func TestCommentsListModelFileGrouping(t *testing.T) {
	m := newCommentsListModel(testThreads())

	// First item should be a file header for "internal/api/graphql.go"
	if m.items[0].kind != listItemFileHeader {
		t.Error("expected first item to be file header")
	}
	if m.items[0].filePath != "internal/api/graphql.go" {
		t.Errorf("expected path 'internal/api/graphql.go', got %q", m.items[0].filePath)
	}

	// Second group header for "internal/cli/comments.go"
	if m.items[3].kind != listItemFileHeader {
		t.Error("expected fourth item to be file header")
	}
	if m.items[3].filePath != "internal/cli/comments.go" {
		t.Errorf("expected path 'internal/cli/comments.go', got %q", m.items[3].filePath)
	}
}

func TestCommentsListModelCursorStartsOnFirstThread(t *testing.T) {
	m := newCommentsListModel(testThreads())
	// Cursor should skip the first file header and land on the first thread.
	if m.cursor != 1 {
		t.Errorf("expected cursor at 1 (first thread), got %d", m.cursor)
	}
	if m.items[m.cursor].kind != listItemThread {
		t.Error("cursor should be on a thread item")
	}
}

func TestCommentsListModelMoveDown(t *testing.T) {
	m := newCommentsListModel(testThreads())
	m.setSize(80, 24)

	// Start on item 1 (first thread).
	if m.cursor != 1 {
		t.Fatalf("expected cursor at 1, got %d", m.cursor)
	}

	// Move down → item 2 (second thread, same file).
	m.moveCursor(1)
	if m.cursor != 2 {
		t.Errorf("expected cursor at 2, got %d", m.cursor)
	}

	// Move down → should skip file header at 3, land on item 4 (third thread).
	m.moveCursor(1)
	if m.cursor != 4 {
		t.Errorf("expected cursor at 4 (skipping header), got %d", m.cursor)
	}
}

func TestCommentsListModelMoveUp(t *testing.T) {
	m := newCommentsListModel(testThreads())
	m.setSize(80, 24)

	// Move to last thread.
	m.moveCursor(1)
	m.moveCursor(1)
	if m.cursor != 4 {
		t.Fatalf("expected cursor at 4, got %d", m.cursor)
	}

	// Move up → should skip file header at 3, land on item 2.
	m.moveCursor(-1)
	if m.cursor != 2 {
		t.Errorf("expected cursor at 2 (skipping header), got %d", m.cursor)
	}

	// Move up → item 1.
	m.moveCursor(-1)
	if m.cursor != 1 {
		t.Errorf("expected cursor at 1, got %d", m.cursor)
	}

	// Move up from first thread → stays at 1 (no wrapping).
	m.moveCursor(-1)
	if m.cursor != 1 {
		t.Errorf("expected cursor to stay at 1, got %d", m.cursor)
	}
}

func TestCommentsListModelEmptyList(t *testing.T) {
	m := newCommentsListModel(nil)
	m.setSize(80, 24)

	if len(m.items) != 0 {
		t.Error("expected empty items for nil threads")
	}

	output := m.View()
	if !strings.Contains(output, "No review threads") {
		t.Error("expected empty state message")
	}
}

func TestCommentsListModelViewRendersThreads(t *testing.T) {
	m := newCommentsListModel(testThreads())
	m.setSize(120, 30)

	output := m.View()

	// Should contain file paths as headers.
	if !strings.Contains(output, "internal/api/graphql.go") {
		t.Error("missing file header 'internal/api/graphql.go'")
	}
	if !strings.Contains(output, "internal/cli/comments.go") {
		t.Error("missing file header 'internal/cli/comments.go'")
	}

	// Should contain line numbers.
	if !strings.Contains(output, ":47") {
		t.Error("missing line number :47")
	}
	if !strings.Contains(output, ":23") {
		t.Error("missing line number :23")
	}

	// Should contain authors.
	if !strings.Contains(output, "@reviewer1") {
		t.Error("missing author @reviewer1")
	}

	// Should contain body preview.
	if !strings.Contains(output, "Wrap this error") {
		t.Error("missing body preview text")
	}

	// Should contain cursor marker on first thread.
	if !strings.Contains(output, "▶") {
		t.Error("missing cursor marker")
	}
}

func TestCommentsListModelViewRendersReplyCount(t *testing.T) {
	m := newCommentsListModel(testThreads())
	m.setSize(120, 30)

	output := m.View()

	// First thread has 2 comments → "1 reply" (with · separator)
	if !strings.Contains(output, "1 reply") {
		t.Error("missing reply count for first thread")
	}
}

func TestCommentsListModelViewRendersThreadID(t *testing.T) {
	m := newCommentsListModel(testThreads())
	m.setSize(120, 30)

	output := m.View()

	if !strings.Contains(output, "PRRT_1") {
		t.Error("missing thread ID PRRT_1")
	}
}

func TestCommentsListUpdateEnterReturnsSelectMsg(t *testing.T) {
	m := newCommentsListModel(testThreads())
	m.setSize(80, 24)

	// Press Enter on the first thread.
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected a command from Enter, got nil")
	}

	msg := cmd()
	selectMsg, ok := msg.(selectThreadMsg)
	if !ok {
		t.Fatalf("expected selectThreadMsg, got %T", msg)
	}
	if selectMsg.threadIdx != 0 {
		t.Errorf("expected threadIdx 0, got %d", selectMsg.threadIdx)
	}
}

func TestCommentsListUpdateJKNavigation(t *testing.T) {
	m := newCommentsListModel(testThreads())
	m.setSize(80, 24)

	// Initial cursor at first thread (index 1).
	if m.cursor != 1 {
		t.Fatalf("expected cursor at 1, got %d", m.cursor)
	}

	// Press 'j' to move down.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.cursor != 2 {
		t.Errorf("expected cursor at 2 after j, got %d", m.cursor)
	}

	// Press 'k' to move back up.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if m.cursor != 1 {
		t.Errorf("expected cursor at 1 after k, got %d", m.cursor)
	}
}

func TestCommentsListSelectedThreadIdx(t *testing.T) {
	m := newCommentsListModel(testThreads())

	// First thread should be at index 0 in the original slice.
	idx := m.selectedThreadIdx()
	if idx != 0 {
		t.Errorf("expected selectedThreadIdx 0, got %d", idx)
	}

	// Move to second thread.
	m.moveCursor(1)
	idx = m.selectedThreadIdx()
	if idx != 1 {
		t.Errorf("expected selectedThreadIdx 1, got %d", idx)
	}

	// Move to third thread (different file).
	m.moveCursor(1)
	idx = m.selectedThreadIdx()
	if idx != 2 {
		t.Errorf("expected selectedThreadIdx 2, got %d", idx)
	}
}

func TestCommentsListScrolling(t *testing.T) {
	// Create enough threads to exceed visible height.
	threads := make([]domain.ReviewThread, 10)
	for i := range threads {
		threads[i] = domain.ReviewThread{
			ID: "t", Path: "file.go", Line: i + 1,
			Comments: []domain.Comment{{Author: "alice", Body: "comment"}},
		}
	}

	m := newCommentsListModel(threads)
	m.setSize(80, 5) // very small viewport

	// Move cursor down past the visible area.
	for range 8 {
		m.moveCursor(1)
	}

	// Offset should have adjusted to keep cursor visible.
	if m.offset <= 0 {
		t.Errorf("expected positive offset after scrolling, got %d", m.offset)
	}
}

func TestFormatTimeAgo(t *testing.T) {
	tests := []struct {
		ago  time.Duration
		want string
	}{
		{30 * time.Second, "just now"},
		{5 * time.Minute, "5m ago"},
		{45 * time.Minute, "45m ago"},
		{2 * time.Hour, "2h ago"},
		{23 * time.Hour, "23h ago"},
		{3 * 24 * time.Hour, "3d ago"},
		{60 * 24 * time.Hour, "2mo ago"},
		{400 * 24 * time.Hour, "1y ago"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := formatTimeAgo(time.Now().Add(-tt.ago))
			if got != tt.want {
				t.Errorf("formatTimeAgo(-%v) = %q, want %q", tt.ago, got, tt.want)
			}
		})
	}
}

func TestStripMarkdown(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		// Images removed entirely.
		{"Check ![badge](https://img.shields.io/badge.svg) here", "Check here"},
		// Links converted to text.
		{"See [this docs](https://example.com) for details", "See this docs for details"},
		// HTML tags removed.
		{"<sub><sub>text</sub></sub>", "text"},
		// Bold/italic markers removed.
		{"This is **bold** and __underlined__", "This is bold and underlined"},
		// Triple asterisks.
		{"***important***", "important"},
		// Backticks removed.
		{"Use `fmt.Errorf` here", "Use fmt.Errorf here"},
		// Newlines flattened.
		{"line1\nline2\nline3", "line1 line2 line3"},
		// Multiple spaces collapsed.
		{"too    many   spaces", "too many spaces"},
		// Combined.
		{
			"***<sub>![P2](url)</sub> Propagate**",
			"Propagate",
		},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := stripMarkdown(tt.input)
			if got != tt.want {
				t.Errorf("stripMarkdown(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestCommentsListModelViewRendersTimeAgo(t *testing.T) {
	threads := []domain.ReviewThread{
		{
			ID: "PRRT_1", Path: "file.go", Line: 10,
			Comments: []domain.Comment{
				{Author: "alice", Body: "fix this", CreatedAt: time.Now().Add(-2 * time.Hour)},
			},
		},
	}
	m := newCommentsListModel(threads)
	m.setSize(120, 30)

	output := m.View()
	if !strings.Contains(output, "2h ago") {
		t.Error("missing time-ago in thread rendering")
	}
}

func TestItemScreenLines(t *testing.T) {
	m := newCommentsListModel(testThreads())
	// File header = 1 line.
	if got := m.itemScreenLines(0); got != 1 {
		t.Errorf("header item: got %d lines, want 1", got)
	}
	// Thread item = 3 lines.
	if got := m.itemScreenLines(1); got != 3 {
		t.Errorf("thread item: got %d lines, want 3", got)
	}
}

func TestScreenLinesBetween(t *testing.T) {
	m := newCommentsListModel(testThreads())
	// items: [header, thread, thread, header, thread]
	// lines: [1, 3, 3, 1, 3] = 11 total
	if got := m.screenLinesBetween(0, len(m.items)); got != 11 {
		t.Errorf("screenLinesBetween(0, %d) = %d, want 11", len(m.items), got)
	}
	// First two items: header(1) + thread(3) = 4
	if got := m.screenLinesBetween(0, 2); got != 4 {
		t.Errorf("screenLinesBetween(0, 2) = %d, want 4", got)
	}
}
