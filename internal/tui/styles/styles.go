package styles

import (
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// ── Status Bar (top) ────────────────────────────────────────────

// StatusBar is the base style for the top status bar.
var StatusBar = lipgloss.NewStyle().
	Foreground(lipgloss.Color(string(Text))).
	Padding(0, 1)

// StatusBarLeft is the left-aligned portion of the status bar.
var StatusBarLeft = lipgloss.NewStyle().
	Foreground(lipgloss.Color(string(Blue))).
	Bold(true)

// StatusBarDim is for secondary text in the status bar.
var StatusBarDim = lipgloss.NewStyle().
	Foreground(lipgloss.Color(string(Dim)))

// ── Badges ──────────────────────────────────────────────────────

// BadgeBlue is for info badges (e.g., PR number).
var BadgeBlue = lipgloss.NewStyle().
	Foreground(lipgloss.Color(string(Blue))).
	Bold(true).
	Padding(0, 1)

// BadgeGreen is for success badges (e.g., "4 passed").
var BadgeGreen = lipgloss.NewStyle().
	Foreground(lipgloss.Color(string(Green))).
	Bold(true).
	Padding(0, 1)

// BadgeRed is for failure badges (e.g., "1 failed", "5 unresolved").
var BadgeRed = lipgloss.NewStyle().
	Foreground(lipgloss.Color(string(Red))).
	Bold(true).
	Padding(0, 1)

// BadgeYellow is for pending/warning badges.
var BadgeYellow = lipgloss.NewStyle().
	Foreground(lipgloss.Color(string(Yellow))).
	Bold(true).
	Padding(0, 1)

// BadgePurple is for PR/thread ID badges.
var BadgePurple = lipgloss.NewStyle().
	Foreground(lipgloss.Color(string(Purple))).
	Bold(true).
	Padding(0, 1)

// ── Help Bar (bottom) ───────────────────────────────────────────

// HelpBar is the base style for the bottom help bar.
var HelpBar = lipgloss.NewStyle().
	Foreground(lipgloss.Color(string(Dim)))

// HelpKey is the keyboard shortcut styling.
var HelpKey = lipgloss.NewStyle().
	Foreground(lipgloss.Color(string(Blue))).
	Bold(true)

// HelpSep is the separator between help items.
var HelpSep = lipgloss.NewStyle().
	Foreground(lipgloss.Color(string(Dim)))

// ── List Items ──────────────────────────────────────────────────

// ListItemNormal is the default list row style.
var ListItemNormal = lipgloss.NewStyle().
	Foreground(lipgloss.Color(string(Text))).
	Padding(0, 1)

// ListItemSelected is the style for the cursor/selected row.
// Uses a left border accent as shown in the mockups.
var ListItemSelected = lipgloss.NewStyle().
	Foreground(lipgloss.Color(string(Text))).
	Padding(0, 1).
	Border(lipgloss.ThickBorder(), false, false, false, true).
	BorderForeground(lipgloss.Color(string(Blue)))

// ListItemDim is for de-emphasized list items.
var ListItemDim = lipgloss.NewStyle().
	Foreground(lipgloss.Color(string(Dim))).
	Padding(0, 1)

// ── File Path / Code ────────────────────────────────────────────

// FilePath is for file paths (cyan in mockups).
var FilePath = lipgloss.NewStyle().
	Foreground(lipgloss.Color(string(Cyan)))

// LineNumber is for line numbers (yellow in mockups).
var LineNumber = lipgloss.NewStyle().
	Foreground(lipgloss.Color(string(Yellow)))

// Author is for author names (orange in mockups).
var Author = lipgloss.NewStyle().
	Foreground(lipgloss.Color(string(Orange)))

// ThreadID is for thread IDs (purple, smaller).
var ThreadID = lipgloss.NewStyle().
	Foreground(lipgloss.Color(string(Purple)))

// OwnComment is for the current user's comments (teal).
var OwnComment = lipgloss.NewStyle().
	Foreground(lipgloss.Color(string(Teal)))

// ── Borders ─────────────────────────────────────────────────────

// Box is a standard bordered box using the theme border color.
var Box = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color(string(Border)))

// BoxFocused is a bordered box with the focus accent color.
var BoxFocused = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color(string(BorderFocus)))

// ── Check Status ────────────────────────────────────────────────

// CheckPass is for passed check icons/text.
var CheckPass = lipgloss.NewStyle().
	Foreground(lipgloss.Color(string(Green)))

// CheckFail is for failed check icons/text.
var CheckFail = lipgloss.NewStyle().
	Foreground(lipgloss.Color(string(Red)))

// CheckPending is for pending/queued check icons/text.
var CheckPending = lipgloss.NewStyle().
	Foreground(lipgloss.Color(string(Yellow)))

// CheckRunning is for in-progress check icons/text.
var CheckRunning = lipgloss.NewStyle().
	Foreground(lipgloss.Color(string(Blue)))

// ── Diff Hunk ───────────────────────────────────────────────────

// DiffAdd is for added lines (+) in diff hunks.
var DiffAdd = lipgloss.NewStyle().
	Foreground(lipgloss.Color(string(Green)))

// DiffDel is for deleted lines (-) in diff hunks.
var DiffDel = lipgloss.NewStyle().
	Foreground(lipgloss.Color(string(Red)))

// DiffContext is for context lines in diff hunks.
var DiffContext = lipgloss.NewStyle().
	Foreground(lipgloss.Color(string(Dim)))

// DiffHeader is for diff hunk headers (@@...@@).
var DiffHeader = lipgloss.NewStyle().
	Foreground(lipgloss.Color(string(Purple)))

// ── Resolve ─────────────────────────────────────────────────────

// CheckboxOn is the style for a checked checkbox.
var CheckboxOn = lipgloss.NewStyle().
	Foreground(lipgloss.Color(string(Green)))

// CheckboxOff is the style for an unchecked checkbox.
var CheckboxOff = lipgloss.NewStyle().
	Foreground(lipgloss.Color(string(Dim)))

// ── Summary ─────────────────────────────────────────────────────

// SummaryCount is for large KPI numbers.
var SummaryCount = lipgloss.NewStyle().
	Bold(true)

// SummaryLabel is for labels under KPI numbers.
var SummaryLabel = lipgloss.NewStyle().
	Foreground(lipgloss.Color(string(Dim)))

// ── Terminal Background ─────────────────────────────────────────

// SetAppBackground sets the terminal background color to the Tokyo Night
// background. Call BEFORE starting BubbleTea.
//
// CRITICAL: Do NOT use lipgloss.Background() — it only affects rendered
// characters, leaving empty cells with the terminal's default background
// (pitfall 7.1 in testing-strategy.md).
func SetAppBackground() *termenv.Output {
	output := termenv.NewOutput(os.Stdout)
	output.SetBackgroundColor(output.Color(string(Background)))
	return output
}

// ResetAppBackground resets the terminal to its default state.
// Call AFTER BubbleTea exits, before os.Exit.
func ResetAppBackground(output *termenv.Output) {
	if output != nil {
		output.Reset()
	}
}

// ── Helpers ─────────────────────────────────────────────────────

// Pad returns a string of n spaces. Use this for manual padding instead of
// lipgloss.Width() on inner elements, which causes padding bleed
// (pitfall 7.5 in testing-strategy.md).
func Pad(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.Repeat(" ", n)
}

// Truncate truncates a string to maxLen, appending "…" if truncated.
func Truncate(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	if maxLen <= 1 {
		return "…"
	}
	return string(runes[:maxLen-1]) + "…"
}
