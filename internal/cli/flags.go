package cli

import "time"

// GlobalFlags holds flags shared across all subcommands.
type GlobalFlags struct {
	Repo    string
	Format  string
	Verbose bool
	NoTUI   bool
	Debug   bool
	IsTTY   bool // resolved at runtime in PersistentPreRunE
	PR      int
	Since   time.Time // parsed from --since flag; zero means no filter
}
