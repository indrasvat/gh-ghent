package tui

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/indrasvat/gh-ghent/internal/domain"
	"github.com/indrasvat/gh-ghent/internal/tui/styles"
)

// ── Messages ────────────────────────────────────────────────────

// selectThreadMsg is sent when the user presses Enter to expand a thread.
type selectThreadMsg struct {
	threadIdx int
}

// ── List item types ─────────────────────────────────────────────

type listItemKind int

const (
	listItemFileHeader listItemKind = iota
	listItemThread
)

type listItem struct {
	kind     listItemKind
	filePath string               // file header text
	thread   *domain.ReviewThread // nil for headers
	idx      int                  // index into original threads slice
}

// ── Comments list model ─────────────────────────────────────────

// commentsListModel renders a scrollable list of review threads grouped by file.
type commentsListModel struct {
	threads []domain.ReviewThread
	items   []listItem // flattened: file headers + thread rows
	cursor  int        // index into items (only lands on thread items)
	offset  int        // scroll offset for viewport
	width   int
	height  int
}

func newCommentsListModel(threads []domain.ReviewThread) commentsListModel {
	m := commentsListModel{
		threads: threads,
	}
	m.buildItems()
	// Set cursor to the first thread item.
	for i, item := range m.items {
		if item.kind == listItemThread {
			m.cursor = i
			break
		}
	}
	return m
}

// buildItems creates the flattened item list from threads, grouped by file path.
func (m *commentsListModel) buildItems() {
	if len(m.threads) == 0 {
		m.items = nil
		return
	}

	// Group threads by file path.
	groups := make(map[string][]int) // path → thread indices
	var paths []string
	for i := range m.threads {
		p := m.threads[i].Path
		if _, ok := groups[p]; !ok {
			paths = append(paths, p)
		}
		groups[p] = append(groups[p], i)
	}
	sort.Strings(paths)

	var items []listItem
	for _, p := range paths {
		items = append(items, listItem{kind: listItemFileHeader, filePath: p})
		for _, idx := range groups[p] {
			items = append(items, listItem{
				kind:   listItemThread,
				thread: &m.threads[idx],
				idx:    idx,
			})
		}
	}
	m.items = items
}

// setSize sets the viewport dimensions.
func (m *commentsListModel) setSize(w, h int) {
	m.width = w
	m.height = h
}

// selectedThreadIdx returns the index of the focused thread in the original slice.
func (m commentsListModel) selectedThreadIdx() int {
	if m.cursor >= 0 && m.cursor < len(m.items) && m.items[m.cursor].kind == listItemThread {
		return m.items[m.cursor].idx
	}
	return -1
}

// Update handles key events for the comments list.
func (m commentsListModel) Update(msg tea.Msg) (commentsListModel, tea.Cmd) {
	if typedMsg, ok := msg.(tea.KeyMsg); ok {
		switch {
		case key.Matches(typedMsg, commentsKeys.Down):
			m.moveCursor(1)
		case key.Matches(typedMsg, commentsKeys.Up):
			m.moveCursor(-1)
		case key.Matches(typedMsg, commentsKeys.Enter):
			idx := m.selectedThreadIdx()
			if idx >= 0 {
				return m, func() tea.Msg { return selectThreadMsg{threadIdx: idx} }
			}
		}
	}
	return m, nil
}

// moveCursor moves the cursor by delta, skipping file headers.
func (m *commentsListModel) moveCursor(delta int) {
	if len(m.items) == 0 {
		return
	}

	start := m.cursor
	pos := start
	for {
		pos += delta
		if pos < 0 || pos >= len(m.items) {
			return // don't wrap
		}
		if m.items[pos].kind == listItemThread {
			m.cursor = pos
			m.ensureVisible()
			return
		}
	}
}

// itemScreenLines returns the number of screen lines an item occupies.
// Thread items render 3 lines (marker + body + thread ID); headers render 1.
func (m *commentsListModel) itemScreenLines(i int) int {
	if i >= 0 && i < len(m.items) && m.items[i].kind == listItemThread {
		return 3
	}
	return 1
}

// screenLinesBetween returns total screen lines for items in [from, to).
func (m *commentsListModel) screenLinesBetween(from, to int) int {
	total := 0
	for i := from; i < to && i < len(m.items); i++ {
		total += m.itemScreenLines(i)
	}
	return total
}

// ensureVisible adjusts scroll offset so the cursor item is fully in the visible area.
func (m *commentsListModel) ensureVisible() {
	if m.height <= 0 {
		return
	}
	// Cursor above viewport → scroll up.
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	// Cursor below viewport → scroll down until entire cursor item fits.
	for m.screenLinesBetween(m.offset, m.cursor+1) > m.height {
		m.offset++
	}
}

// View renders the comments list.
func (m commentsListModel) View() string {
	if len(m.items) == 0 {
		return styles.StatusBarDim.Render("  No review threads found.")
	}

	var lines []string

	// Determine visible window. Thread items span 3 screen lines,
	// file headers span 1 — stop adding items when we fill the viewport.
	visibleStart := max(m.offset, 0)
	screenLines := 0
	for i := visibleStart; i < len(m.items) && screenLines < m.height; i++ {
		item := m.items[i]
		switch item.kind {
		case listItemFileHeader:
			lines = append(lines, m.renderFileHeader(item.filePath))
			screenLines++
		case listItemThread:
			lines = append(lines, m.renderThreadRow(item, i == m.cursor))
			screenLines += 3 // marker + body + thread ID
		}
	}

	result := strings.Join(lines, "\n")

	// Pad remaining height with empty lines.
	// Count actual screen lines, not items — thread items span 3 lines each.
	actualLines := strings.Count(result, "\n") + 1
	if actualLines < m.height {
		result += strings.Repeat("\n", m.height-actualLines)
	}

	return result
}

// renderFileHeader renders a file path separator row.
func (m commentsListModel) renderFileHeader(path string) string {
	fileStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(string(styles.Cyan))).
		Bold(true)

	label := " " + fileStyle.Render(path) + " "

	// Fill remaining width with dim dash.
	labelW := lipgloss.Width(label)
	remaining := m.width - labelW
	remaining = max(remaining, 0)
	sep := styles.StatusBarDim.Render(strings.Repeat("─", remaining))

	return label + sep + styles.ANSIReset
}

// renderThreadRow renders a single thread list item matching the mockup layout:
//
//	Line 1:  ▶  :47 — @reviewer1  2h ago
//	Line 2:      This error should be wrapped with context...
//	Line 3:      PRRT_kwDON1... · 2 replies
func (m commentsListModel) renderThreadRow(item listItem, isCursor bool) string {
	t := item.thread

	// Cursor indicator.
	var marker string
	if isCursor {
		marker = lipgloss.NewStyle().
			Foreground(lipgloss.Color(string(styles.Blue))).
			Render("▶")
	} else {
		marker = " "
	}

	// Line number range.
	lineStr := fmt.Sprintf(":%d", t.Line)
	if t.StartLine > 0 && t.StartLine != t.Line {
		lineStr = fmt.Sprintf(":%d-%d", t.StartLine, t.Line)
	}
	fileLine := styles.LineNumber.Render(lineStr)

	// Author.
	author := ""
	if len(t.Comments) > 0 {
		author = styles.Author.Render("@" + t.Comments[0].Author)
	}

	// Time ago.
	timeAgo := ""
	if len(t.Comments) > 0 && !t.Comments[0].CreatedAt.IsZero() {
		timeAgo = styles.StatusBarDim.Render(formatTimeAgo(t.Comments[0].CreatedAt))
	}

	// ── LINE 1: marker  :line — @author  2h ago ──
	line1Parts := []string{" " + marker, fileLine}
	if author != "" {
		line1Parts = append(line1Parts, styles.StatusBarDim.Render("—"), author)
	}
	if timeAgo != "" {
		line1Parts = append(line1Parts, timeAgo)
	}
	line1 := strings.Join(line1Parts, " ")

	// ── LINE 2: body preview (markdown stripped) ──
	body := ""
	if len(t.Comments) > 0 {
		bodyText := stripMarkdown(t.Comments[0].Body)
		maxBody := max(m.width-10, 20)
		body = styles.Truncate(bodyText, maxBody)
	}
	line2 := "     " + body

	// ── LINE 3: thread ID · N replies ──
	var metaParts []string
	if t.ID != "" {
		idDisplay := t.ID
		if len(idDisplay) > 14 {
			idDisplay = idDisplay[:14] + "..."
		}
		metaParts = append(metaParts, styles.ThreadID.Render(idDisplay))
	}
	if len(t.Comments) > 1 {
		n := len(t.Comments) - 1
		if n == 1 {
			metaParts = append(metaParts, "1 reply")
		} else {
			metaParts = append(metaParts, fmt.Sprintf("%d replies", n))
		}
	}
	line3 := "     " + styles.StatusBarDim.Render(strings.Join(metaParts, " · "))

	var content string
	if isCursor {
		content = styles.ListItemSelected.Render(line1+"\n"+line2+"\n"+line3) + styles.ANSIReset
	} else {
		content = styles.ListItemNormal.Render(line1+"\n"+line2+"\n"+line3) + styles.ANSIReset
	}

	return content
}

// ── Helpers ─────────────────────────────────────────────────────

// formatTimeAgo returns a human-readable relative time string.
func formatTimeAgo(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		m := int(d.Minutes())
		return fmt.Sprintf("%dm ago", m)
	case d < 24*time.Hour:
		h := int(d.Hours())
		return fmt.Sprintf("%dh ago", h)
	case d < 30*24*time.Hour:
		days := int(d.Hours() / 24)
		return fmt.Sprintf("%dd ago", days)
	default:
		months := int(d.Hours() / 24 / 30)
		if months < 12 {
			return fmt.Sprintf("%dmo ago", months)
		}
		return fmt.Sprintf("%dy ago", int(d.Hours()/24/365))
	}
}

// Compiled regex patterns for markdown stripping.
var (
	reMarkdownImage  = regexp.MustCompile(`!\[[^\]]*\]\([^)]*\)`)
	reMarkdownLink   = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)
	reHTMLTags       = regexp.MustCompile(`<[^>]+>`)
	reMultipleSpaces = regexp.MustCompile(`\s{2,}`)
)

// stripMarkdown removes common markdown/HTML artifacts from comment body for preview.
func stripMarkdown(s string) string {
	// Flatten to single line.
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")

	// Remove images ![alt](url).
	s = reMarkdownImage.ReplaceAllString(s, "")

	// Convert links [text](url) → text.
	s = reMarkdownLink.ReplaceAllString(s, "$1")

	// Remove HTML tags.
	s = reHTMLTags.ReplaceAllString(s, "")

	// Remove bold/italic markers.
	s = strings.ReplaceAll(s, "***", "")
	s = strings.ReplaceAll(s, "**", "")
	s = strings.ReplaceAll(s, "__", "")

	// Remove inline code backticks.
	s = strings.ReplaceAll(s, "`", "")

	// Collapse multiple spaces.
	s = reMultipleSpaces.ReplaceAllString(s, " ")

	return strings.TrimSpace(s)
}

// ── Key bindings ────────────────────────────────────────────────

type commentsKeyMap struct {
	Up    key.Binding
	Down  key.Binding
	Enter key.Binding
}

var commentsKeys = commentsKeyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("j", "down"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "expand"),
	),
}
