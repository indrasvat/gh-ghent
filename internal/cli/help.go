package cli

import (
	"text/template"

	"github.com/charmbracelet/lipgloss"
	"github.com/cli/go-gh/v2/pkg/term"
	"github.com/spf13/cobra"

	"github.com/indrasvat/gh-ghent/internal/tui/styles"
	"github.com/indrasvat/gh-ghent/internal/version"
)

// isTTYOutput returns true if stdout is a terminal.
// Evaluated lazily at template render time (not at init), because
// --help/--version run before PersistentPreRunE.
func isTTYOutput() bool {
	return term.FromEnv().IsTerminalOutput()
}

// styled applies a lipgloss style to a string only when outputting to a TTY.
// Returns the string unchanged when piped.
func styled(style lipgloss.Style, s string) string {
	if !isTTYOutput() {
		return s
	}
	return style.Render(s)
}

// ── Lipgloss styles for help output ─────────────────────────────

var (
	helpBlue   = lipgloss.NewStyle().Foreground(lipgloss.Color(string(styles.Blue)))
	helpBold   = lipgloss.NewStyle().Bold(true)
	helpCyan   = lipgloss.NewStyle().Foreground(lipgloss.Color(string(styles.Cyan)))
	helpDim    = lipgloss.NewStyle().Foreground(lipgloss.Color(string(styles.Dim)))
	helpGreen  = lipgloss.NewStyle().Foreground(lipgloss.Color(string(styles.Green)))
	helpOrange = lipgloss.NewStyle().Foreground(lipgloss.Color(string(styles.Orange)))
	helpPurple = lipgloss.NewStyle().Foreground(lipgloss.Color(string(styles.Purple)))
	helpYellow = lipgloss.NewStyle().Foreground(lipgloss.Color(string(styles.Yellow)))

	helpTitle   = lipgloss.NewStyle().Foreground(lipgloss.Color(string(styles.Blue))).Bold(true)
	helpSection = lipgloss.NewStyle().Foreground(lipgloss.Color(string(styles.Blue))).Bold(true)
)

// ── Template functions ──────────────────────────────────────────

// helpTemplateFuncs returns functions available in help/version templates.
func helpTemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"isTTY":     isTTYOutput,
		"blue":      func(s string) string { return styled(helpBlue, s) },
		"bold":      func(s string) string { return styled(helpBold, s) },
		"cyan":      func(s string) string { return styled(helpCyan, s) },
		"dim":       func(s string) string { return styled(helpDim, s) },
		"green":     func(s string) string { return styled(helpGreen, s) },
		"orange":    func(s string) string { return styled(helpOrange, s) },
		"purple":    func(s string) string { return styled(helpPurple, s) },
		"yellow":    func(s string) string { return styled(helpYellow, s) },
		"title":     func(s string) string { return styled(helpTitle, s) },
		"section":   func(s string) string { return styled(helpSection, s) },
		"shortHash": version.ShortCommit,
		"shortDate": version.ShortDate,
	}
}

// ── Version template ────────────────────────────────────────────

const versionTemplate = `{{if isTTY}}  {{title .Name}} {{green .Version}}  {{dim "·"}}  {{purple (printf "commit %s" shortHash)}}  {{dim "·"}}  {{dim (printf "built %s" shortDate)}}
  {{dim "Agentic PR monitoring for GitHub"}}
{{else}}{{.Name}} {{.Version}} (commit: {{shortHash}}, built: {{shortDate}})
{{end}}`

// ── Root help template ──────────────────────────────────────────

const rootHelpTemplate = `  {{title .Name}} {{dim "—"}} {{dim .Short}}

{{.Long}}

{{section "Commands:"}}
{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}  {{cyan (rpad .Name .NamePadding)}}   {{.Short}}
{{end}}{{end}}
{{section "Examples:"}}
{{.Example}}

{{section "Flags:"}}
{{.LocalFlags.FlagUsages}}
  Use "{{.CommandPath}} [command] --help" for more information about a command.
`

// ── Subcommand help template ────────────────────────────────────

const subcommandHelpTemplate = `  {{title .CommandPath}}{{if .Short}} {{dim "—"}} {{dim .Short}}{{end}}

{{if .Long}}{{.Long}}

{{end}}{{if .HasExample}}{{section "Examples:"}}
{{.Example}}

{{end}}{{section "Flags:"}}
{{.LocalFlags.FlagUsages}}{{if .HasAvailableInheritedFlags}}
{{section "Global Flags:"}}
{{.InheritedFlags.FlagUsages}}{{end}}
`

// setupHelp registers custom templates and template functions on the root command.
func setupHelp(root *cobra.Command) {
	// Register template functions globally for all commands.
	for name, fn := range helpTemplateFuncs() {
		cobra.AddTemplateFunc(name, fn)
	}

	root.SetVersionTemplate(versionTemplate)
	root.SetHelpTemplate(rootHelpTemplate)

	// Apply subcommand template to all child commands.
	for _, child := range root.Commands() {
		child.SetHelpTemplate(subcommandHelpTemplate)
	}
}
