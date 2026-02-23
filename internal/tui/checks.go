package tui

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/indrasvat/gh-ghent/internal/domain"
	"github.com/indrasvat/gh-ghent/internal/tui/styles"
)

// ── Messages ────────────────────────────────────────────────────

// selectCheckMsg is sent when the user presses Enter to view a check's log.
type selectCheckMsg struct {
	checkIdx int
}

// ── Checks list model ───────────────────────────────────────────

// checksListModel renders a scrollable list of CI check runs with status icons
// and auto-expanded annotations for failing checks.
type checksListModel struct {
	checks []domain.CheckRun
	cursor int // index into checks slice
	offset int // scroll offset (check index, not screen line)
	width  int
	height int
}

func newChecksListModel(checks []domain.CheckRun) checksListModel {
	return checksListModel{
		checks: checks,
	}
}

func (m *checksListModel) setSize(w, h int) {
	m.width = w
	m.height = h
}

func (m checksListModel) selectedCheckIdx() int {
	if m.cursor >= 0 && m.cursor < len(m.checks) {
		return m.cursor
	}
	return -1
}

// screenLinesForCheck returns the number of screen lines a check occupies.
// Base: 1 line. Failed checks with annotations: 1 + header + annotation count.
func (m *checksListModel) screenLinesForCheck(i int) int {
	if i < 0 || i >= len(m.checks) {
		return 1
	}
	lines := 1
	ch := m.checks[i]
	if checkIsFailed(ch) && len(ch.Annotations) > 0 {
		lines++ // annotation count header
		lines += len(ch.Annotations)
	}
	return lines
}

// ensureVisible adjusts scroll offset so the cursor check is fully visible.
func (m *checksListModel) ensureVisible() {
	if m.height <= 0 {
		return
	}
	// Cursor above viewport → scroll up.
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	// Cursor below viewport → scroll down until cursor check fits.
	for {
		totalLines := 0
		for i := m.offset; i <= m.cursor && i < len(m.checks); i++ {
			totalLines += m.screenLinesForCheck(i)
		}
		if totalLines <= m.height || m.offset >= m.cursor {
			break
		}
		m.offset++
	}
}

// Update handles key events for the checks list.
func (m checksListModel) Update(msg tea.Msg) (checksListModel, tea.Cmd) {
	if typedMsg, ok := msg.(tea.KeyMsg); ok {
		switch {
		case key.Matches(typedMsg, checksKeys.Down):
			if m.cursor < len(m.checks)-1 {
				m.cursor++
				m.ensureVisible()
			}
		case key.Matches(typedMsg, checksKeys.Up):
			if m.cursor > 0 {
				m.cursor--
				m.ensureVisible()
			}
		case key.Matches(typedMsg, checksKeys.Enter),
			key.Matches(typedMsg, checksKeys.ViewLog):
			idx := m.selectedCheckIdx()
			if idx >= 0 {
				return m, func() tea.Msg { return selectCheckMsg{checkIdx: idx} }
			}
		case key.Matches(typedMsg, checksKeys.Open):
			idx := m.selectedCheckIdx()
			if idx >= 0 && m.checks[idx].HTMLURL != "" {
				return m, openInBrowser(m.checks[idx].HTMLURL)
			}
		}
	}
	return m, nil
}

// View renders the checks list.
func (m checksListModel) View() string {
	if len(m.checks) == 0 {
		return styles.StatusBarDim.Render("  No check runs found.")
	}

	var lines []string
	screenLines := 0

	for i := m.offset; i < len(m.checks) && screenLines < m.height; i++ {
		ch := m.checks[i]
		isCursor := i == m.cursor

		// Render check row.
		lines = append(lines, m.renderCheckRow(ch, isCursor))
		screenLines++

		// Auto-expand annotations for failed checks.
		if checkIsFailed(ch) && len(ch.Annotations) > 0 && screenLines < m.height {
			lines = append(lines, m.renderAnnotationHeader(ch))
			screenLines++

			for _, a := range ch.Annotations {
				if screenLines >= m.height {
					break
				}
				lines = append(lines, m.renderAnnotation(a))
				screenLines++
			}
		}
	}

	result := strings.Join(lines, "\n")

	// Pad remaining height with empty lines.
	actualLines := strings.Count(result, "\n") + 1
	if actualLines < m.height {
		result += strings.Repeat("\n", m.height-actualLines)
	}

	return result
}

// renderCheckRow renders a single check run row matching the mockup:
//
//	  ✓  build (ubuntu-latest, go-1.26)              42s   passed
//	▶ ✗  lint (golangci-lint)                         28s   failed
func (m checksListModel) renderCheckRow(ch domain.CheckRun, isCursor bool) string {
	// Status icon.
	icon := checkStatusIcon(ch)

	// Name — bold red for failed.
	name := ch.Name
	nameStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(string(styles.Text)))
	if checkIsFailed(ch) {
		nameStyle = nameStyle.Foreground(lipgloss.Color(string(styles.Red))).Bold(true)
	}
	nameStr := nameStyle.Render(name)

	// Duration.
	durStr := styles.StatusBarDim.Render(formatCheckDuration(ch))

	// Status text.
	statusStr := renderCheckStatusText(ch)

	// Build row: " cursor_or_space  icon  name ... dur  status"
	var leftPart string
	if isCursor {
		marker := lipgloss.NewStyle().
			Foreground(lipgloss.Color(string(styles.Blue))).
			Render("▶")
		leftPart = " " + marker + " " + icon + " " + nameStr
	} else {
		leftPart = "   " + icon + " " + nameStr
	}

	// Right-align duration and status.
	rightPart := durStr + "  " + statusStr
	leftW := lipgloss.Width(leftPart)
	rightW := lipgloss.Width(rightPart)
	gap := m.width - leftW - rightW - 2
	gap = max(gap, 1)

	fullRow := leftPart + styles.Pad(gap) + rightPart

	if isCursor {
		return styles.ListItemSelected.Render(fullRow) + styles.ANSIReset
	}
	return styles.ListItemNormal.Render(fullRow) + styles.ANSIReset
}

// renderAnnotationHeader renders the "N errors" header below a failed check.
func (m checksListModel) renderAnnotationHeader(ch domain.CheckRun) string {
	count := len(ch.Annotations)
	label := fmt.Sprintf("%d error", count)
	if count != 1 {
		label += "s"
	}
	return "      " + styles.CheckFail.Render(label) + styles.ANSIReset
}

// renderAnnotation renders a single annotation row:
//
//	● internal/api/rest.go:23 [errcheck] Error return value...
func (m checksListModel) renderAnnotation(a domain.Annotation) string {
	file := styles.FilePath.Render(a.Path)
	line := styles.LineNumber.Render(fmt.Sprintf(":%d", a.StartLine))

	msg := a.Message
	if a.Title != "" {
		msg = styles.BadgeYellow.Render("["+a.Title+"]") + " " + msg
	}
	maxMsg := max(m.width-30, 20)
	msg = styles.Truncate(msg, maxMsg)

	return "      " + styles.StatusBarDim.Render("●") + " " +
		file + line + " " + msg + styles.ANSIReset
}

// ── Checks log model ────────────────────────────────────────────

// checksLogModel renders the full detail/log view for a single check run
// with annotations and log excerpt in a scrollable viewport.
type checksLogModel struct {
	check   *domain.CheckRun
	content []string // pre-rendered content lines
	offset  int      // scroll offset (line-based viewport)
	width   int
	height  int
}

func newChecksLogModel(check *domain.CheckRun) checksLogModel {
	m := checksLogModel{check: check}
	m.buildContent()
	return m
}

func (m *checksLogModel) setSize(w, h int) {
	m.width = w
	m.height = h
	m.buildContent()
}

func (m *checksLogModel) buildContent() {
	if m.check == nil {
		m.content = nil
		return
	}

	ch := m.check
	var lines []string

	// ── Header: check name + status icon ──
	icon := checkStatusIcon(*ch)
	nameStyle := lipgloss.NewStyle().Bold(true)
	if checkIsFailed(*ch) {
		nameStyle = nameStyle.Foreground(lipgloss.Color(string(styles.Red)))
	} else {
		nameStyle = nameStyle.Foreground(lipgloss.Color(string(styles.Green)))
	}

	lines = append(lines, " "+icon+" "+nameStyle.Render(ch.Name)+styles.ANSIReset)
	lines = append(lines, "")

	// ── Duration + conclusion ──
	dur := formatCheckDuration(*ch)
	conclusion := ch.Conclusion
	if conclusion == "" {
		conclusion = ch.Status
	}
	lines = append(lines, " "+styles.StatusBarDim.Render("Duration: "+dur+"  Status: "+conclusion)+styles.ANSIReset)
	lines = append(lines, "")

	// ── Annotations ──
	if len(ch.Annotations) > 0 {
		count := len(ch.Annotations)
		label := fmt.Sprintf("%d annotation(s)", count)
		lines = append(lines, " "+styles.CheckFail.Render(label)+styles.ANSIReset)
		lines = append(lines, "")

		for _, a := range ch.Annotations {
			file := styles.FilePath.Render(a.Path)
			line := styles.LineNumber.Render(fmt.Sprintf(":%d", a.StartLine))
			lines = append(lines, "  "+styles.StatusBarDim.Render("●")+" "+file+line+styles.ANSIReset)

			if a.Title != "" {
				lines = append(lines, "    "+styles.BadgeYellow.Render("["+a.Title+"]")+" "+a.Message+styles.ANSIReset)
			} else {
				lines = append(lines, "    "+a.Message+styles.ANSIReset)
			}
			lines = append(lines, "")
		}
	}

	// ── Log excerpt ──
	if ch.LogExcerpt != "" {
		lines = append(lines, " "+styles.StatusBarDim.Render("Log excerpt:")+styles.ANSIReset)
		lines = append(lines, "")

		logLines := strings.Split(ch.LogExcerpt, "\n")
		for i, l := range logLines {
			lineNum := styles.StatusBarDim.Render(fmt.Sprintf("%3d", i+1))

			// Color error-relevant lines.
			var lineContent string
			lower := strings.ToLower(l)
			switch {
			case strings.Contains(lower, "error") || strings.Contains(lower, "fail"):
				lineContent = styles.CheckFail.Render(l)
			case strings.Contains(lower, "warn"):
				lineContent = styles.CheckPending.Render(l)
			case l == "...":
				lineContent = styles.StatusBarDim.Render(l)
			default:
				lineContent = l
			}
			lines = append(lines, "  "+lineNum+" "+lineContent+styles.ANSIReset)
		}
	} else if checkIsFailed(*ch) {
		lines = append(lines, " "+styles.StatusBarDim.Render("No log excerpt available.")+styles.ANSIReset)
	}

	m.content = lines
}

// Update handles key events for the checks log viewer.
func (m checksLogModel) Update(msg tea.Msg) (checksLogModel, tea.Cmd) {
	if typedMsg, ok := msg.(tea.KeyMsg); ok {
		switch {
		case key.Matches(typedMsg, logViewKeys.ScrollDown):
			m.scrollDown()
		case key.Matches(typedMsg, logViewKeys.ScrollUp):
			m.scrollUp()
		case key.Matches(typedMsg, logViewKeys.Open):
			if m.check != nil && m.check.HTMLURL != "" {
				return m, openInBrowser(m.check.HTMLURL)
			}
		}
	}
	return m, nil
}

func (m *checksLogModel) scrollDown() {
	maxOffset := max(len(m.content)-m.height, 0)
	if m.offset < maxOffset {
		m.offset++
	}
}

func (m *checksLogModel) scrollUp() {
	if m.offset > 0 {
		m.offset--
	}
}

// View renders the log content with line-based viewport scrolling.
func (m checksLogModel) View() string {
	if len(m.content) == 0 {
		return styles.StatusBarDim.Render("  No check selected.")
	}

	end := min(m.offset+m.height, len(m.content))
	visible := m.content[m.offset:end]

	result := strings.Join(visible, "\n")

	// Pad remaining height.
	visibleCount := len(visible)
	if visibleCount < m.height {
		result += strings.Repeat("\n", m.height-visibleCount)
	}

	return result
}

// ── Helpers ─────────────────────────────────────────────────────

// checkIsFailed returns true if the check run has a failure conclusion.
func checkIsFailed(ch domain.CheckRun) bool {
	if ch.Status != "completed" {
		return false
	}
	switch ch.Conclusion {
	case "failure", "timed_out", "action_required", "startup_failure", "stale", "cancelled":
		return true
	}
	return false
}

// checkStatusIcon returns the colored status icon for a check run.
func checkStatusIcon(ch domain.CheckRun) string {
	if ch.Status != "completed" {
		if ch.Status == "in_progress" {
			return styles.CheckRunning.Render("⟳")
		}
		return styles.CheckPending.Render("◌")
	}
	switch ch.Conclusion {
	case "success", "neutral", "skipped":
		return styles.CheckPass.Render("✓")
	default:
		return styles.CheckFail.Render("✗")
	}
}

// renderCheckStatusText returns the colored status text for a check run.
func renderCheckStatusText(ch domain.CheckRun) string {
	if ch.Status != "completed" {
		if ch.Status == "in_progress" {
			return styles.CheckRunning.Render("running")
		}
		return styles.CheckPending.Render(ch.Status)
	}
	switch ch.Conclusion {
	case "success":
		return styles.CheckPass.Render("passed")
	case "failure":
		return styles.CheckFail.Render("failed")
	case "cancelled":
		return styles.CheckFail.Render("cancelled")
	case "skipped":
		return styles.CheckPass.Render("skipped")
	case "neutral":
		return styles.CheckPass.Render("neutral")
	default:
		return styles.StatusBarDim.Render(ch.Conclusion)
	}
}

// formatCheckDuration formats the elapsed time of a check run.
func formatCheckDuration(ch domain.CheckRun) string {
	if ch.CompletedAt.IsZero() || ch.StartedAt.IsZero() {
		if ch.Status == "in_progress" {
			return "running..."
		}
		return "—"
	}
	d := ch.CompletedAt.Sub(ch.StartedAt)
	if d < time.Minute {
		return fmt.Sprintf("%ds", max(int(d.Seconds()), 0))
	}
	m := int(d.Minutes())
	s := int(d.Seconds()) % 60
	if s == 0 {
		return fmt.Sprintf("%dm", m)
	}
	return fmt.Sprintf("%dm %ds", m, s)
}

// openInBrowser opens a URL in the system browser without suspending the TUI.
func openInBrowser(url string) tea.Cmd {
	return func() tea.Msg {
		//nolint:gosec // URL comes from GitHub API, not user input
		_ = exec.CommandContext(context.Background(), "open", url).Start()
		return nil
	}
}

// ── Key bindings ────────────────────────────────────────────────

type checksKeyBindings struct {
	Up      key.Binding
	Down    key.Binding
	Enter   key.Binding
	ViewLog key.Binding
	Open    key.Binding
}

var checksKeys = checksKeyBindings{
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
		key.WithHelp("enter", "view logs"),
	),
	ViewLog: key.NewBinding(
		key.WithKeys("l"),
		key.WithHelp("l", "view full log"),
	),
	Open: key.NewBinding(
		key.WithKeys("o"),
		key.WithHelp("o", "open in browser"),
	),
}

type logViewKeyBindings struct {
	ScrollDown key.Binding
	ScrollUp   key.Binding
	Open       key.Binding
}

var logViewKeys = logViewKeyBindings{
	ScrollDown: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("j", "scroll down"),
	),
	ScrollUp: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("k", "scroll up"),
	),
	Open: key.NewBinding(
		key.WithKeys("o"),
		key.WithHelp("o", "open in browser"),
	),
}
