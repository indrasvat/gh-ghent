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
// Cobra's default template prepends "{Name} version", so this excludes the command name.
func String() string {
	return fmt.Sprintf("%s (commit: %s, built: %s)", Version, ShortCommit(), ShortDate())
}

// ShortCommit returns the first 7 characters of the commit hash,
// or "unknown" if the commit is not set.
func ShortCommit() string {
	if len(Commit) > 7 {
		return Commit[:7]
	}
	return Commit
}

// ShortDate returns the date portion of BuildDate (everything before "T"),
// or the full string if there is no "T" separator.
func ShortDate() string {
	if i := len(BuildDate); i > 0 {
		for j := 0; j < i; j++ {
			if BuildDate[j] == 'T' {
				return BuildDate[:j]
			}
		}
	}
	return BuildDate
}
