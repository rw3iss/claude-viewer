// Package version exposes build metadata, set via -ldflags at link time.
package version

var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// String returns "vX.Y.Z (commit @ date)".
func String() string {
	return Version + " (" + Commit + " @ " + Date + ")"
}
