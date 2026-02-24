package tui

import "github.com/charmbracelet/bubbles/key"

// AppKeyMap defines global key bindings for the app shell.
// View-specific bindings are handled within each view's Update.
type AppKeyMap struct {
	Tab      key.Binding
	ShiftTab key.Binding
	Enter    key.Binding
	Esc      key.Binding
	Quit     key.Binding

	// Summary shortcuts — jump directly to a view.
	Comments key.Binding
	Checks   key.Binding
	Resolve  key.Binding

	// Cross-view actions.
	OpenPR key.Binding
	Rerun  key.Binding
}

// DefaultKeyMap returns the default global key bindings.
func DefaultKeyMap() AppKeyMap {
	return AppKeyMap{
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "switch view"),
		),
		ShiftTab: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "switch view"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "expand"),
		),
		Esc: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q"),
			key.WithHelp("q", "quit"),
		),
		Comments: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "comments"),
		),
		Checks: key.NewBinding(
			key.WithKeys("k"),
			key.WithHelp("k", "checks"),
		),
		Resolve: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "resolve"),
		),
		OpenPR: key.NewBinding(
			key.WithKeys("o"),
			key.WithHelp("o", "open PR"),
		),
		Rerun: key.NewBinding(
			key.WithKeys("R"),
			key.WithHelp("R", "re-run failed"),
		),
	}
}
