package styles

import (
	"fmt"
	"strings"
	"testing"
)

func TestTokyoNightPalette(t *testing.T) {
	// Verify all palette colors are valid hex values
	colors := map[string]string{
		"Background":  string(Background),
		"Surface":     string(Surface),
		"Surface2":    string(Surface2),
		"Border":      string(Border),
		"BorderFocus": string(BorderFocus),
		"Text":        string(Text),
		"Dim":         string(Dim),
		"Bright":      string(Bright),
		"Green":       string(Green),
		"Red":         string(Red),
		"Blue":        string(Blue),
		"Purple":      string(Purple),
		"Cyan":        string(Cyan),
		"Orange":      string(Orange),
		"Yellow":      string(Yellow),
		"Teal":        string(Teal),
		"Pink":        string(Pink),
	}

	for name, hex := range colors {
		if !strings.HasPrefix(hex, "#") {
			t.Errorf("%s: expected hex prefix #, got %q", name, hex)
		}
		if len(hex) != 7 {
			t.Errorf("%s: expected 7-char hex (e.g. #1a1b26), got %q (len=%d)", name, hex, len(hex))
		}
	}
}

func TestStylesRenderWithoutPanic(t *testing.T) {
	// Verify all styles can render text without panicking at various widths
	widths := []int{20, 40, 80, 120, 200}
	for _, w := range widths {
		t.Run(fmt.Sprintf("width_%d", w), func(t *testing.T) {
			text := strings.Repeat("x", w)

			// These should not panic
			_ = StatusBar.Render(text)
			_ = StatusBarLeft.Render(text)
			_ = StatusBarDim.Render(text)
			_ = BadgeBlue.Render("PASS")
			_ = BadgeGreen.Render("4 passed")
			_ = BadgeRed.Render("1 failed")
			_ = BadgeYellow.Render("pending")
			_ = BadgePurple.Render("PR #42")
			_ = HelpBar.Render(text)
			_ = HelpKey.Render("j/k")
			_ = HelpSep.Render(" · ")
			_ = ListItemNormal.Render(text)
			_ = ListItemSelected.Render(text)
			_ = ListItemDim.Render(text)
			_ = FilePath.Render("internal/api/graphql.go")
			_ = LineNumber.Render(":47")
			_ = Author.Render("@reviewer1")
			_ = ThreadID.Render("PRRT_kwDON1...")
			_ = OwnComment.Render("@you")
			_ = Box.Render(text)
			_ = BoxFocused.Render(text)
			_ = CheckPass.Render("✓")
			_ = CheckFail.Render("✗")
			_ = CheckPending.Render("○")
			_ = CheckRunning.Render("⠋")
			_ = DiffAdd.Render("+    return nil, fmt.Errorf(...)")
			_ = DiffDel.Render("-    return nil, err")
			_ = DiffContext.Render(" func (c *Client) FetchThreads(...)")
			_ = DiffHeader.Render("@@ -44,8 +44,10 @@")
			_ = CheckboxOn.Render("[✓]")
			_ = CheckboxOff.Render("[ ]")
			_ = SummaryCount.Render("5")
			_ = SummaryLabel.Render("UNRESOLVED")
		})
	}
}

func TestNoLipglossBackground(t *testing.T) {
	// We can't directly inspect lipgloss styles for Background() usage at runtime,
	// but we can verify our terminal background approach works.
	s := ResetStyle()
	if s != "\033[0m" {
		t.Errorf("ResetStyle() = %q, want ANSI reset", s)
	}
}

func TestPad(t *testing.T) {
	tests := []struct {
		n    int
		want string
	}{
		{0, ""},
		{-1, ""},
		{1, " "},
		{5, "     "},
		{10, "          "},
	}
	for _, tt := range tests {
		got := Pad(tt.n)
		if got != tt.want {
			t.Errorf("Pad(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		s      string
		maxLen int
		want   string
	}{
		{"hello", 10, "hello"},
		{"hello", 5, "hello"},
		{"hello world", 5, "hell…"},
		{"hello", 1, "…"},
		{"hello", 0, ""},
		{"", 5, ""},
	}
	for _, tt := range tests {
		got := Truncate(tt.s, tt.maxLen)
		if got != tt.want {
			t.Errorf("Truncate(%q, %d) = %q, want %q", tt.s, tt.maxLen, got, tt.want)
		}
	}
}

func TestANSIResetConstant(t *testing.T) {
	if ANSIReset != "\033[0m" {
		t.Errorf("ANSIReset = %q, want ANSI escape reset", ANSIReset)
	}
}
