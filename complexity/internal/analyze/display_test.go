package analyze

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ryangerardwilson/c3po/complexity/internal/config"
	"github.com/ryangerardwilson/c3po/complexity/internal/metric"
)

func TestRelativeToInvokeDir(t *testing.T) {
	const stored = "pkg/hot.go"
	abs := filepath.Join("/repo", "pkg", "hot.go")
	res := Result{
		Functions: []metric.Function{{File: stored, Name: "hot"}},
		Hotspots:  []metric.Function{{File: stored, Name: "hot"}},
		abs:       map[string]string{stored: abs},
	}
	shown := res.relativeTo(filepath.Join("/repo", "pkg"))
	if shown.Functions[0].File != "hot.go" || shown.Hotspots[0].File != "hot.go" {
		t.Fatalf("shown %+v", shown.Functions)
	}
	if res.Functions[0].File != stored {
		t.Fatalf("original changed to %s", res.Functions[0].File)
	}
	parent := res.relativeTo("/repo")
	if parent.Functions[0].File != "pkg/hot.go" {
		t.Fatalf("parent %s", parent.Functions[0].File)
	}
	up := res.relativeTo(filepath.Join("/repo", "pkg", "nested"))
	if up.Functions[0].File != "../hot.go" {
		t.Fatalf("up %s", up.Functions[0].File)
	}
}

func TestDiagnosticPathsAreRelative(t *testing.T) {
	const stored = "pkg/hot.go"
	abs := filepath.Join("/repo", "pkg", "hot.go")
	res := Result{
		Diagnostics: []string{stored + ": parse error", abs + ": read error"},
		abs:         map[string]string{stored: abs},
	}
	shown := res.relativeTo(filepath.Join("/repo", "pkg"))
	for _, d := range shown.Diagnostics {
		if !strings.Contains(d, "hot.go") || strings.Contains(d, "/repo") || strings.Contains(d, "pkg/hot.go") {
			t.Fatalf("diag %s", d)
		}
	}
	if got := internalPath(filepath.Join("/repo", "pkg"), "/repo", "hot.go"); got != "pkg/hot.go" {
		t.Fatalf("internal %s", got)
	}
}

func TestBaselineRoundTrip(t *testing.T) {
	root := t.TempDir()
	pkg := filepath.Join(root, "pkg")
	if err := os.MkdirAll(pkg, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "package p\nfunc hot() {}\n"
	if err := os.WriteFile(filepath.Join(pkg, "hot.go"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(pkg)
	res, err := Run(Options{Paths: []string{"."}, Config: config.Default()})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Functions) == 0 {
		t.Fatal("no functions")
	}
	shown := res.ForDisplay()
	if shown.Functions[0].File != "hot.go" {
		t.Fatalf("shown %s", shown.Functions[0].File)
	}
	base := filepath.Join(pkg, "base.json")
	if err := WriteBaseline(base, shown.Functions, shown.Models.Models); err != nil {
		t.Fatal(err)
	}
	buf, err := os.ReadFile(base)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(buf, []byte("pkg/hot.go")) || !bytes.Contains(buf, []byte(`"hot.go"`)) {
		t.Fatalf("baseline %s", buf)
	}
	fns, _, err := loadBaseline(base, res.GitRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(fns) == 0 || fns[0].File != "pkg/hot.go" {
		t.Fatalf("loaded %+v", fns)
	}
	if res.Functions[0].File != "pkg/hot.go" {
		t.Fatalf("identity %s", res.Functions[0].File)
	}
}
