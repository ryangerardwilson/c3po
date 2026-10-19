package version

// Version is the semantic version, stamped at build/release time.
// Source defaults are replaced via -ldflags when building releases or install.sh.
var Version = "0.4.0"

// Commit is the short git SHA when available (ldflags).
// A dirty checkout is stamped with a "-dirty" suffix so it cannot be
// mistaken for the clean commit.
var Commit = ""

// SourceHash is a hash of the Go sources used to build this binary.
var SourceHash = ""

// BuildTime is RFC3339 UTC build time when stamped (ldflags).
var BuildTime = ""

// Dirty reports whether the stamped commit was qualified as dirty.
func Dirty() bool {
	return len(Commit) > 6 && Commit[len(Commit)-6:] == "-dirty"
}

// String returns a human-readable version line.
func String() string {
	if Commit == "" {
		return Version
	}
	if Version == "0.0.0-dev" || Version == "0.0.0" {
		return Version + "+" + Commit
	}
	return Version + "+" + Commit
}
