// Package components provides reusable TUI building blocks shared across
// all ghent views: status bar, help bar, and diff hunk renderer.
package components

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"

	"github.com/indrasvat/gh-ghent/internal/tui/styles"
)

// StatusBarData holds the data rendered in the top status bar.
type StatusBarData struct {
	Repo       string // "owner/repo"
	PR         int    // PR number
	View       string // Current view name (e.g., "comments", "checks")
	PRTitle    string // PR title (optional, shown in summary)
	Left       string // Additional left text
	Right      string // Additional right text (e.g., "5 unresolved · 2 resolved")
	RightBadge string // Badge text for right side (e.g., "NOT READY")
	BadgeColor lipgloss.Color
}

// RenderStatusBar renders the top status bar with left-aligned repo+PR and
// right-aligned counts/badges. Adapts to the given terminal width.
func RenderStatusBar(data StatusBarData, width int) string {
	if width <= 0 {
		return ""
	}

	// Left side: "ghent  owner/repo  PR #42"
	left := styles.StatusBarLeft.Render("ghent")
	if data.Repo != "" {
		left += "  " + styles.StatusBarDim.Render(data.Repo)
	}
	if data.PR > 0 {
		left += "  " + styles.BadgePurple.Render(fmt.Sprintf("PR #%d", data.PR))
	}
	if data.PRTitle != "" {
		left += "  " + styles.StatusBarDim.Render(styles.Truncate(data.PRTitle, 40))
	}
	if data.Left != "" {
		left += "  " + data.Left
	}

	// Right side: counts + badge
	right := ""
	if data.Right != "" {
		right = data.Right
	}
	if data.RightBadge != "" {
		badgeStyle := styles.BadgeRed
		if data.BadgeColor != "" {
			badgeStyle = lipgloss.NewStyle().
				Foreground(data.BadgeColor).
				Bold(true).
				Padding(0, 1)
		}
		if right != "" {
			right += "  "
		}
		right += badgeStyle.Render(data.RightBadge)
	}

	// Compose with manual padding (avoid lipgloss.Width on inner elements)
	leftW := lipgloss.Width(left)
	rightW := lipgloss.Width(right)
	gap := width - leftW - rightW - 2 // 2 for left/right padding
	if gap < 1 {
		// Narrow terminal: truncate left, keep right
		if rightW > 0 && width > rightW+4 {
			maxLeft := width - rightW - 3
			left = styles.Truncate(left, maxLeft)
			gap = 1
		} else {
			gap = 1
		}
	}

	bar := " " + left + styles.Pad(gap) + right + " "

	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(string(styles.Text))).
		Render(bar) + styles.ANSIReset
}
