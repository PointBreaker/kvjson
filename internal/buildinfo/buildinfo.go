package buildinfo

import "fmt"

// These values are replaced at build time by the release workflow.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// String returns a human-readable build description.
func String() string {
	return fmt.Sprintf("kv %s (commit %s, built %s)", Version, Commit, Date)
}
