// Package tui implements the ghent interactive Bubble Tea TUI.
//
// The root App model manages view state, routes key events to the active view,
// handles Tab switching between top-level views, and renders the shared
// status bar + help bar framing all views.
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

	// Key map
	keys AppKeyMap

	// Sub-models
	commentsList commentsListModel
}

// NewApp creates a new App model with the given repo, PR, and initial view.
func NewApp(repo string, pr int, initialView View) App {
	return App{
		activeView: initialView,
		repo:       repo,
		pr:         pr,
		keys:       DefaultKeyMap(),
	}
}

// Init implements tea.Model.
func (a App) Init() tea.Cmd {
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
		return a, nil

	case tea.KeyMsg:
		return a.handleKey(typedMsg)

	// Messages from sub-models
	case selectThreadMsg:
		a.activeView = ViewCommentsExpand
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
		// From resolve/summary, return to previous view.
		if a.activeView == ViewResolve || a.activeView == ViewSummary {
			a.activeView = a.prevView
			return a, nil
		}
		return a, nil
	}

	// Enter: drill into detail views from list views.
	// Comments list handles Enter via its sub-model (returns selectThreadMsg).
	// Checks list still uses direct handling until its sub-model is wired.
	if key.Matches(msg, a.keys.Enter) && a.activeView == ViewChecksList {
		a.activeView = ViewChecksLog
		return a, nil
	}

	// Summary-specific shortcuts: c/k/r jump to views.
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
	}

	// Forward remaining keys to the active sub-model.
	return a.forwardToActiveView(tea.Msg(msg))
}

// forwardToActiveView dispatches a message to the active sub-model.
func (a App) forwardToActiveView(msg tea.Msg) (tea.Model, tea.Cmd) {
	if a.activeView == ViewCommentsList {
		var cmd tea.Cmd
		a.commentsList, cmd = a.commentsList.Update(msg)
		return a, cmd
	}
	return a, nil
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
		data.RightBadge = "SUMMARY"
		data.BadgeColor = lipgloss.Color(string(styles.Blue))

	case ViewResolve:
		data.RightBadge = "RESOLVE"
		data.BadgeColor = lipgloss.Color(string(styles.Yellow))
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
		bindings = components.CommentsExpandedKeys() // reuse for log view
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
	// Comments list uses its sub-model; others use placeholders.
	if a.activeView == ViewCommentsList {
		return a.commentsList.View()
	}

	// Placeholder text for views not yet wired.
	var placeholder string
	switch a.activeView {
	case ViewCommentsExpand:
		placeholder = "  [Comment Thread Expanded — pending task 5.2]"
	case ViewChecksList:
		placeholder = "  [Checks List View — pending task 4.6]"
	case ViewChecksLog:
		placeholder = "  [Check Log View — pending task 4.6]"
	case ViewResolve:
		placeholder = "  [Resolve View — pending task 4.7]"
	case ViewSummary:
		placeholder = "  [Summary Dashboard — pending task 4.8]"
	case ViewWatch:
		placeholder = "  [Watch Mode — pending task 4.9]"
	default:
		placeholder = "  [Unknown View]"
	}

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
	}
}

// SetChecks updates the shared checks data.
func (a *App) SetChecks(c *domain.ChecksResult) {
	a.checks = c
}

// SetReviews updates the shared reviews data.
func (a *App) SetReviews(r []domain.Review) {
	a.reviews = r
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
