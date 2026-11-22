// Package archtest encodes the import rules between the cores and their
// edges, so a core that reaches for HTTP, SQL, a terminal, or its own
// adapters fails go test.
package archtest

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const module = "github.com/ryangerardwilson/c3po/"

// infra is what no core may import, directly or through another package.
var infra = []string{
	"net/http",
	"database/sql",
	"modernc.org/sqlite",
	"golang.org/x/term",
	"golang.org/x/sys/unix",
	module + "internal/sqlxdb",
}

// rule forbids pkg from reaching any of deps, directly or transitively,
// and from importing any of direct itself. direct holds side effects such
// as os/exec that the edge wires in.
type rule struct {
	pkg    string
	deps   []string
	direct []string
}

var rules = []rule{
	{pkg: "gmail/internal/desk", deps: with("gmail/internal/api", "gmail/internal/oauth"), direct: []string{"os/exec"}},
	{pkg: "gmail/internal/mailbox", deps: with("gmail/internal/api", "gmail/internal/oauth")},
	{pkg: "gcal/internal/desk", deps: with("gcal/internal/calendar", "gcal/internal/oauth"), direct: []string{"os/exec"}},
	{pkg: "slack/internal/app", deps: with("slack/internal/slackapi", "slack/internal/cachedb", "slack/internal/tty"), direct: []string{"os/exec"}},
	{
		pkg:    "gdrive/internal/syncer",
		deps:   with("gdrive/internal/driveapi", "gdrive/internal/googleapi", "gdrive/internal/auth", "gdrive/internal/localfs"),
		direct: []string{"os"},
	},
	{pkg: "idealake/internal/lake", deps: with("idealake/internal/store", "idealake/internal/xai")},
	// score and funcparse are shared so that cyclotree and c3po ast do not
	// import each other to score a file.
	{pkg: "internal/score", deps: []string{module + "cyclotree", module + "ast"}},
	{pkg: "internal/funcparse", deps: []string{module + "cyclotree", module + "ast"}},
}

func with(local ...string) []string {
	out := append([]string{}, infra...)
	for _, pkg := range local {
		out = append(out, module+pkg)
	}
	return out
}

func TestCoresImportInward(t *testing.T) {
	root := moduleRoot(t)
	for _, r := range rules {
		t.Run(r.pkg, func(t *testing.T) {
			for _, dep := range golist(t, root, "-deps", r.pkg) {
				if hit := forbidden(dep, r.deps); hit != "" {
					t.Errorf("%s reaches %s (rule %s)", r.pkg, dep, hit)
				}
			}
			for _, imp := range golist(t, root, "-f={{join .Imports \"\\n\"}}", r.pkg) {
				if hit := forbidden(imp, r.direct); hit != "" {
					t.Errorf("%s imports %s", r.pkg, imp)
				}
			}
		})
	}
}

// forbidden returns the rule that dep breaks: the rule itself or a
// package under it.
func forbidden(dep string, rules []string) string {
	for _, r := range rules {
		if dep == r || strings.HasPrefix(dep, r+"/") {
			return r
		}
	}
	return ""
}

func golist(t *testing.T, root, flag, pkg string) []string {
	t.Helper()
	cmd := exec.Command("go", "list", flag, "./"+pkg)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "GOWORK=off")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go list %s %s: %v\n%s", flag, pkg, err, out)
	}
	return strings.Fields(string(out))
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for !fileExists(filepath.Join(dir, "go.mod")) {
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
	return dir
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
