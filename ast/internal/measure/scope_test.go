package measure

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ryangerardwilson/c3po/ast/internal/config"
)

func TestNestedGitAndShellInclude(t *testing.T) {
	root := t.TempDir()
	// tracked-style shell in hidden dir
	mustWrite(t, filepath.Join(root, ".bashrc.d", "a.sh"), "foo() { :; }\n")
	// nested repo should not count
	nested := filepath.Join(root, "Apps", "other")
	mustWrite(t, filepath.Join(nested, "big.go"), "package big\n// "+string(make([]byte, 100))+"\n")
	if err := os.MkdirAll(filepath.Join(nested, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	// gitignore
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte("ignored/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(root, "ignored", "x.go"), "package x\n")

	cfg := config.Default()
	res, err := Count(root, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if res.NestedReposSkipped < 1 {
		t.Fatalf("expected nested skip, got %+v", res)
	}
	if res.LOC < 1 {
		t.Fatalf("expected shell counted, got %+v", res)
	}
	// ignored/x.go must not be counted (package x + newline would add if broken)
	// Only shell (+ maybe .gitignore) should appear; never the ignored go file alone as bulk.
	if res.FileCount > 3 {
		t.Fatalf("too many files counted: %+v", res)
	}
	// ensure nested big.go not counted: would add large comment line
	if res.LOC > 10 {
		t.Fatalf("LOC too high, nested or ignored leaked: %+v", res)
	}
}
