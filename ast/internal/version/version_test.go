package version

import "testing"

func TestDirtyCommitIsQualified(t *testing.T) {
	oldV, oldC, oldH := Version, Commit, SourceHash
	t.Cleanup(func() {
		Version, Commit, SourceHash = oldV, oldC, oldH
	})
	Version = "0.1.0"
	Commit = "7f83fefb"
	if Dirty() || String() != "0.1.0+7f83fefb" {
		t.Fatalf("clean %s dirty %v", String(), Dirty())
	}
	Version = "0.0.0-dev"
	if String() != "0.0.0-dev+7f83fefb" {
		t.Fatalf("dev %s", String())
	}
	Version = "0.1.0"
	Commit = "7f83fefb-dirty"
	SourceHash = "abc123"
	if !Dirty() || String() != "0.1.0+7f83fefb-dirty" {
		t.Fatalf("dirty %s %v", String(), Dirty())
	}
}
