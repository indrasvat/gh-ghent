// Package tui implements the ghent interactive Bubble Tea TUI.
//
// The root App model manages view state, routes key events to the active view,
// handles Tab switching between top-level views, and renders the shared
// status bar + help bar framing all views.
package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/indrasvat/gh-ghent/internal/domain"
	"github.com/indrasvat/gh-ghent/internal/tui/components"
	"github.com/indrasvat/gh-ghent/internal/tui/styles"
)

// View represents the active TUI view.
type View int

const (
	ViewCommentsList   View = iota // Top-level comments list
	ViewCommentsExpand             // Expanded single thread
	ViewChecksList                 // Top-level checks list
	ViewChecksLog                  // Expanded single check log
	ViewResolve                    // Multi-select resolve
	ViewSummary                    // Summary dashboard
	ViewWatch                      // Watch mode (spinner + progress)
)

// String returns a human-readable name for the view.
func (v View) String() string {
	switch v {
	case ViewCommentsList:
		return "comments"
	case ViewCommentsExpand:
		return "comments-expand"
	case ViewChecksList:
		return "checks"
	case ViewChecksLog:
		return "checks-log"
	case ViewResolve:
		return "resolve"
	case ViewSummary:
		return "summary"
	case ViewWatch:
		return "watch"
	default:
		return "unknown"
	}
}

// isDetail returns true for drill-in views (Esc returns to parent).
func (v View) isDetail() bool {
	return v == ViewCommentsExpand || v == ViewChecksLog
}

// parentView returns the list view a detail view should return to.
func (v View) parentView() View {
	switch v {
	case ViewCommentsExpand:
		return ViewCommentsList
	case ViewChecksLog:
		return ViewChecksList
	default:
		return v
	}
}

// App is the root Bubble Tea model for the ghent TUI.
type App struct {
	// View state
	activeView View
	prevView   View // for Esc to return from summary/resolve

	// Terminal dimensions — propagated to ALL sub-models on WindowSizeMsg.
	width  int
	height int

	// Shared data
	repo     string // "owner/repo"
	pr       int
	comments *domain.CommentsResult
	checks   *domain.ChecksResult
	reviews  []domain.Review

	// Resolver callback for resolve view mutations.
	resolveFunc func(threadID string) error

	// Key map
	keys AppKeyMap

	// Sub-models
	commentsList     commentsListModel
	commentsExpanded commentsExpandedModel
	checksList       checksListModel
	checksLog        checksLogModel
	resolve          resolveModel
	summary          summaryModel
	watcher          watcherModel
}

// NewApp creates a new App model with the given repo, PR, and initial view.
func NewApp(repo string, pr int, initialView View) App {
	return App{
		activeView: initialView,
		prevView:   initialView, // Esc is no-op until user navigates away
		repo:       repo,
		pr:         pr,
		keys:       DefaultKeyMap(),
	}
}

// Init implements tea.Model.
func (a App) Init() tea.Cmd {
	if a.activeView == ViewWatch {
		return a.watcher.Init()
	}
	return nil
}

// Update implements tea.Model.
//
// CRITICAL: Uses `typedMsg := msg.(type)` pattern to avoid switch shadowing
// (pitfall #5 in testing-strategy.md).
func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// CRITICAL: Declare typedMsg outside the switch to avoid shadowing (pitfall #5).
	switch typedMsg := msg.(type) {
	case tea.WindowSizeMsg:
		// CRITICAL: Propagate to ALL sub-models, active AND inactive (pitfall #7).
		a.width = typedMsg.Width
		a.height = typedMsg.Height
		contentHeight := max(a.height-2, 1) // minus status bar + help bar
		a.commentsList.setSize(a.width, contentHeight)
		a.commentsExpanded.setSize(a.width, contentHeight)
		a.checksList.setSize(a.width, contentHeight)
		a.checksLog.setSize(a.width, contentHeight)
		a.resolve.setSize(a.width, contentHeight)
		a.summary.setSize(a.width, contentHeight)
		a.watcher.setSize(a.width, contentHeight)
		return a, nil

	case tea.KeyMsg:
		return a.handleKey(typedMsg)

	// Messages from sub-models
	case selectThreadMsg:
		a.activeView = ViewCommentsExpand
		if a.comments != nil {
			a.commentsExpanded = newCommentsExpandedModel(a.comments.Threads, typedMsg.threadIdx)
			contentHeight := max(a.height-2, 1)
			a.commentsExpanded.setSize(a.width, contentHeight)
		}
		return a, nil

	case resolveRequestMsg:
		// Execute resolve mutations via the resolver callback.
		if a.resolveFunc != nil {
			var cmds []tea.Cmd
			for _, id := range typedMsg.threadIDs {
				threadID := id // capture
				cmds = append(cmds, func() tea.Msg {
					err := a.resolveFunc(threadID)
					return resolveThreadMsg{threadID: threadID, err: err}
				})
			}
			return a, tea.Batch(cmds...)
		}
		return a, nil

	case selectCheckMsg:
		a.activeView = ViewChecksLog
		if a.checks != nil && typedMsg.checkIdx >= 0 && typedMsg.checkIdx < len(a.checks.Checks) {
			a.checksLog = newChecksLogModel(&a.checks.Checks[typedMsg.checkIdx])
			contentHeight := max(a.height-2, 1)
			a.checksLog.setSize(a.width, contentHeight)
		}
		return a, nil
	}

	// Forward other messages to the active view's sub-model.
	return a.forwardToActiveView(msg)
}

// handleKey processes key events with routing based on active view.
func (a App) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Quit from any view.
	if key.Matches(msg, a.keys.Quit) {
		return a, tea.Quit
	}

	// Tab/Shift+Tab: cycle top-level views (comments ↔ checks).
	if key.Matches(msg, a.keys.Tab) {
		a = a.cycleView(1)
		return a, nil
	}
	if key.Matches(msg, a.keys.ShiftTab) {
		a = a.cycleView(-1)
		return a, nil
	}

	// Esc: back to parent list from detail views.
	if key.Matches(msg, a.keys.Esc) {
		if a.activeView.isDetail() {
			a.activeView = a.activeView.parentView()
			return a, nil
		}
		// In resolve confirming state, forward Esc to cancel the confirmation dialog.
		if a.activeView == ViewResolve && a.resolve.state == resolveStateConfirming {
			return a.forwardToActiveView(tea.Msg(msg))
		}
		// Return to previous view if set (covers summary→comments, summary→checks, etc.).
		if a.prevView != a.activeView {
			a.activeView = a.prevView
			return a, nil
		}
		return a, nil
	}

	// Summary-specific shortcuts: c/k/r jump to views, j/↑/↓ scroll.
	if a.activeView == ViewSummary {
		switch {
		case key.Matches(msg, a.keys.Comments):
			a.prevView = ViewSummary
			a.activeView = ViewCommentsList
			return a, nil
		case key.Matches(msg, a.keys.Checks):
			a.prevView = ViewSummary
			a.activeView = ViewChecksList
			return a, nil
		case key.Matches(msg, a.keys.Resolve):
			a.prevView = ViewSummary
			a.activeView = ViewResolve
			return a, nil
		}
		// Scroll: j/↓ = down, ↑ = up (k is taken by checks shortcut).
		switch msg.String() {
		case "j", "down":
			a.summary.scrollDown()
			return a, nil
		case "up":
			a.summary.scrollUp()
			return a, nil
		}
	}

	// Forward remaining keys to the active sub-model.
	return a.forwardToActiveView(tea.Msg(msg))
}

// forwardToActiveView dispatches a message to the active sub-model.
func (a App) forwardToActiveView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch a.activeView {
	case ViewCommentsList:
		a.commentsList, cmd = a.commentsList.Update(msg)
	case ViewCommentsExpand:
		a.commentsExpanded, cmd = a.commentsExpanded.Update(msg)
	case ViewChecksList:
		a.checksList, cmd = a.checksList.Update(msg)
	case ViewChecksLog:
		a.checksLog, cmd = a.checksLog.Update(msg)
	case ViewResolve:
		a.resolve, cmd = a.resolve.Update(msg)
	case ViewWatch:
		a.watcher, cmd = a.watcher.Update(msg)
	}
	return a, cmd
}

// cycleView switches between top-level views (comments ↔ checks).
func (a App) cycleView(direction int) App {
	topLevelViews := []View{ViewCommentsList, ViewChecksList}
	current := 0
	for i, v := range topLevelViews {
		if a.activeView == v || a.activeView.parentView() == v {
			current = i
			break
		}
	}
	next := (current + direction + len(topLevelViews)) % len(topLevelViews)
	a.activeView = topLevelViews[next]
	return a
}

// View implements tea.Model.
// Renders: status bar (top) + active view (middle) + help bar (bottom).
func (a App) View() string {
	if a.width == 0 || a.height == 0 {
		return ""
	}

	// Status bar (top)
	statusBar := a.renderStatusBar()

	// Help bar (bottom)
	helpBar := a.renderHelpBar()

	// Content area: total height minus status bar (1 line) and help bar (1 line)
	contentHeight := max(a.height-2, 1)

	// Active view content (placeholder until sub-models are wired)
	content := a.renderActiveView(contentHeight)

	return statusBar + "\n" + content + "\n" + helpBar
}

// renderStatusBar builds the top status bar based on active view.
func (a App) renderStatusBar() string {
	data := components.StatusBarData{
		Repo: a.repo,
		PR:   a.pr,
		View: a.activeView.String(),
	}

	switch a.activeView {
	case ViewCommentsList, ViewCommentsExpand:
		if a.comments != nil {
			right := ""
			if a.comments.UnresolvedCount > 0 {
				right += styles.BadgeRed.Render(
					formatCount(a.comments.UnresolvedCount, "unresolved"))
			}
			if a.comments.ResolvedCount > 0 {
				if right != "" {
					right += "  "
				}
				right += styles.StatusBarDim.Render(
					formatCount(a.comments.ResolvedCount, "resolved"))
			}
			// Show "Thread X of Y" in expanded view.
			if a.activeView == ViewCommentsExpand && a.commentsExpanded.ThreadCount() > 0 {
				if right != "" {
					right += "  "
				}
				right += styles.StatusBarDim.Render(
					fmt.Sprintf("Thread %d of %d",
						a.commentsExpanded.ThreadIndex()+1,
						a.commentsExpanded.ThreadCount()))
			}
			data.Right = right
		}

	case ViewChecksList, ViewChecksLog, ViewWatch:
		if a.checks != nil {
			data.Left = styles.StatusBarDim.Render("HEAD: " + truncateSHA(a.checks.HeadSHA))
			right := ""
			if a.checks.PassCount > 0 {
				right += styles.BadgeGreen.Render(
					formatCount(a.checks.PassCount, "passed"))
			}
			if a.checks.FailCount > 0 {
				if right != "" {
					right += "  "
				}
				right += styles.BadgeRed.Render(
					formatCount(a.checks.FailCount, "failed"))
			}
			data.Right = right
		}

	case ViewSummary:
		badge, badgeColor := a.summary.mergeReadyBadge()
		data.RightBadge = badge
		data.BadgeColor = badgeColor

	case ViewResolve:
		if a.comments != nil {
			right := ""
			sel := a.resolve.selectedCount()
			if sel > 0 {
				right += lipgloss.NewStyle().Foreground(lipgloss.Color(string(styles.Green))).
					Render(fmt.Sprintf("%d selected", sel))
				right += "  "
			}
			right += styles.StatusBarDim.Render(
				fmt.Sprintf("of %d unresolved", a.comments.UnresolvedCount))
			data.Left = styles.StatusBarDim.Render("resolve mode")
			data.Right = right
		} else {
			data.RightBadge = "RESOLVE"
			data.BadgeColor = lipgloss.Color(string(styles.Yellow))
		}
	}

	return components.RenderStatusBar(data, a.width)
}

// renderHelpBar builds the bottom help bar with context-sensitive key bindings.
func (a App) renderHelpBar() string {
	var bindings []components.KeyBinding
	switch a.activeView {
	case ViewCommentsList:
		bindings = components.CommentsListKeys()
	case ViewCommentsExpand:
		bindings = components.CommentsExpandedKeys()
	case ViewChecksList:
		bindings = components.ChecksListKeys()
	case ViewChecksLog:
		bindings = components.ChecksLogKeys()
	case ViewWatch:
		bindings = components.ChecksWatchKeys()
	case ViewResolve:
		bindings = components.ResolveKeys()
	case ViewSummary:
		bindings = components.SummaryKeys()
	}
	return components.RenderHelpBar(bindings, a.width)
}

// renderActiveView renders the content area for the current view.
// Returns a string sized to fill contentHeight lines.
func (a App) renderActiveView(contentHeight int) string {
	// Sub-models with real views.
	switch a.activeView {
	case ViewCommentsList:
		return a.commentsList.View()
	case ViewCommentsExpand:
		return a.commentsExpanded.View()
	case ViewChecksList:
		return a.checksList.View()
	case ViewChecksLog:
		return a.checksLog.View()
	case ViewResolve:
		return a.resolve.View()
	case ViewSummary:
		return a.summary.View()
	case ViewWatch:
		return a.watcher.View()
	}

	// Placeholder text for views not yet wired.
	placeholder := "  [Unknown View]"

	content := styles.StatusBarDim.Render(placeholder)

	// Pad to fill content area height.
	if contentHeight > 1 {
		content += strings.Repeat("\n", contentHeight-1)
	}
	return content
}

// SetComments updates the shared comments data and rebuilds the comments list.
func (a *App) SetComments(c *domain.CommentsResult) {
	a.comments = c
	if c != nil {
		a.commentsList = newCommentsListModel(c.Threads)
		a.resolve = newResolveModel(c.Threads)
	}
	a.summary.comments = c
}

// SetChecks updates the shared checks data and rebuilds the checks list.
func (a *App) SetChecks(c *domain.ChecksResult) {
	a.checks = c
	if c != nil {
		a.checksList = newChecksListModel(c.Checks)
	}
	a.summary.checks = c
}

// SetResolver sets the callback function for resolving threads.
func (a *App) SetResolver(fn func(threadID string) error) {
	a.resolveFunc = fn
}

// SetWatchFetch configures the watcher model with a fetch function and interval.
func (a *App) SetWatchFetch(fn watchFetchFunc, interval time.Duration) {
	a.watcher = newWatcherModel(interval)
	a.watcher.fetchFn = fn
}

// SetReviews updates the shared reviews data.
func (a *App) SetReviews(r []domain.Review) {
	a.reviews = r
	a.summary.reviews = r
}

// ActiveView returns the current active view.
func (a App) ActiveView() View {
	return a.activeView
}

// Width returns the current terminal width.
func (a App) Width() int {
	return a.width
}

// Height returns the current terminal height.
func (a App) Height() int {
	return a.height
}

// formatCount returns "N label" (e.g. "5 unresolved").
func formatCount(n int, label string) string {
	return fmt.Sprintf("%d %s", n, label)
}

// truncateSHA returns the first 7 chars of a SHA.
func truncateSHA(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}
