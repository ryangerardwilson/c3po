package dirt

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// isolate keeps the user's git config and hooks out of the fixtures.
func isolate(t *testing.T) string {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir, "-c", "user.name=t", "-c", "user.email=t@example.com"}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func write(t *testing.T, path, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

// fixture makes a repository on main with files committed.
func fixture(t *testing.T, dir string, files ...string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	git(t, dir, "init", "-q", "-b", "main")
	for _, f := range files {
		write(t, filepath.Join(dir, f), "one\n")
	}
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-q", "--allow-empty", "-m", "init")
}

func run(args ...string) (string, string, int) {
	var out, errb bytes.Buffer
	code := Run(args, &out, &errb)
	return out.String(), errb.String(), code
}

func TestCleanRepositoryIsOmitted(t *testing.T) {
	dir := isolate(t)
	fixture(t, filepath.Join(dir, "app"), "README.md")
	out, errs, code := run(dir)
	if code != 0 || out != "" || errs != "" {
		t.Fatalf("code %d\nout %q\nerr %q", code, out, errs)
	}
}

func TestDirtyRepositoryIsPrinted(t *testing.T) {
	dir := isolate(t)
	app := filepath.Join(dir, "app")
	fixture(t, app, "README.md")
	write(t, filepath.Join(app, "README.md"), "two\n")
	write(t, filepath.Join(app, "notes.txt"), "x\n")
	out, errs, code := run(dir)
	want := app + "\n## main\n M README.md\n?? notes.txt\n"
	if code != 0 || out != want || errs != "" {
		t.Fatalf("code %d\nout %q\nwant %q\nerr %q", code, out, want, errs)
	}
}

func TestNestedRepositoryIsSeparate(t *testing.T) {
	dir := isolate(t)
	app := filepath.Join(dir, "app")
	fixture(t, app, "README.md", ".gitignore")
	write(t, filepath.Join(app, ".gitignore"), "vendor/\n")
	git(t, app, "commit", "-q", "-am", "ignore vendor")
	lib := filepath.Join(app, "vendor", "lib")
	fixture(t, lib, "lib.go")
	write(t, filepath.Join(app, "README.md"), "two\n")
	write(t, filepath.Join(lib, "lib.go"), "two\n")
	out, _, code := run(dir)
	want := app + "\n## main\n M README.md\n\n" + lib + "\n## main\n M lib.go\n"
	if code != 0 || out != want {
		t.Fatalf("code %d\nout %q\nwant %q", code, out, want)
	}
}

func TestSymlinkIsNotFollowed(t *testing.T) {
	dir := isolate(t)
	outside := filepath.Join(t.TempDir(), "outside")
	fixture(t, outside)
	write(t, filepath.Join(outside, "x.txt"), "x\n")
	scan := filepath.Join(dir, "scan")
	if err := os.MkdirAll(scan, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(scan, "link")); err != nil {
		t.Fatal(err)
	}
	out, errs, code := run(scan)
	if code != 0 || out != "" || errs != "" {
		t.Fatalf("code %d\nout %q\nerr %q", code, out, errs)
	}
}

func TestSkippedNamesAreNotEntered(t *testing.T) {
	dir := isolate(t)
	pkg := filepath.Join(dir, "web", "node_modules", "pkg")
	fixture(t, pkg)
	write(t, filepath.Join(pkg, "x.txt"), "x\n")
	for _, name := range []string{".venv", "venv", ".cache", "__pycache__", ".npm", ".cargo", "site-packages"} {
		other := filepath.Join(dir, "web", name, "repo")
		fixture(t, other)
		write(t, filepath.Join(other, "x.txt"), "x\n")
	}
	out, errs, code := run(dir)
	if code != 0 || out != "" || errs != "" {
		t.Fatalf("code %d\nout %q\nerr %q", code, out, errs)
	}
	out, _, code = run(filepath.Join(dir, "web", "node_modules"))
	want := pkg + "\n## main\n?? x.txt\n"
	if code != 0 || out != want {
		t.Fatalf("root named node_modules: code %d\nout %q\nwant %q", code, out, want)
	}
}

func TestHomeLocalIsSkipped(t *testing.T) {
	dir := isolate(t)
	app := filepath.Join(dir, "Apps", "app")
	fixture(t, app)
	write(t, filepath.Join(app, "x.txt"), "x\n")
	hidden := filepath.Join(dir, ".local", "share", "plugin")
	fixture(t, hidden)
	write(t, filepath.Join(hidden, "x.txt"), "x\n")
	nested := filepath.Join(dir, "Work", "proj", ".local", "tool")
	fixture(t, nested)
	write(t, filepath.Join(nested, "x.txt"), "x\n")

	out, errs, code := run(dir)
	if code != 0 || errs != "" || !strings.Contains(out, hidden+"\n") {
		t.Fatalf("outside home: code %d\nout %q\nerr %q", code, out, errs)
	}

	t.Setenv("HOME", dir)
	out, errs, code = run(dir)
	want := app + "\n## main\n?? x.txt\n\n" + nested + "\n## main\n?? x.txt\n"
	if code != 0 || out != want || errs != "" {
		t.Fatalf("home: code %d\nout %q\nwant %q\nerr %q", code, out, want, errs)
	}
	out, _, code = run(filepath.Join(dir, ".local"))
	want = hidden + "\n## main\n?? x.txt\n"
	if code != 0 || out != want {
		t.Fatalf("explicit: code %d\nout %q\nwant %q", code, out, want)
	}
}

func TestScanInsideRepositoryKeepsOnlyThatPath(t *testing.T) {
	dir := isolate(t)
	app := filepath.Join(dir, "app")
	fixture(t, app, "README.md", "src/main.go", "docs/a.md")
	write(t, filepath.Join(app, "README.md"), "two\n")
	write(t, filepath.Join(app, "src", "main.go"), "two\n")
	write(t, filepath.Join(app, "fresh", "deep", "x.txt"), "x\n")
	lib := filepath.Join(app, "src", "lib")
	fixture(t, lib, "lib.go")
	write(t, filepath.Join(lib, "lib.go"), "two\n")

	out, _, code := run(filepath.Join(app, "src"))
	want := app + "\n## main\n M src/main.go\n?? src/lib/\n\n" + lib + "\n## main\n M lib.go\n"
	if code != 0 || out != want {
		t.Fatalf("src: code %d\nout %q\nwant %q", code, out, want)
	}
	out, _, code = run(filepath.Join(app, "docs"))
	if code != 0 || out != "" {
		t.Fatalf("clean docs: code %d\nout %q", code, out)
	}
	out, _, code = run(filepath.Join(app, "fresh", "deep"))
	want = app + "\n## main\n?? fresh/\n"
	if code != 0 || out != want {
		t.Fatalf("untracked parent: code %d\nout %q\nwant %q", code, out, want)
	}
}

func TestInsideReadsRenamesAndQuotes(t *testing.T) {
	cases := []struct {
		line string
		want bool
	}{
		{" M src/a.go", true},
		{" M srcx/a.go", false},
		{"R  old/a.go -> src/a.go", true},
		{"R  src/a.go -> old/a.go", true},
		{"R  x/a.go -> y/a.go", false},
		{`?? "src/sp ace\303\251.go"`, true},
		{"?? s/", false},
		{"?? src/", true},
	}
	for _, tc := range cases {
		if got := inside(tc.line, "src"); got != tc.want {
			t.Fatalf("inside(%q) = %v", tc.line, got)
		}
	}
}

func TestGitFileIsARepository(t *testing.T) {
	dir := isolate(t)
	app := filepath.Join(dir, "app")
	if err := os.MkdirAll(app, 0o755); err != nil {
		t.Fatal(err)
	}
	git(t, app, "init", "-q", "-b", "main", "--separate-git-dir", filepath.Join(t.TempDir(), "app.git"))
	write(t, filepath.Join(app, "x.txt"), "x\n")
	out, _, code := run(dir)
	if code != 0 || !strings.HasPrefix(out, app+"\n## ") || !strings.Contains(out, "?? x.txt") {
		t.Fatalf("code %d\nout %q", code, out)
	}
}

func TestUnlistableDirectoryIsSkipped(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root can list every directory")
	}
	dir := isolate(t)
	app := filepath.Join(dir, "app")
	fixture(t, app)
	write(t, filepath.Join(app, "x.txt"), "x\n")
	locked := filepath.Join(dir, "locked")
	if err := os.Mkdir(locked, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })
	out, errs, code := run(dir)
	if code != 0 || out != app+"\n## main\n?? x.txt\n" || errs != "" {
		t.Fatalf("code %d\nout %q\nerr %q", code, out, errs)
	}
}

func TestFailedStatusExitsOneAfterPrinting(t *testing.T) {
	dir := isolate(t)
	app := filepath.Join(dir, "app")
	fixture(t, app)
	write(t, filepath.Join(app, "x.txt"), "x\n")
	broken := filepath.Join(dir, "broken")
	if err := os.MkdirAll(broken, 0o755); err != nil {
		t.Fatal(err)
	}
	git(t, broken, "init", "-q", "--bare", ".git")
	zed := filepath.Join(dir, "zed")
	fixture(t, zed)
	write(t, filepath.Join(zed, "y.txt"), "y\n")
	out, errs, code := run(dir)
	want := app + "\n## main\n?? x.txt\n\n" + zed + "\n## main\n?? y.txt\n"
	if code != 1 || out != want {
		t.Fatalf("code %d\nout %q\nwant %q", code, out, want)
	}
	if !strings.Contains(errs, "dirt: "+broken+": git status failed") {
		t.Fatalf("stderr %q", errs)
	}
}

func TestEmptyScanPrintsNothing(t *testing.T) {
	dir := isolate(t)
	out, errs, code := run(dir)
	if code != 0 || out != "" || errs != "" {
		t.Fatalf("code %d\nout %q\nerr %q", code, out, errs)
	}
}

func TestHomeIsTheDefaultRoot(t *testing.T) {
	dir := isolate(t)
	t.Setenv("HOME", dir)
	app := filepath.Join(dir, "app")
	fixture(t, app)
	write(t, filepath.Join(app, "x.txt"), "x\n")
	out, _, code := run()
	if code != 0 || out != app+"\n## main\n?? x.txt\n" {
		t.Fatalf("code %d\nout %q", code, out)
	}
}

func TestRelativePathIsResolved(t *testing.T) {
	dir := isolate(t)
	app := filepath.Join(dir, "work", "app")
	fixture(t, app)
	write(t, filepath.Join(app, "x.txt"), "x\n")
	t.Chdir(dir)
	out, _, code := run("./work/../work")
	if code != 0 || out != app+"\n## main\n?? x.txt\n" {
		t.Fatalf("code %d\nout %q", code, out)
	}
}

func TestUsageErrorsExitTwo(t *testing.T) {
	dir := isolate(t)
	file := filepath.Join(dir, "file.txt")
	write(t, file, "x\n")
	for _, args := range [][]string{
		{filepath.Join(dir, "missing")},
		{file},
		{dir, dir},
		{""},
	} {
		out, errs, code := run(args...)
		if code != 2 || out != "" || !strings.Contains(errs, usageLine) {
			t.Fatalf("%q: code %d\nout %q\nerr %q", args, code, out, errs)
		}
	}
}

func TestHelpAndVersion(t *testing.T) {
	for _, word := range []string{"help", "-h", "--help"} {
		out, errs, code := run(word)
		if code != 0 || out != Help || errs != "" {
			t.Fatalf("%s: code %d\n%s", word, code, out)
		}
	}
	if !strings.Contains(Help, "c3po dirt <path>") {
		t.Fatal("help does not name c3po dirt <path>")
	}
	for _, word := range []string{"version", "--version", "-v"} {
		out, _, code := run(word)
		if code != 0 || !strings.HasPrefix(out, "c3po 0.5.0") {
			t.Fatalf("%s: code %d\n%s", word, code, out)
		}
	}
}
