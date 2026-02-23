package cli

// GlobalFlags holds flags shared across all subcommands.
type GlobalFlags struct {
	Repo    string
	Format  string
	Verbose bool
	NoTUI   bool
	Debug   bool
	IsTTY   bool // resolved at runtime in PersistentPreRunE
	PR      int
}
