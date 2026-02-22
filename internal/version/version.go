// Package version provides build information set via ldflags.
package version

import "fmt"

// Build information, set via ldflags at compile time.
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

// String returns a formatted version string.
func String() string {
	return fmt.Sprintf("ghent %s (commit: %s, built: %s)", Version, Commit, BuildDate)
}
