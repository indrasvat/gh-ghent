package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/indrasvat/gh-ghent/internal/domain"
	ghub "github.com/indrasvat/gh-ghent/internal/github"
	"github.com/indrasvat/gh-ghent/internal/tui/styles"
)

// yellowStyle is defined here because greenStyle/redStyle/dimStyle
// are in resolve.go (same package), and watcher needs yellow for the
// review-await phase.
var yellowStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(string(styles.Yellow)))

// ── Watch state ──────────────────────────────────────────────────

type watchState int

const (
	watchStatePolling        watchState = iota // Actively polling CI checks
	watchStateAwaitingReview                   // CI passed, waiting for review activity to settle
	watchStateDone                             // Terminal state reached
	watchStateFailed                           // Fail-fast triggered
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

// ReviewPollFunc fetches a lightweight activity snapshot for review settlement.
type ReviewPollFunc func() (*domain.ActivitySnapshot, error)

// reviewPollResultMsg carries the result of a review activity probe.
type reviewPollResultMsg struct {
	snapshot *domain.ActivitySnapshot
	err      error
}

// reviewTickMsg triggers a review poll cycle.
type reviewTickMsg time.Time

// watchDoneMsg signals the watcher has reached a terminal state.
// The App listens for this to transition to ViewStatus.
type watchDoneMsg struct {
	settlement *domain.ReviewSettlement // nil if no review-await was active
}

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

	// Review-await mode.
	awaitReview          bool
	reviewTimeout        time.Duration
	reviewFetchFn        ReviewPollFunc
	reviewStartAt        time.Time
	reviewDeadline       time.Time
	lastActivityAt       time.Time
	prevHash             string
	baselineHash         string // fingerprint from before CI started
	activityCount        int
	initialHeadSHA       string
	reviewTailIntervals  []time.Duration
	reviewTailIndex      int
	reviewTailProbes     int
	reviewTailRearmed    bool
	reviewLateGrace      time.Duration
	reviewLateExtensions int

	// Status transition: when true, emit watchDoneMsg on terminal state.
	statusTransition bool
}

func newWatcherModel(interval time.Duration) watcherModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color(string(styles.Blue)))
	return watcherModel{
		state:               watchStatePolling,
		spinner:             s,
		startAt:             time.Now(),
		interval:            interval,
		seen:                make(map[int64]string),
		reviewTailIntervals: append([]time.Duration(nil), ghub.DefaultReviewWatchConfig().TailIntervals...),
		reviewTailIndex:     -1,
		reviewLateGrace:     ghub.DefaultReviewWatchConfig().LateActivityGrace,
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

	case reviewTickMsg:
		if m.state != watchStateAwaitingReview {
			return m, nil
		}
		return m, m.reviewPollCmd()

	case reviewPollResultMsg:
		return m.handleReviewPollResult(typedMsg)
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
		if m.awaitReview && m.reviewFetchFn != nil {
			// CI passed — transition to review-await phase.
			m.state = watchStateAwaitingReview
			m.reviewStartAt = time.Now()
			m.reviewDeadline = m.reviewStartAt.Add(m.reviewTimeout)
			m.lastActivityAt = time.Now()
			m.initialHeadSHA = msg.checks.HeadSHA
			m.reviewTailIndex = -1
			m.reviewTailProbes = 0
			m.reviewTailRearmed = false
			m.reviewLateExtensions = 0

			// Take initial probe and compare against baseline from before CI.
			// If different, review activity happened during CI — arm debounce.
			if snap, err := m.reviewFetchFn(); err == nil {
				m.prevHash = ghub.Fingerprint(snap)
				if m.baselineHash != "" && m.prevHash != m.baselineHash {
					m.activityCount++
					m.events = append(m.events, watchEvent{
						timestamp: time.Now(),
						icon:      yellowStyle.Render("●"),
						name:      "Review activity detected during CI",
					})
				} else if snap.ThreadCount > 0 {
					m.activityCount++
					m.events = append(m.events, watchEvent{
						timestamp: time.Now(),
						icon:      yellowStyle.Render("●"),
						name:      "Existing review threads detected",
					})
				}
			}

			m.events = append(m.events, watchEvent{
				timestamp: time.Now(),
				icon:      yellowStyle.Render("◎"),
				name:      "CI passed — awaiting reviews",
			})
			return m, m.scheduleNextReviewPoll()
		}
		m.state = watchStateDone
		m.events = append(m.events, watchEvent{
			timestamp: time.Now(),
			icon:      greenStyle.Render("✓"),
			name:      "All checks passed",
		})
		if m.statusTransition {
			return m, func() tea.Msg { return watchDoneMsg{} }
		}
		return m, nil
	case domain.StatusFail:
		m.state = watchStateFailed
		m.events = append(m.events, watchEvent{
			timestamp: time.Now(),
			icon:      redStyle.Render("✗"),
			name:      "Check failure detected",
			detail:    "fail-fast triggered",
		})
		if m.statusTransition {
			return m, func() tea.Msg { return watchDoneMsg{} }
		}
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

// ── Review-await poll methods ───────────────────────────────────

func (m watcherModel) reviewPollCmd() tea.Cmd {
	fn := m.reviewFetchFn
	if fn == nil {
		return nil
	}
	return func() tea.Msg {
		snap, err := fn()
		return reviewPollResultMsg{snapshot: snap, err: err}
	}
}

func (m watcherModel) scheduleNextReviewPoll() tea.Cmd {
	interval := 15 * time.Second
	if m.reviewTailIndex >= 0 && m.reviewTailIndex < len(m.reviewTailIntervals) {
		interval = m.reviewTailIntervals[m.reviewTailIndex]
	}
	if m.reviewTimeout > 0 && m.reviewTimeout < interval {
		interval = m.reviewTimeout
	}
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		return reviewTickMsg(t)
	})
}

func (m watcherModel) handleReviewPollResult(msg reviewPollResultMsg) (watcherModel, tea.Cmd) {
	now := time.Now()
	if m.reviewDeadline.IsZero() && m.reviewTimeout > 0 && !m.reviewStartAt.IsZero() {
		m.reviewDeadline = m.reviewStartAt.Add(m.reviewTimeout)
	}

	if msg.err != nil {
		m.events = append(m.events, watchEvent{
			timestamp: now,
			icon:      redStyle.Render("✗"),
			name:      "review poll error",
			detail:    msg.err.Error(),
		})
		m.logOffset = max(len(m.events)-m.eventLogHeight(), 0)

		// Hard timeout still applies during errors.
		if !m.reviewDeadline.IsZero() && !now.Before(m.reviewDeadline) {
			return m.finishReviewWait(domain.ReviewPhaseTimeout, now)
		}
		return m, m.scheduleNextReviewPoll()
	}

	// Check for head SHA change (new push).
	if msg.snapshot.HeadSHA != m.initialHeadSHA {
		m.state = watchStatePolling
		m.events = append(m.events, watchEvent{
			timestamp: now,
			icon:      yellowStyle.Render("↻"),
			name:      "New push detected — restarting CI watch",
		})
		m.logOffset = max(len(m.events)-m.eventLogHeight(), 0)
		// Reset CI watch state for new head.
		m.seen = make(map[int64]string)
		m.completed = 0
		m.total = 0
		return m, m.pollCmd()
	}

	// Compare fingerprints.
	newHash := ghub.Fingerprint(msg.snapshot)
	sawActivity := false
	if newHash != m.prevHash {
		m.lastActivityAt = now
		m.activityCount++
		m.prevHash = newHash
		sawActivity = true
		if m.reviewTailIndex >= 0 {
			m.reviewTailIndex = -1
			m.reviewTailRearmed = true
			m.events = append(m.events, watchEvent{
				timestamp: now,
				icon:      yellowStyle.Render("↺"),
				name:      "New review activity detected — resuming active watch",
			})
		} else {
			m.events = append(m.events, watchEvent{
				timestamp: now,
				icon:      yellowStyle.Render("●"),
				name:      "New review activity detected",
			})
		}
		if !m.reviewDeadline.IsZero() && m.reviewDeadline.Sub(now) <= 30*time.Second && m.reviewLateExtensions == 0 {
			m.reviewDeadline = m.reviewDeadline.Add(max(m.reviewLateGrace, 30*time.Second))
			m.reviewLateExtensions++
			m.events = append(m.events, watchEvent{
				timestamp: now,
				icon:      yellowStyle.Render("⏱"),
				name:      "Late review activity grace applied",
				detail:    formatDuration(max(m.reviewLateGrace, 30*time.Second)),
			})
		}
	}

	m.logOffset = max(len(m.events)-m.eventLogHeight(), 0)

	// Check debounce: settled when idle for 30s after at least one activity.
	// Don't settle on zero activity — the reviewer may still be working
	// (e.g., Codex shows 👀 for 2-4 min before posting comments).
	idleDuration := now.Sub(m.lastActivityAt)
	if m.reviewTailIndex >= 0 && !sawActivity {
		m.reviewTailProbes++
		if m.reviewTailIndex == len(m.reviewTailIntervals)-1 {
			return m.finishReviewWait(domain.ReviewPhaseSettled, now)
		}
		m.reviewTailIndex++
		return m, m.scheduleNextReviewPoll()
	}
	if m.activityCount > 0 && idleDuration >= 30*time.Second {
		if len(m.reviewTailIntervals) == 0 {
			return m.finishReviewWait(domain.ReviewPhaseSettled, now)
		}
		m.reviewTailIndex = 0
		m.events = append(m.events, watchEvent{
			timestamp: now,
			icon:      yellowStyle.Render("◎"),
			name:      "Review activity quiet — confirming stability",
		})
		m.logOffset = max(len(m.events)-m.eventLogHeight(), 0)
		return m, m.scheduleNextReviewPoll()
	}

	// Check hard timeout.
	if !m.reviewDeadline.IsZero() && !now.Before(m.reviewDeadline) {
		return m.finishReviewWait(domain.ReviewPhaseTimeout, now)
	}

	return m, m.scheduleNextReviewPoll()
}

func (m watcherModel) finishReviewWait(phase domain.ReviewWatchPhase, now time.Time) (watcherModel, tea.Cmd) {
	m.state = watchStateDone
	elapsed := now.Sub(m.reviewStartAt)

	icon := greenStyle.Render("✓")
	label := "Reviews settled"
	detail := formatDuration(elapsed)
	if phase == domain.ReviewPhaseTimeout {
		icon = yellowStyle.Render("⏱")
		label = "Review timeout reached"
		detail += " — more bot comments may still arrive"
	} else if m.reviewTailProbes > 0 {
		detail += fmt.Sprintf(" — tail probes %d", m.reviewTailProbes)
	}

	m.events = append(m.events, watchEvent{
		timestamp: now,
		icon:      icon,
		name:      label,
		detail:    detail,
	})
	m.logOffset = max(len(m.events)-m.eventLogHeight(), 0)

	settlement := &domain.ReviewSettlement{}
	*settlement = domain.NewReviewMonitor(
		phase,
		m.activityCount,
		int(elapsed.Seconds()),
		m.reviewTailProbes,
		m.reviewTailRearmed,
	)

	if m.statusTransition {
		return m, func() tea.Msg { return watchDoneMsg{settlement: settlement} }
	}
	return m, nil
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
			yellowStyle.Bold(true).Render("watching"))
	case watchStateAwaitingReview:
		label := "awaiting reviews"
		if m.reviewTailIndex >= 0 {
			label = "confirming review quiet"
		}
		parts = append(parts, " "+m.spinner.View()+" "+
			yellowStyle.Bold(true).Render(label))
	case watchStateDone:
		parts = append(parts, " "+greenStyle.Render("✓")+" "+
			greenStyle.Bold(true).Render("all checks passed"))
	case watchStateFailed:
		parts = append(parts, " "+redStyle.Render("✗")+" "+
			redStyle.Bold(true).Render("failure detected"))
	}

	if m.state == watchStateAwaitingReview {
		// Review-phase stats.
		idle := time.Since(m.lastActivityAt)
		parts = append(parts, dimStyle.Render(fmt.Sprintf("idle: %s", formatDuration(idle))))
		remaining := time.Until(m.reviewDeadline)
		if m.reviewDeadline.IsZero() {
			remaining = m.reviewTimeout - time.Since(m.reviewStartAt)
		}
		if remaining > 0 {
			parts = append(parts, dimStyle.Render(fmt.Sprintf("timeout: %s", formatDuration(remaining))))
		}
		if m.reviewTailProbes > 0 {
			parts = append(parts, dimStyle.Render(fmt.Sprintf("tail: %d", m.reviewTailProbes)))
		}
	} else {
		// CI-phase stats.
		if m.total > 0 {
			progress := fmt.Sprintf("%d/%d", m.completed, m.total)
			parts = append(parts, lipgloss.NewStyle().
				Foreground(lipgloss.Color(string(styles.Blue))).Render(progress))
		}

		elapsed := time.Since(m.startAt)
		parts = append(parts, dimStyle.Render("elapsed: "+formatDuration(elapsed)))

		parts = append(parts, dimStyle.Render(fmt.Sprintf("poll: %ds", int(m.interval.Seconds()))))
	}

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
