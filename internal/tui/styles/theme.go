// Package styles defines the Tokyo Night color palette and Lipgloss style
// definitions used across all ghent TUI views.
package styles

import "github.com/charmbracelet/lipgloss"

// Tokyo Night color palette — hex values match docs/tui-mockups.html.
const (
	// Backgrounds
	Background lipgloss.Color = "#1a1b26" // term-bg
	Surface    lipgloss.Color = "#24283b" // term-surface (cards, panels)
	Surface2   lipgloss.Color = "#292e42" // term-surface2 (elevated elements)

	// Borders
	Border      lipgloss.Color = "#3b4261" // lip-border
	BorderFocus lipgloss.Color = "#7aa2f7" // lip-border-focus (active element)

	// Text
	Text   lipgloss.Color = "#c0caf5" // primary text
	Dim    lipgloss.Color = "#565f89" // secondary/muted text
	Bright lipgloss.Color = "#c0caf5" // bright text (same as Text in Tokyo Night)

	// Semantic colors
	Green  lipgloss.Color = "#9ece6a" // pass, success, additions
	Red    lipgloss.Color = "#f7768e" // fail, error, deletions
	Blue   lipgloss.Color = "#7aa2f7" // info, links, active
	Purple lipgloss.Color = "#bb9af7" // PR numbers, thread IDs
	Cyan   lipgloss.Color = "#7dcfff" // file paths, code
	Orange lipgloss.Color = "#ff9e64" // authors, warnings
	Yellow lipgloss.Color = "#e0af68" // line numbers, pending
	Teal   lipgloss.Color = "#73daca" // own comments
	Pink   lipgloss.Color = "#ff007c" // urgent/critical
)

// ANSIReset is an explicit ANSI reset sequence to prevent color bleed
// between styled elements.
const ANSIReset = "\033[0m"

// ResetStyle returns an ANSI reset sequence. Use between styled elements
// to prevent color bleed (pitfall 7.6 in testing-strategy.md).
func ResetStyle() string {
	return ANSIReset
}
