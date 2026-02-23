package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/indrasvat/gh-ghent/internal/tui/styles"
)

// KeyBinding represents a single key → action pair for the help bar.
type KeyBinding struct {
	Key    string // Display text for the key (e.g., "j/k", "enter", "tab")
	Action string // Description of the action (e.g., "navigate", "expand")
}

// RenderHelpBar renders the bottom help bar with context-sensitive key binding
// hints. Adapts to the given terminal width by truncating items that don't fit.
func RenderHelpBar(bindings []KeyBinding, width int) string {
	if width <= 0 || len(bindings) == 0 {
		return ""
	}

	var items []string
	for _, b := range bindings {
		item := styles.HelpKey.Render(b.Key) + " " +
			styles.HelpSep.Render(b.Action)
		items = append(items, item)
	}

	// Join with separator, truncating if needed
	sep := "  "
	result := " " + items[0]
	for i := 1; i < len(items); i++ {
		candidate := result + sep + items[i]
		// Check visible width (lipgloss.Width handles ANSI)
		if lipgloss.Width(candidate)+2 > width {
			break
		}
		result = candidate
	}
	result += " "

	// Pad to full width with spaces (not lipgloss.Width — pitfall 7.5)
	visW := lipgloss.Width(result)
	if visW < width {
		result += styles.Pad(width - visW)
	}

	return result + styles.ANSIReset
}

// ── Predefined key binding sets per view ───────────────────────

// CommentsListKeys returns key bindings for the comments list view.
func CommentsListKeys() []KeyBinding {
	return []KeyBinding{
		{"j/k", "navigate"},
		{"enter", "expand"},
		{"r", "resolve"},
		{"y", "copy ID"},
		{"o", "open in browser"},
		{"f", "filter by file"},
		{"tab", "checks view"},
		{"q", "quit"},
	}
}

// CommentsExpandedKeys returns key bindings for the expanded thread view.
func CommentsExpandedKeys() []KeyBinding {
	return []KeyBinding{
		{"esc", "back to list"},
		{"j/k", "scroll"},
		{"r", "resolve thread"},
		{"y", "copy ID"},
		{"o", "open in browser"},
		{"n/p", "next/prev thread"},
		{"q", "quit"},
	}
}

// ChecksListKeys returns key bindings for the checks list view.
func ChecksListKeys() []KeyBinding {
	return []KeyBinding{
		{"j/k", "navigate"},
		{"enter", "view logs"},
		{"l", "view full log"},
		{"o", "open in browser"},
		{"R", "re-run failed"},
		{"tab", "comments view"},
		{"q", "quit"},
	}
}

// ChecksWatchKeys returns key bindings for watch mode.
func ChecksWatchKeys() []KeyBinding {
	return []KeyBinding{
		{"j/k", "navigate"},
		{"enter", "view logs"},
		{"ctrl+c", "stop watching"},
		{"q", "quit"},
	}
}

// ResolveKeys returns key bindings for the resolve view.
func ResolveKeys() []KeyBinding {
	return []KeyBinding{
		{"j/k", "navigate"},
		{"space", "toggle select"},
		{"a", "select all"},
		{"enter", "resolve selected"},
		{"esc", "cancel"},
		{"q", "quit"},
	}
}

// SummaryKeys returns key bindings for the summary dashboard.
func SummaryKeys() []KeyBinding {
	return []KeyBinding{
		{"c", "comments"},
		{"k", "checks"},
		{"r", "resolve"},
		{"o", "open PR"},
		{"R", "re-run failed"},
		{"q", "quit"},
	}
}

// PadLine pads a line to exactly the given width using spaces.
// Use this instead of empty strings for padding (pitfall 7.3).
func PadLine(s string, width int) string {
	visW := lipgloss.Width(s)
	if visW >= width {
		return s
	}
	return s + strings.Repeat(" ", width-visW)
}
