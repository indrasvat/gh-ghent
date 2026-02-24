package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/indrasvat/gh-ghent/internal/domain"
	"github.com/indrasvat/gh-ghent/internal/tui/styles"
)

// summaryModel renders the dashboard overview with KPI cards, section
// previews, and merge readiness badge.
type summaryModel struct {
	comments     *domain.CommentsResult
	checks       *domain.ChecksResult
	reviews      []domain.Review
	width        int
	height       int
	scrollOffset int
	loading      bool // true while async data is being fetched
}

func (m *summaryModel) setSize(width, height int) {
	m.width = width
	m.height = height
}

// scrollDown moves the viewport down by one line.
func (m *summaryModel) scrollDown() {
	m.scrollOffset++
}

// scrollUp moves the viewport up by one line.
func (m *summaryModel) scrollUp() {
	if m.scrollOffset > 0 {
		m.scrollOffset--
	}
}

// isMergeReady mirrors the CLI's IsMergeReady logic.
func (m summaryModel) isMergeReady() bool {
	if m.comments != nil && m.comments.UnresolvedCount > 0 {
		return false
	}
	if m.checks != nil && m.checks.OverallStatus != domain.StatusPass {
		return false
	}
	if m.reviews != nil {
		hasApproval := false
		for _, r := range m.reviews {
			if r.State == domain.ReviewApproved {
				hasApproval = true
			}
			if r.State == domain.ReviewChangesRequested {
				return false
			}
		}
		if !hasApproval {
			return false
		}
	}
	return true
}

// View renders the summary dashboard.
func (m summaryModel) View() string {
	if m.width == 0 {
		return ""
	}

	// Show loading state while data is being fetched.
	if m.loading && m.comments == nil && m.checks == nil && m.reviews == nil {
		return m.renderLoadingView()
	}

	var sections []string

	// ── KPI cards row ────────────────────────────────────
	sections = append(sections, m.renderKPICards())
	sections = append(sections, "")

	// ── Review Threads section ───────────────────────────
	sections = append(sections, m.renderThreadsSection())
	sections = append(sections, "")

	// ── CI Checks section ────────────────────────────────
	sections = append(sections, m.renderChecksSection())
	sections = append(sections, "")

	// ── Approvals section ────────────────────────────────
	sections = append(sections, m.renderApprovalsSection())

	content := strings.Join(sections, "\n")
	allLines := strings.Split(content, "\n")
	totalLines := len(allLines)

	// Clamp scroll offset to valid range.
	maxScroll := max(totalLines-m.height, 0)
	if m.scrollOffset > maxScroll {
		m.scrollOffset = maxScroll
	}

	// Apply scroll: slice to visible window.
	startLine := m.scrollOffset
	endLine := min(startLine+m.height, totalLines)
	visibleLines := allLines[startLine:endLine]

	// Pad to fill content height.
	if len(visibleLines) < m.height {
		for range m.height - len(visibleLines) {
			visibleLines = append(visibleLines, "")
		}
	}

	return strings.Join(visibleLines, "\n")
}

// renderLoadingView shows a loading message while data is being fetched.
func (m summaryModel) renderLoadingView() string {
	loading := dimStyle.Render("  Loading PR data...")

	// Pad to fill content area height.
	content := loading
	lineCount := strings.Count(content, "\n") + 1
	if lineCount < m.height {
		content += strings.Repeat("\n", m.height-lineCount)
	}
	return content
}

// ── KPI Cards ────────────────────────────────────────────────────

func (m summaryModel) renderKPICards() string {
	var cards []string

	// Unresolved threads card.
	unresolvedCount := 0
	if m.comments != nil {
		unresolvedCount = m.comments.UnresolvedCount
	}
	cards = append(cards, m.renderCard(
		unresolvedCount, "Unresolved", cardColorForCount(unresolvedCount, true)))

	// Checks passed card.
	passCount := 0
	if m.checks != nil {
		passCount = m.checks.PassCount
	}
	cards = append(cards, m.renderCard(passCount, "Passed", lipgloss.Color(string(styles.Green))))

	// Checks failed card.
	failCount := 0
	if m.checks != nil {
		failCount = m.checks.FailCount
	}
	cards = append(cards, m.renderCard(
		failCount, "Failed", cardColorForCount(failCount, true)))

	// Approvals card.
	approvalCount := 0
	for _, r := range m.reviews {
		if r.State == domain.ReviewApproved {
			approvalCount++
		}
	}
	approvalColor := lipgloss.Color(string(styles.Yellow))
	if approvalCount > 0 {
		approvalColor = lipgloss.Color(string(styles.Green))
	}
	cards = append(cards, m.renderCard(approvalCount, "Approvals", approvalColor))

	// Layout: distribute cards across width.
	cardWidth := max((m.width-4*3)/4, 10) // 3 gaps between 4 cards

	var rendered []string
	for _, card := range cards {
		styled := lipgloss.NewStyle().
			Width(cardWidth).
			Align(lipgloss.Center).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(string(styles.Surface2))).
			Render(card)
		rendered = append(rendered, styled)
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, rendered...)
}

func (m summaryModel) renderCard(count int, label string, color lipgloss.Color) string {
	countStr := lipgloss.NewStyle().
		Foreground(color).
		Bold(true).
		Render(fmt.Sprintf("%d", count))
	labelStr := dimStyle.Render(strings.ToUpper(label))
	return countStr + "\n" + labelStr
}

// cardColorForCount returns green if count is 0, red otherwise.
func cardColorForCount(count int, redIfNonZero bool) lipgloss.Color {
	if redIfNonZero && count > 0 {
		return lipgloss.Color(string(styles.Red))
	}
	return lipgloss.Color(string(styles.Green))
}

// ── Section: Review Threads ──────────────────────────────────────

func (m summaryModel) renderThreadsSection() string {
	headerDot := redStyle.Render("●")
	title := "Review Threads"
	rightInfo := ""

	if m.comments != nil {
		var parts []string
		if m.comments.UnresolvedCount > 0 {
			parts = append(parts, fmt.Sprintf("%d unresolved", m.comments.UnresolvedCount))
		}
		if m.comments.ResolvedCount > 0 {
			parts = append(parts, fmt.Sprintf("%d resolved", m.comments.ResolvedCount))
		}
		rightInfo = dimStyle.Render(strings.Join(parts, " · "))

		if m.comments.UnresolvedCount == 0 {
			headerDot = greenStyle.Render("●")
		}
	}

	header := m.renderSectionHeader(headerDot, title, rightInfo)

	if m.comments == nil || len(m.comments.Threads) == 0 {
		return header + "\n" + dimStyle.Render("   No review threads")
	}

	var lines []string
	lines = append(lines, header)

	maxShow := 3
	threads := m.comments.Threads
	for i, t := range threads {
		if i >= maxShow {
			break
		}
		fileLine := styles.FilePath.Render(t.Path) +
			styles.LineNumber.Render(fmt.Sprintf(":%d", t.Line))
		author := ""
		timeAgo := ""
		if len(t.Comments) > 0 {
			author = styles.Author.Render("@" + t.Comments[0].Author)
			timeAgo = dimStyle.Render(formatTimeAgo(t.Comments[0].CreatedAt))
		}
		line := "   " + fileLine + " " + dimStyle.Render("—") + " " + author
		line = padWithRight(line, timeAgo, m.width-2)
		lines = append(lines, line)
	}

	if len(threads) > maxShow {
		more := fmt.Sprintf("   ... and %d more", len(threads)-maxShow)
		lines = append(lines, dimStyle.Render(more))
	}

	return strings.Join(lines, "\n")
}

// ── Section: CI Checks ───────────────────────────────────────────

func (m summaryModel) renderChecksSection() string {
	headerDot := greenStyle.Render("●")
	title := "CI Checks"
	rightInfo := ""

	if m.checks != nil {
		var parts []string
		if m.checks.PassCount > 0 {
			parts = append(parts, fmt.Sprintf("%d passed", m.checks.PassCount))
		}
		if m.checks.FailCount > 0 {
			parts = append(parts, fmt.Sprintf("%d failed", m.checks.FailCount))
		}
		rightInfo = dimStyle.Render(strings.Join(parts, " · "))

		if m.checks.FailCount > 0 {
			headerDot = redStyle.Render("●")
		} else if m.checks.PendingCount > 0 {
			headerDot = lipgloss.NewStyle().
				Foreground(lipgloss.Color(string(styles.Yellow))).Render("●")
		}
	}

	header := m.renderSectionHeader(headerDot, title, rightInfo)

	if m.checks == nil || len(m.checks.Checks) == 0 {
		return header + "\n" + dimStyle.Render("   No CI checks")
	}

	var lines []string
	lines = append(lines, header)

	// Show failed checks first with annotations.
	for _, c := range m.checks.Checks {
		if !checkIsFailed(c) {
			continue
		}
		icon := redStyle.Render("✗")
		name := redStyle.Render(c.Name)
		annotCount := ""
		if len(c.Annotations) > 0 {
			annotCount = dimStyle.Render(fmt.Sprintf("%d errors", len(c.Annotations)))
		}
		line := "   " + icon + " " + name
		if annotCount != "" {
			line = padWithRight(line, annotCount, m.width-2)
		}
		lines = append(lines, line)

		// Show up to 3 annotations.
		for j, a := range c.Annotations {
			if j >= 3 {
				break
			}
			aFile := dimStyle.Render(a.Path)
			aLine := dimStyle.Render(fmt.Sprintf(":%d", a.StartLine))
			aTitle := ""
			if a.Title != "" {
				aTitle = lipgloss.NewStyle().
					Foreground(lipgloss.Color(string(styles.Yellow))).
					Render(fmt.Sprintf(" [%s]", a.Title))
			}
			lines = append(lines, "      "+aFile+aLine+aTitle)
		}
	}

	// Show passed count summary.
	if m.checks.PassCount > 0 {
		icon := greenStyle.Render("✓")
		passNames := checkNames(m.checks.Checks, false)
		summary := greenStyle.Render(fmt.Sprintf("%d checks passed", m.checks.PassCount))
		if passNames != "" {
			summary += " " + dimStyle.Render("("+passNames+")")
		}
		lines = append(lines, "   "+icon+" "+summary)
	}

	return strings.Join(lines, "\n")
}

// checkNames returns a comma-separated list of check names filtered by failed status.
func checkNames(checks []domain.CheckRun, failed bool) string {
	var names []string
	for _, c := range checks {
		if checkIsFailed(c) == failed {
			names = append(names, c.Name)
		}
	}
	if len(names) > 4 {
		return strings.Join(names[:3], ", ") + fmt.Sprintf(", +%d more", len(names)-3)
	}
	return strings.Join(names, ", ")
}

// ── Section: Approvals ───────────────────────────────────────────

// maxReviewsShow is the maximum number of reviews displayed in the summary.
const maxReviewsShow = 5

func (m summaryModel) renderApprovalsSection() string {
	headerDot := lipgloss.NewStyle().
		Foreground(lipgloss.Color(string(styles.Yellow))).Render("●")
	title := "Approvals"
	rightInfo := ""

	approvedCount := 0
	for _, r := range m.reviews {
		if r.State == domain.ReviewApproved {
			approvedCount++
		}
	}
	if approvedCount > 0 {
		headerDot = greenStyle.Render("●")
	}

	if len(m.reviews) > 0 {
		rightInfo = dimStyle.Render(fmt.Sprintf("%d reviews", len(m.reviews)))
	}

	header := m.renderSectionHeader(headerDot, title, rightInfo)

	if len(m.reviews) == 0 {
		return header + "\n" + dimStyle.Render("   No reviews yet")
	}

	// Sort by priority: CHANGES_REQUESTED > APPROVED > rest (most actionable first).
	sorted := make([]domain.Review, len(m.reviews))
	copy(sorted, m.reviews)
	sort.SliceStable(sorted, func(i, j int) bool {
		return reviewPriority(sorted[i].State) < reviewPriority(sorted[j].State)
	})

	var lines []string
	lines = append(lines, header)

	for i, r := range sorted {
		if i >= maxReviewsShow {
			break
		}
		icon, stateText := reviewIcon(r.State)
		author := styles.Author.Render("@" + r.Author)
		timeAgo := dimStyle.Render(formatTimeAgo(r.SubmittedAt))
		line := "   " + icon + " " + author + " " + stateText
		line = padWithRight(line, timeAgo, m.width-2)
		lines = append(lines, line)
	}

	if len(sorted) > maxReviewsShow {
		more := fmt.Sprintf("   ... and %d more", len(sorted)-maxReviewsShow)
		lines = append(lines, dimStyle.Render(more))
	}

	return strings.Join(lines, "\n")
}

// reviewPriority returns a sort key — lower values sort first.
func reviewPriority(state domain.ReviewState) int {
	switch state {
	case domain.ReviewChangesRequested:
		return 0
	case domain.ReviewApproved:
		return 1
	case domain.ReviewCommented:
		return 2
	default:
		return 3
	}
}

// reviewIcon returns the icon and styled state text for a review.
func reviewIcon(state domain.ReviewState) (string, string) {
	switch state {
	case domain.ReviewApproved:
		return greenStyle.Render("✓"), greenStyle.Render("approved")
	case domain.ReviewChangesRequested:
		return lipgloss.NewStyle().
				Foreground(lipgloss.Color(string(styles.Yellow))).Render("✗"),
			lipgloss.NewStyle().
				Foreground(lipgloss.Color(string(styles.Yellow))).Render("changes requested")
	case domain.ReviewCommented:
		return dimStyle.Render("○"), dimStyle.Render("commented")
	case domain.ReviewDismissed:
		return dimStyle.Render("—"), dimStyle.Render("dismissed")
	default:
		return dimStyle.Render("◌"), dimStyle.Render("pending")
	}
}

// ── Section header helper ────────────────────────────────────────

func (m summaryModel) renderSectionHeader(dot, title, rightInfo string) string {
	left := " " + dot + " " + lipgloss.NewStyle().Bold(true).Render(title)
	if rightInfo == "" {
		return left
	}
	return padWithRight(left, rightInfo+" ", m.width)
}

// mergeReadyBadge returns the badge text and color for the status bar.
func (m summaryModel) mergeReadyBadge() (string, lipgloss.Color) {
	if m.isMergeReady() {
		return "READY", lipgloss.Color(string(styles.Green))
	}
	return "NOT READY", lipgloss.Color(string(styles.Red))
}
