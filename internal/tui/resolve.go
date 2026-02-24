package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/indrasvat/gh-ghent/internal/domain"
	"github.com/indrasvat/gh-ghent/internal/tui/components"
	"github.com/indrasvat/gh-ghent/internal/tui/styles"
)

// Resolve-local reusable styles.
var (
	dimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color(string(styles.Dim)))
	greenStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color(string(styles.Green)))
	redStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color(string(styles.Red)))
	cursorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(string(styles.Blue))).Bold(true)
)

// ── Resolve view states ──────────────────────────────────────────

type resolveState int

const (
	resolveStateBrowsing   resolveState = iota // Normal multi-select browsing
	resolveStateConfirming                     // Confirmation bar visible
	resolveStateResolving                      // Mutations in progress
	resolveStateDone                           // All mutations complete
)

// ── Messages ─────────────────────────────────────────────────────

// resolveThreadMsg is emitted after a single thread resolve completes.
type resolveThreadMsg struct {
	threadID string
	err      error
}

// resolveAllDoneMsg is emitted when all resolve mutations complete.
type resolveAllDoneMsg struct {
	successCount int
	failureCount int
}

// resolveRequestMsg requests the App to resolve threads via the API.
type resolveRequestMsg struct {
	threadIDs []string
}

// ── Resolve model ────────────────────────────────────────────────

// resolveModel renders a multi-select list of threads with checkboxes.
type resolveModel struct {
	threads  []domain.ReviewThread
	selected map[int]bool // index → selected
	cursor   int
	offset   int // scroll offset
	width    int
	height   int
	state    resolveState

	// Resolution tracking
	resolved map[int]bool   // index → resolved successfully
	errors   map[int]string // index → error message
}

func newResolveModel(threads []domain.ReviewThread) resolveModel {
	return resolveModel{
		threads:  threads,
		selected: make(map[int]bool),
		resolved: make(map[int]bool),
		errors:   make(map[int]string),
	}
}

func (m *resolveModel) setSize(width, height int) {
	m.width = width
	m.height = height
}

// ── Key bindings ─────────────────────────────────────────────────

var resolveKeys = struct {
	Up    key.Binding
	Down  key.Binding
	Space key.Binding
	All   key.Binding
	Enter key.Binding
	Esc   key.Binding
	Yes   key.Binding
	No    key.Binding
	Open  key.Binding
}{
	Up:    key.NewBinding(key.WithKeys("k", "up")),
	Down:  key.NewBinding(key.WithKeys("j", "down")),
	Space: key.NewBinding(key.WithKeys(" ")),
	All:   key.NewBinding(key.WithKeys("a")),
	Enter: key.NewBinding(key.WithKeys("enter")),
	Esc:   key.NewBinding(key.WithKeys("esc")),
	Yes:   key.NewBinding(key.WithKeys("y")),
	No:    key.NewBinding(key.WithKeys("n")),
	Open:  key.NewBinding(key.WithKeys("o")),
}

// ── Update ───────────────────────────────────────────────────────

func (m resolveModel) Update(msg tea.Msg) (resolveModel, tea.Cmd) {
	switch typedMsg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(typedMsg)
	case resolveThreadMsg:
		return m.handleResolveResult(typedMsg)
	case resolveAllDoneMsg:
		m.state = resolveStateDone
		return m, nil
	}
	return m, nil
}

func (m resolveModel) handleKey(msg tea.KeyMsg) (resolveModel, tea.Cmd) {
	if len(m.threads) == 0 {
		return m, nil
	}

	// In confirming state, only accept y/n/esc/enter.
	if m.state == resolveStateConfirming {
		return m.handleConfirmKey(msg)
	}

	// In resolving or done state, no key input.
	if m.state == resolveStateResolving || m.state == resolveStateDone {
		return m, nil
	}

	switch {
	case key.Matches(msg, resolveKeys.Down):
		m.moveCursor(1)
	case key.Matches(msg, resolveKeys.Up):
		m.moveCursor(-1)
	case key.Matches(msg, resolveKeys.Space):
		m.toggleSelected(m.cursor)
	case key.Matches(msg, resolveKeys.All):
		m.selectAll()
	case key.Matches(msg, resolveKeys.Enter):
		if m.selectedCount() > 0 {
			m.state = resolveStateConfirming
		}
	case key.Matches(msg, resolveKeys.Open):
		if m.cursor >= 0 && m.cursor < len(m.threads) {
			t := m.threads[m.cursor]
			if len(t.Comments) > 0 {
				return m, openInBrowser(t.Comments[0].URL)
			}
		}
	}
	return m, nil
}

func (m resolveModel) handleConfirmKey(msg tea.KeyMsg) (resolveModel, tea.Cmd) {
	switch {
	case key.Matches(msg, resolveKeys.Enter), key.Matches(msg, resolveKeys.Yes):
		// Proceed with resolution.
		m.state = resolveStateResolving
		return m, m.resolveSelectedCmd()
	case key.Matches(msg, resolveKeys.Esc), key.Matches(msg, resolveKeys.No):
		m.state = resolveStateBrowsing
		return m, nil
	}
	return m, nil
}

func (m resolveModel) handleResolveResult(msg resolveThreadMsg) (resolveModel, tea.Cmd) {
	for i, t := range m.threads {
		if t.ID == msg.threadID {
			if msg.err != nil {
				m.errors[i] = msg.err.Error()
			} else {
				m.resolved[i] = true
			}
			break
		}
	}

	// Check if all selected are done.
	allDone := true
	for i := range m.threads {
		if m.selected[i] && !m.resolved[i] && m.errors[i] == "" {
			allDone = false
			break
		}
	}
	if allDone {
		return m, func() tea.Msg {
			return resolveAllDoneMsg{
				successCount: len(m.resolved),
				failureCount: len(m.errors),
			}
		}
	}
	return m, nil
}

// resolveSelectedCmd returns a tea.Cmd that emits a resolveRequestMsg.
// The App layer intercepts this to execute the actual API calls.
func (m resolveModel) resolveSelectedCmd() tea.Cmd {
	var ids []string
	for i, t := range m.threads {
		if m.selected[i] {
			ids = append(ids, t.ID)
		}
	}
	return func() tea.Msg {
		return resolveRequestMsg{threadIDs: ids}
	}
}

// ── Cursor / selection helpers ───────────────────────────────────

func (m *resolveModel) moveCursor(delta int) {
	if len(m.threads) == 0 {
		return
	}
	next := m.cursor + delta
	if next < 0 {
		next = 0
	}
	if next >= len(m.threads) {
		next = len(m.threads) - 1
	}
	m.cursor = next
	m.ensureVisible()
}

func (m *resolveModel) ensureVisible() {
	linesPerItem := 2 // each thread takes 2 lines
	cursorLine := m.cursor * linesPerItem
	visible := m.visibleLines()
	if cursorLine < m.offset {
		m.offset = cursorLine
	}
	if cursorLine+linesPerItem > m.offset+visible {
		m.offset = cursorLine + linesPerItem - visible
	}
	m.offset = max(m.offset, 0)
}

func (m resolveModel) visibleLines() int {
	h := m.height
	// Reserve lines for confirmation bar or status message.
	if m.state == resolveStateConfirming || m.state == resolveStateDone || m.state == resolveStateResolving {
		h -= 2
	}
	if h < 1 {
		h = 1
	}
	return h
}

func (m *resolveModel) toggleSelected(idx int) {
	if idx < 0 || idx >= len(m.threads) {
		return
	}
	if !m.threads[idx].ViewerCanResolve {
		return
	}
	if m.resolved[idx] {
		return
	}
	m.selected[idx] = !m.selected[idx]
}

func (m *resolveModel) selectAll() {
	allEligible := true
	for i, t := range m.threads {
		if t.ViewerCanResolve && !m.resolved[i] && !m.selected[i] {
			allEligible = false
			break
		}
	}
	if allEligible {
		for i := range m.threads {
			m.selected[i] = false
		}
		return
	}
	for i, t := range m.threads {
		if t.ViewerCanResolve && !m.resolved[i] {
			m.selected[i] = true
		}
	}
}

func (m resolveModel) selectedCount() int {
	count := 0
	for _, v := range m.selected {
		if v {
			count++
		}
	}
	return count
}

// ── View ─────────────────────────────────────────────────────────

func (m resolveModel) View() string {
	if len(m.threads) == 0 {
		return dimStyle.Render(" No review threads to resolve.")
	}

	var lines []string
	linesPerItem := 2

	for i, t := range m.threads {
		startLine := i * linesPerItem
		endLine := startLine + linesPerItem

		if endLine <= m.offset {
			continue
		}
		if startLine >= m.offset+m.visibleLines() {
			break
		}

		lines = append(lines, m.renderThread(i, t)...)
	}

	content := strings.Join(lines, "\n")

	lineCount := strings.Count(content, "\n") + 1
	visible := m.visibleLines()
	if lineCount < visible {
		content += strings.Repeat("\n", visible-lineCount)
	}

	switch m.state {
	case resolveStateConfirming:
		content += "\n" + m.renderConfirmBar()
	case resolveStateResolving:
		content += "\n" + m.renderResolvingStatus()
	case resolveStateDone:
		content += "\n" + m.renderDoneStatus()
	}

	return content
}

func (m resolveModel) renderThread(idx int, t domain.ReviewThread) []string {
	isCursor := idx == m.cursor
	isSelected := m.selected[idx]
	isResolved := m.resolved[idx]
	hasError := m.errors[idx] != ""
	canResolve := t.ViewerCanResolve

	// ── Line 1: checkbox + file:line — author ──
	var checkbox string
	switch {
	case isResolved:
		checkbox = greenStyle.Render("[✓]")
	case hasError:
		checkbox = redStyle.Render("[✗]")
	case !canResolve:
		checkbox = dimStyle.Render("[-]")
	case isSelected:
		checkbox = greenStyle.Render("[✓]")
	default:
		checkbox = dimStyle.Render("[ ]")
	}

	cursor := "  "
	if isCursor {
		cursor = cursorStyle.Render("▶") + " "
	}

	fileLine := styles.FilePath.Render(t.Path) +
		styles.LineNumber.Render(fmt.Sprintf(":%d", t.Line))

	author := ""
	if len(t.Comments) > 0 {
		author = dimStyle.Render(" — ") + styles.Author.Render("@"+t.Comments[0].Author)
	}

	permLabel := ""
	if !canResolve {
		permLabel = dimStyle.Render(" (no permission)")
	}

	line1 := cursor + checkbox + " " + fileLine + author + permLabel

	threadID := styles.ThreadID.Render(truncateID(t.ID))
	line1 = padWithRight(line1, threadID, m.width)

	// ── Line 2: body preview ──
	body := ""
	if len(t.Comments) > 0 {
		body = truncateBody(stripMarkdown(t.Comments[0].Body), m.width-8)
	}
	line2 := "     " + dimStyle.Render(body)

	if isCursor {
		bg := lipgloss.NewStyle().Background(lipgloss.Color(string(styles.Surface2)))
		line1 = bg.Render(components.PadLine(line1, m.width))
		line2 = bg.Render(components.PadLine(line2, m.width))
	}

	return []string{line1, line2}
}

func (m resolveModel) renderConfirmBar() string {
	count := m.selectedCount()
	prompt := greenStyle.Bold(true).
		Render(fmt.Sprintf("Resolve %d thread%s?", count, pluralS(count)))

	hint := dimStyle.Render("  Press ") +
		styles.HelpKey.Render("enter") +
		dimStyle.Render(" to confirm, ") +
		styles.HelpKey.Render("esc") +
		dimStyle.Render(" to cancel")

	return " " + prompt + hint + styles.ANSIReset
}

func (m resolveModel) renderResolvingStatus() string {
	done := len(m.resolved) + len(m.errors)
	total := m.selectedCount()
	return " " + styles.StatusBarDim.Render(
		fmt.Sprintf("⟳ Resolving... %d/%d", done, total)) + styles.ANSIReset
}

func (m resolveModel) renderDoneStatus() string {
	success := len(m.resolved)
	failures := len(m.errors)
	var parts []string
	if success > 0 {
		parts = append(parts, greenStyle.Render(
			fmt.Sprintf("✓ %d resolved", success)))
	}
	if failures > 0 {
		parts = append(parts, redStyle.Render(
			fmt.Sprintf("✗ %d failed", failures)))
	}
	return " " + strings.Join(parts, "  ") + styles.ANSIReset
}

// ── Helpers ──────────────────────────────────────────────────────

func truncateID(id string) string {
	if len(id) > 16 {
		return id[:16] + "…"
	}
	return id
}

func truncateBody(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if len(s) > maxWidth {
		return s[:maxWidth-1] + "…"
	}
	return s
}

func pluralS(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// padWithRight places rightText at the end of the line, padding between.
func padWithRight(left, right string, width int) string {
	leftW := lipgloss.Width(left)
	rightW := lipgloss.Width(right)
	gap := width - leftW - rightW
	if gap < 1 {
		return left + " " + right
	}
	return left + strings.Repeat(" ", gap) + right
}
