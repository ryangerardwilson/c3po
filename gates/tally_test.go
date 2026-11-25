package gates

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// tempRepo makes a git repository with one committed file, a.go, and
// moves the test into it. It never touches the c3po checkout.
func tempRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(dir))
	t.Chdir(dir)
	write(t, "a.go", "package a\n\nfunc A() {}\n")
	write(t, "sub/b.go", "package sub\n")
	git(t, "init", "-q")
	git(t, "add", ".")
	git(t, "-c", "user.name=t", "-c", "user.email=t@t", "-c", "commit.gpgsign=false", "commit", "-q", "-m", "init")
	return dir
}

func git(t *testing.T, args ...string) {
	t.Helper()
	if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
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

// fullSheet is one pass line per provision, then the lines in extra, which
// replace the line of the same section.
func fullSheet(extra map[string]string) string {
	var b strings.Builder
	for _, p := range architecture {
		line, ok := extra[p.section]
		if !ok {
			line = p.section + " " + receipt(p) + " pass none"
		}
		if line != "" {
			b.WriteString(line + "\n")
		}
	}
	return b.String()
}

func line(section, rest string) string {
	for _, p := range architecture {
		if p.section == section {
			return section + " " + receipt(p) + " " + rest
		}
	}
	return ""
}

func tally(t *testing.T, sheetText string) (string, string, int) {
	t.Helper()
	write(t, "sheet.txt", sheetText)
	return call("code", "architecture", "tally", "sheet.txt")
}

func TestTallyCompleteSheet(t *testing.T) {
	tempRepo(t)
	// the sheet is untracked, so it is not a changed file
	out, errOut, code := tally(t, fullSheet(map[string]string{"§2.3": line("§2.3", "pass a.go:3")}))
	if code != 0 || out != "140 of 140 judged\n" || errOut != "" {
		t.Fatalf("code %d\n%s%s", code, out, errOut)
	}
}

func TestTallyMissingSections(t *testing.T) {
	tempRepo(t)
	out, _, code := tally(t, fullSheet(map[string]string{"§1.3": "", "§2.4": "", "§8": ""}))
	want := "137 of 140 judged\nmissing §1.3 §2.4 §8\nc3po gates code architecture 1.3 2.4 8\n"
	if code != 2 || out != want {
		t.Fatalf("code %d\n%s", code, out)
	}
}

func TestTallyBadLines(t *testing.T) {
	tempRepo(t)
	bad := strings.Replace(line("§1.2", "pass none"), receipt(architecture[2]), "zzzzzzzz", 1)
	for _, tc := range []struct{ section, line, want string }{
		{"§1.2", bad, "bad line 3: bad receipt: " + bad + "\n"},
		{"§1.3", line("§1.3", "fail nope.go:1"), "bad line 4: missing path nope.go: "},
		{"§1.4", line("§1.4", "fail a.go:9"), "bad line 5: a.go has no line 9: "},
		{"§1.5", line("§1.5", "pass"), "bad line 6: empty site: "},
		{"§1.6", line("§1.6", "maybe none"), "bad line 7: verdict is not pass or fail: "},
		{"§1.7", line("§1.7", "pass none because"), "bad line 8: extra words: "},
		{"§1.8", line("§1.8", "pass none files"), "bad line 9: extra words: "},
		{"§1.9", line("§1.9", "pass a.go"), "bad line 10: site a.go is not none or path:line: "},
	} {
		out, _, code := tally(t, fullSheet(map[string]string{tc.section: tc.line}))
		if code != 2 || !strings.HasPrefix(out, "139 of 140 judged\nmissing "+tc.section+"\n") || !strings.Contains(out, tc.want) {
			t.Fatalf("%s code %d\n%s", tc.line, code, out)
		}
	}
	out, _, code := tally(t, fullSheet(nil)+line("§8.3", "pass none")+"\n")
	if code != 2 || !strings.HasPrefix(out, "140 of 140 judged\nbad line 141: duplicate §8.3: ") {
		t.Fatalf("duplicate code %d\n%s", code, out)
	}
	out, _, code = tally(t, fullSheet(nil)+"§11 00000000 pass none\n")
	if code != 2 || !strings.Contains(out, `bad line 141: unknown section "§11"`) {
		t.Fatalf("unknown code %d\n%s", code, out)
	}
}

func TestTallyChangedFiles(t *testing.T) {
	tempRepo(t)
	write(t, "a.go", "package a\n")
	write(t, "new.go", "package a\n")
	out, _, code := tally(t, fullSheet(nil))
	if code != 2 || out != "140 of 140 judged\nunclaimed a.go\n" {
		t.Fatalf("unclaimed code %d\n%s", code, out)
	}
	claim := map[string]string{"§1.3": line("§1.3", "fail a.go:1 files a.go")}
	if out, _, code := tally(t, fullSheet(claim)); code != 0 || out != "140 of 140 judged\n" {
		t.Fatalf("claimed code %d\n%s", code, out)
	}
	claim["§3.2"] = line("§3.2", "fail none")
	if out, _, code := tally(t, fullSheet(claim)); code != 2 || out != "140 of 140 judged\nunfixed §3.2\n" {
		t.Fatalf("dirty unfixed code %d\n%s", code, out)
	}
	git(t, "rm", "-q", "sub/b.go")
	if out, _, _ := tally(t, fullSheet(claim)); !strings.Contains(out, "unclaimed sub/b.go\n") {
		t.Fatalf("a deleted file is not changed\n%s", out)
	}
	write(t, "sub/c.txt", "")
	t.Chdir("sub")
	out, _, code = call("code", "architecture", "tally", "c.txt")
	if code != 2 || !strings.Contains(out, "unclaimed ../a.go b.go\n") {
		t.Fatalf("paths are not relative to the current directory: code %d\n%s", code, out)
	}
}

func TestTallyRename(t *testing.T) {
	tempRepo(t)
	git(t, "mv", "a.go", "moved.go")
	out, _, code := tally(t, fullSheet(nil))
	if code != 2 || out != "140 of 140 judged\nunclaimed moved.go\n" {
		t.Fatalf("code %d\n%s", code, out)
	}
}

func TestTallyCleanTreeAllowsUnfixed(t *testing.T) {
	tempRepo(t)
	out, _, code := tally(t, fullSheet(map[string]string{"§3.2": line("§3.2", "fail none"), "§1.1": line("§1.1", "fail a.go:1")}))
	if code != 0 || out != "140 of 140 judged\nunfixed §1.1 §3.2\n" {
		t.Fatalf("code %d\n%s", code, out)
	}
}

func TestTallyRefusals(t *testing.T) {
	tempRepo(t)
	if out, errOut, code := call("code", "architecture", "tally", "gone.txt"); code != 2 || out != "" || !strings.Contains(errOut, "cannot read sheet gone.txt") {
		t.Fatalf("missing sheet code %d\n%s%s", code, out, errOut)
	}
	for _, args := range [][]string{{"tally"}, {"tally", "a", "b"}} {
		if _, errOut, code := call(append([]string{"code", "architecture"}, args...)...); code != 2 || !strings.Contains(errOut, "one sheet path") {
			t.Fatalf("%q code %d %s", args, code, errOut)
		}
	}
	notRepo := t.TempDir()
	t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(notRepo))
	t.Chdir(notRepo)
	write(t, "sheet.txt", fullSheet(nil))
	if out, errOut, code := call("code", "architecture", "tally", "sheet.txt"); code != 2 || out != "" || !strings.Contains(errOut, "git status failed") {
		t.Fatalf("no repo code %d\n%s%s", code, out, errOut)
	}
}
