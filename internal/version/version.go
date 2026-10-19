package version

var (
	Version    = "0.3.0"
	Commit     = "unknown"
	SourceHash = "unknown"
	BuildTime  = "unknown"
)

// String is the version stamped into the c3po binary.
func String() string {
	if Commit == "" || Commit == "unknown" {
		return Version
	}
	return Version + "+" + Commit
}
