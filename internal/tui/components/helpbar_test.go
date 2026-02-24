package components

import (
	"strings"
	"testing"
)

func TestRenderHelpBar(t *testing.T) {
	bindings := CommentsListKeys()

	tests := []struct {
		name  string
		width int
		want  []string // expected substrings
	}{
		{
			name:  "wide",
			width: 120,
			want:  []string{"j/k", "navigate", "enter", "expand", "quit"},
		},
		{
			name:  "medium",
			width: 60,
			want:  []string{"j/k", "navigate"}, // at least first items
		},
		{
			name:  "narrow truncates",
			width: 20,
			want:  []string{"j/k"}, // at least the first key
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RenderHelpBar(bindings, tt.width)
			for _, sub := range tt.want {
				if !strings.Contains(got, sub) {
					t.Errorf("RenderHelpBar(width=%d) missing %q in output:\n%s", tt.width, sub, got)
				}
			}
		})
	}
}

func TestRenderHelpBarEmpty(t *testing.T) {
	got := RenderHelpBar(nil, 80)
	if got != "" {
		t.Errorf("expected empty for nil bindings, got %q", got)
	}

	got = RenderHelpBar(CommentsListKeys(), 0)
	if got != "" {
		t.Errorf("expected empty for zero width, got %q", got)
	}
}

func TestPredefinedKeyBindings(t *testing.T) {
	sets := []struct {
		name string
		fn   func() []KeyBinding
		min  int // minimum expected bindings
	}{
		{"CommentsListKeys", CommentsListKeys, 6},
		{"CommentsExpandedKeys", CommentsExpandedKeys, 5},
		{"ChecksListKeys", ChecksListKeys, 5},
		{"ChecksWatchKeys", ChecksWatchKeys, 3},
		{"ResolveKeys", ResolveKeys, 5},
		{"SummaryKeys", SummaryKeys, 4},
	}

	for _, tt := range sets {
		t.Run(tt.name, func(t *testing.T) {
			bindings := tt.fn()
			if len(bindings) < tt.min {
				t.Errorf("%s() returned %d bindings, want at least %d", tt.name, len(bindings), tt.min)
			}
			for _, b := range bindings {
				if b.Key == "" {
					t.Errorf("%s() has binding with empty key", tt.name)
				}
				if b.Action == "" {
					t.Errorf("%s() has binding with empty action for key %q", tt.name, b.Key)
				}
			}
		})
	}
}

func TestPadLine(t *testing.T) {
	tests := []struct {
		input string
		width int
		check func(string) bool
		desc  string
	}{
		{"hello", 10, func(s string) bool { return strings.HasSuffix(s, "     ") }, "padded to 10"},
		{"hello", 5, func(s string) bool { return s == "hello" }, "exact width"},
		{"hello", 3, func(s string) bool { return s == "hello" }, "shorter than input"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got := PadLine(tt.input, tt.width)
			if !tt.check(got) {
				t.Errorf("PadLine(%q, %d) = %q, check failed", tt.input, tt.width, got)
			}
		})
	}
}
