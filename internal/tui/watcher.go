package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/indrasvat/gh-ghent/internal/domain"
	"github.com/indrasvat/gh-ghent/internal/tui/styles"
)

// ── Watch state ──────────────────────────────────────────────────

type watchState int

const (
	watchStatePolling watchState = iota // Actively polling
	watchStateDone                      // Terminal state reached
	watchStateFailed                    // Fail-fast triggered
)

// ── Messages ─────────────────────────────────────────────────────

// watchTickMsg triggers a poll cycle.
type watchTickMsg time.Time

// watchResultMsg carries the result of a poll.
type watchResultMsg struct {
	checks *domain.ChecksResult
	err    error
}

// watchFetchFunc is called to poll check status.
type watchFetchFunc func() (*domain.ChecksResult, error)

// ── Watch event log entry ────────────────────────────────────────

type watchEvent struct {
	timestamp  time.Time
	icon       string
	name       string
	detail     string
	conclusion string
}

// ── Watcher model ────────────────────────────────────────────────

type watcherModel struct {
	state    watchState
	spinner  spinner.Model
	width    int
	height   int
	startAt  time.Time
	lastPoll time.Time
	interval time.Duration

	// Poll function — set by App from CLI.
	fetchFn watchFetchFunc

	// Current check data.
	checks    *domain.ChecksResult
	completed int
	total     int

	// Event log.
	events []watchEvent
	seen   map[int64]string // checkID → conclusion

	// Scroll offset for event log.
	logOffset int
}

func newWatcherModel(interval time.Duration) watcherModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color(string(styles.Blue)))
	return watcherModel{
		state:    watchStatePolling,
		spinner:  s,
		startAt:  time.Now(),
		interval: interval,
		seen:     make(map[int64]string),
	}
}

func (m *watcherModel) setSize(width, height int) {
	m.width = width
	m.height = height
}

// ── Init / Update ────────────────────────────────────────────────

// Init returns the initial commands: spinner tick + first poll.
func (m watcherModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.pollCmd())
}

// Update handles messages for the watcher model.
func (m watcherModel) Update(msg tea.Msg) (watcherModel, tea.Cmd) {
	switch typedMsg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(typedMsg)
		return m, cmd

	case watchTickMsg:
		if m.state != watchStatePolling {
			return m, nil
		}
		return m, m.pollCmd()

	case watchResultMsg:
		return m.handlePollResult(typedMsg)
	}
	return m, nil
}

func (m watcherModel) handlePollResult(msg watchResultMsg) (watcherModel, tea.Cmd) {
	if msg.err != nil {
		m.events = append(m.events, watchEvent{
			timestamp: time.Now(),
			icon:      redStyle.Render("✗"),
			name:      "poll error",
			detail:    msg.err.Error(),
		})
		return m, m.scheduleNextPoll()
	}

	m.checks = msg.checks
	m.lastPoll = time.Now()
	m.completed = 0
	m.total = len(msg.checks.Checks)

	// Process new events.
	for _, ch := range msg.checks.Checks {
		if ch.Status == "completed" {
			m.completed++
			if _, ok := m.seen[ch.ID]; !ok {
				m.seen[ch.ID] = ch.Conclusion
				m.events = append(m.events, m.makeEvent(ch))
			}
		}
	}

	// Auto-scroll event log to bottom.
	m.logOffset = max(len(m.events)-m.eventLogHeight(), 0)

	// Check terminal conditions.
	switch msg.checks.OverallStatus {
	case domain.StatusPass:
		m.state = watchStateDone
		m.events = append(m.events, watchEvent{
			timestamp: time.Now(),
			icon:      greenStyle.Render("✓"),
			name:      "All checks passed",
		})
		return m, nil
	case domain.StatusFail:
		m.state = watchStateFailed
		m.events = append(m.events, watchEvent{
			timestamp: time.Now(),
			icon:      redStyle.Render("✗"),
			name:      "Check failure detected",
			detail:    "fail-fast triggered",
		})
		return m, nil
	}

	return m, m.scheduleNextPoll()
}

func (m watcherModel) makeEvent(ch domain.CheckRun) watchEvent {
	icon := greenStyle.Render("✓")
	detail := ""
	switch ch.Conclusion {
	case "failure", "timed_out":
		icon = redStyle.Render("✗")
	case "skipped", "cancelled":
		icon = dimStyle.Render("—")
	}
	if !ch.CompletedAt.IsZero() && !ch.StartedAt.IsZero() {
		dur := ch.CompletedAt.Sub(ch.StartedAt)
		detail = formatDuration(dur)
	}
	return watchEvent{
		timestamp:  time.Now(),
		icon:       icon,
		name:       ch.Name,
		detail:     detail,
		conclusion: ch.Conclusion,
	}
}

func (m watcherModel) pollCmd() tea.Cmd {
	fn := m.fetchFn
	if fn == nil {
		return nil
	}
	return func() tea.Msg {
		result, err := fn()
		return watchResultMsg{checks: result, err: err}
	}
}

func (m watcherModel) scheduleNextPoll() tea.Cmd {
	interval := m.interval
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		return watchTickMsg(t)
	})
}

// ── View ─────────────────────────────────────────────────────────

func (m watcherModel) View() string {
	if m.width == 0 {
		return ""
	}

	var sections []string

	// ── Status line ──
	sections = append(sections, m.renderWatchStatus())
	sections = append(sections, "")

	// ── Check list (current state) ──
	sections = append(sections, m.renderCheckList())
	sections = append(sections, "")

	// ── Event log ──
	sections = append(sections, m.renderEventLog())

	content := strings.Join(sections, "\n")

	lineCount := strings.Count(content, "\n") + 1
	if lineCount < m.height {
		content += strings.Repeat("\n", m.height-lineCount)
	}

	return content
}

func (m watcherModel) renderWatchStatus() string {
	var parts []string

	switch m.state {
	case watchStatePolling:
		parts = append(parts, " "+m.spinner.View()+" "+
			lipgloss.NewStyle().Foreground(lipgloss.Color(string(styles.Yellow))).
				Bold(true).Render("watching"))
	case watchStateDone:
		parts = append(parts, " "+greenStyle.Render("✓")+" "+
			greenStyle.Bold(true).Render("all checks passed"))
	case watchStateFailed:
		parts = append(parts, " "+redStyle.Render("✗")+" "+
			redStyle.Bold(true).Render("failure detected"))
	}

	// Progress: completed/total.
	if m.total > 0 {
		progress := fmt.Sprintf("%d/%d", m.completed, m.total)
		parts = append(parts, lipgloss.NewStyle().
			Foreground(lipgloss.Color(string(styles.Blue))).Render(progress))
	}

	// Elapsed time.
	elapsed := time.Since(m.startAt)
	parts = append(parts, dimStyle.Render("elapsed: "+formatDuration(elapsed)))

	// Poll interval.
	parts = append(parts, dimStyle.Render(fmt.Sprintf("poll: %ds", int(m.interval.Seconds()))))

	return strings.Join(parts, "  ")
}

func (m watcherModel) renderCheckList() string {
	if m.checks == nil || len(m.checks.Checks) == 0 {
		return dimStyle.Render("  Waiting for first poll...")
	}

	var lines []string
	for _, ch := range m.checks.Checks {
		icon := checkStatusIcon(ch)
		name := ch.Name
		var status string

		nameStyle := lipgloss.NewStyle()
		statusStyle := dimStyle

		switch {
		case ch.Status == "completed" && ch.Conclusion == "success":
			statusStyle = greenStyle
			status = "passed"
		case ch.Status == "completed" && (ch.Conclusion == "failure" || ch.Conclusion == "timed_out"):
			nameStyle = nameStyle.Foreground(lipgloss.Color(string(styles.Red)))
			statusStyle = redStyle
			status = ch.Conclusion
		case ch.Status == "in_progress":
			nameStyle = nameStyle.Foreground(lipgloss.Color(string(styles.Blue)))
			statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(string(styles.Blue)))
			status = "running..."
		case ch.Status == "queued":
			nameStyle = nameStyle.Foreground(lipgloss.Color(string(styles.Dim)))
			status = "queued"
		default:
			status = ch.Status
		}

		dur := ""
		if ch.Status == "completed" && !ch.CompletedAt.IsZero() && !ch.StartedAt.IsZero() {
			dur = dimStyle.Render(formatDuration(ch.CompletedAt.Sub(ch.StartedAt)))
		}

		left := "  " + icon + " " + nameStyle.Render(name)
		right := dur
		if right != "" {
			right += "  "
		}
		right += statusStyle.Render(status)

		lines = append(lines, padWithRight(left, right, m.width))
	}
	return strings.Join(lines, "\n")
}

func (m watcherModel) renderEventLog() string {
	header := " " + lipgloss.NewStyle().
		Foreground(lipgloss.Color(string(styles.Blue))).Bold(true).
		Render("Event Log")

	if !m.lastPoll.IsZero() {
		ago := time.Since(m.lastPoll)
		header = padWithRight(header,
			dimStyle.Render(fmt.Sprintf("last updated %s ago", formatDuration(ago))),
			m.width)
	}

	if len(m.events) == 0 {
		return header + "\n" + dimStyle.Render("  Waiting for events...")
	}

	var lines []string
	lines = append(lines, header)

	logH := m.eventLogHeight()
	start := m.logOffset
	end := min(start+logH, len(m.events))

	for _, e := range m.events[start:end] {
		ts := dimStyle.Render(e.timestamp.Format("15:04:05"))
		line := "  " + ts + " " + e.icon + " " + e.name
		if e.detail != "" {
			line += " " + dimStyle.Render(e.detail)
		}
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

func (m watcherModel) eventLogHeight() int {
	// Reserve lines for status + check list + header + padding.
	checkLines := 0
	if m.checks != nil {
		checkLines = len(m.checks.Checks)
	}
	overhead := 2 + checkLines + 3 // status + gap + checks + gap + header
	h := m.height - overhead
	return max(h, 3)
}

// ── Helpers ──────────────────────────────────────────────────────

// formatDuration renders a duration as "Xs", "Xm Ys", etc.
func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	m := int(d.Minutes())
	s := int(d.Seconds()) % 60
	if s == 0 {
		return fmt.Sprintf("%dm", m)
	}
	return fmt.Sprintf("%dm %ds", m, s)
}
