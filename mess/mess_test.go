package mess

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	for k, v := range map[string]string{
		"GIT_CONFIG_GLOBAL":   os.DevNull,
		"GIT_CONFIG_NOSYSTEM": "1",
		"GIT_AUTHOR_NAME":     "t", "GIT_AUTHOR_EMAIL": "t@t",
		"GIT_COMMITTER_NAME": "t", "GIT_COMMITTER_EMAIL": "t@t",
		"C3PO_SESSION_ID": "",
	} {
		os.Setenv(k, v)
	}
	os.Exit(m.Run())
}

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// newRepo is a repository with one commit, an ignore file, and an https origin.
func newRepo(t *testing.T) string {
	t.Helper()
	base := t.TempDir()
	repo := filepath.Join(base, "repo")
	git(t, base, "init", "-q", "-b", "main", repo)
	write(t, filepath.Join(repo, ".gitignore"), "node_modules/\n.env\n")
	git(t, repo, "add", ".")
	git(t, repo, "commit", "-qm", "init")
	git(t, repo, "remote", "add", "origin", "https://example.invalid/repo.git")
	return repo
}

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// stub says every remote branch contains HEAD and nothing is in use.
func stub() Probe {
	return Probe{
		Remote: func(dir, remote, branch string) (string, error) {
			out, err := exec.Command("git", "-C", dir, "rev-parse", "HEAD").Output()
			return strings.TrimSpace(string(out)), err
		},
		InUse:  func(string) (bool, error) { return false, nil },
		Remove: worktreeRemove,
	}
}

func call(t *testing.T, cwd, stdin string, p Probe, args ...string) (string, int) {
	t.Helper()
	var out, errb bytes.Buffer
	code := run(args, cwd, strings.NewReader(stdin), &out, &errb, p)
	return out.String() + errb.String(), code
}

func addMarked(t *testing.T, repo, name, session string) string {
	t.Helper()
	path := filepath.Join(filepath.Dir(repo), name)
	offline := stub()
	offline.Remote = func(string, string, string) (string, error) { return "", errors.New("offline") }
	out, code := call(t, repo, "", offline, "tidy", "add", path, "-b", name, "--session", session)
	if code != 0 {
		t.Fatalf("tidy add %s: %d\n%s", name, code, out)
	}
	return path
}

func exists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

func TestEmptyRepoReport(t *testing.T) {
	repo := newRepo(t)
	out, code := call(t, repo, "", stub())
	want := "mess " + repo + "\nclean  0\nkeep   0\nother  0\nblocked  1\n\nblocked\n  git worktree add  use c3po mess tidy add\n"
	if code != 0 || out != want {
		t.Fatalf("code %d\n%q\nwant\n%q", code, out, want)
	}
}

func TestRunFromCwd(t *testing.T) {
	repo := newRepo(t)
	t.Chdir(repo)
	var out, errb bytes.Buffer
	if code := Run(nil, &out, &errb); code != 0 || !strings.Contains(out.String(), "clean  0\nkeep   0\nother  0\n") {
		t.Fatalf("code %d\n%s%s", code, out.String(), errb.String())
	}
}

func TestTidyAddWritesMarker(t *testing.T) {
	repo := newRepo(t)
	path := addMarked(t, repo, "feat", "s1")
	raw, err := os.ReadFile(filepath.Join(repo, ".git", "worktrees", "feat", markerName))
	if err != nil {
		t.Fatal(err)
	}
	body := string(raw)
	for _, want := range []string{"session s1\n", "time ", "path " + path + "\n"} {
		if !strings.Contains(body, want) {
			t.Fatalf("marker missing %q\n%s", want, body)
		}
	}
	if got := git(t, path, "branch", "--show-current"); got != "feat" {
		t.Fatalf("branch %q", got)
	}
}

func TestSessionFromEnv(t *testing.T) {
	repo := newRepo(t)
	t.Setenv("C3PO_SESSION_ID", "env-s")
	path := filepath.Join(filepath.Dir(repo), "e")
	if out, code := call(t, repo, "", stub(), "tidy", "add", path); code != 0 {
		t.Fatalf("%d\n%s", code, out)
	}
	m, ok := readMarker(filepath.Join(repo, ".git", "worktrees", "e"))
	if !ok || m.Session != "env-s" {
		t.Fatalf("marker %+v %v", m, ok)
	}
}

func TestKeepReasons(t *testing.T) {
	repo := newRepo(t)
	dirty := addMarked(t, repo, "dirty", "s")
	write(t, filepath.Join(dirty, "a.txt"), "x")
	env := addMarked(t, repo, "env", "s")
	write(t, filepath.Join(env, ".env"), "SECRET=1")
	local := addMarked(t, repo, "local", "s")
	git(t, local, "config", "branch.local.remote", "bare")
	git(t, local, "config", "branch.local.merge", "refs/heads/local")
	git(t, repo, "remote", "add", "bare", filepath.Join(filepath.Dir(repo), "bare.git"))

	out, code := call(t, repo, "", stub())
	for _, want := range []string{
		"clean  0\nkeep   3\n",
		"  " + dirty + "  dirty\n",
		"  " + env + "  ignored file .env\n",
		"  " + local + "  remote is a local path\n",
	} {
		if code != 0 || !strings.Contains(out, want) {
			t.Fatalf("code %d missing %q\n%s", code, want, out)
		}
	}
	if strings.Contains(out, "c3po mess tidy add  run") || strings.Contains(out, "reset") || strings.Contains(out, "stash") {
		t.Fatalf("keep added a block\n%s", out)
	}
	out, _ = call(t, repo, "", stub(), "tidy")
	if strings.Contains(out, "removed") || !exists(dirty) || !exists(env) || !exists(local) {
		t.Fatalf("tidy removed a keep entry\n%s", out)
	}
}

func TestJunkOnlyIsCleanAndRemoved(t *testing.T) {
	repo := newRepo(t)
	path := addMarked(t, repo, "junk", "s")
	write(t, filepath.Join(path, "node_modules", "x", "index.js"), "x")
	other := filepath.Join(filepath.Dir(repo), "other")
	git(t, repo, "worktree", "add", "-q", other)

	out, code := call(t, repo, "", stub())
	for _, want := range []string{
		"clean  1\nkeep   0\nother  1\nblocked  2\n",
		"  " + path + "  pushed and clean  c3po mess tidy\n",
		"  " + other + "  unmarked linked worktree\n",
		"  c3po mess tidy add  run c3po mess tidy first  " + path + "\n",
	} {
		if code != 0 || !strings.Contains(out, want) {
			t.Fatalf("code %d missing %q\n%s", code, want, out)
		}
	}
	out, code = call(t, repo, "", stub(), "tidy")
	if code != 0 || !strings.HasPrefix(out, "removed "+path+"\n") || exists(path) {
		t.Fatalf("code %d\n%s", code, out)
	}
	if !exists(other) || !exists(repo) || !strings.Contains(out, "clean  0\n") {
		t.Fatalf("main or other removed\n%s", out)
	}
}

func TestTidyAddBlockedByClean(t *testing.T) {
	repo := newRepo(t)
	addMarked(t, repo, "done", "s")
	next := filepath.Join(filepath.Dir(repo), "next")
	out, code := call(t, repo, "", stub(), "tidy", "add", next, "-b", "next")
	if code != 1 || exists(next) || !strings.Contains(out, "c3po mess tidy add  run c3po mess tidy first") {
		t.Fatalf("code %d exists %v\n%s", code, exists(next), out)
	}
}

func TestFailedRemoveIsNotRemoved(t *testing.T) {
	repo := newRepo(t)
	path := addMarked(t, repo, "stuck", "s")
	p := stub()
	p.Remove = func(root, path string) error { return errors.New("refused") }
	out, code := call(t, repo, "", p, "tidy")
	if code != 1 || strings.Contains(out, "removed "+path) || !exists(path) {
		t.Fatalf("code %d\n%s", code, out)
	}
	if strings.Contains(out, path+"  pushed and clean") || !strings.Contains(out, "clean  0\n") {
		t.Fatalf("failed path listed as clean\n%s", out)
	}
}

func TestHookReapsOnlyItsSession(t *testing.T) {
	repo := newRepo(t)
	one := addMarked(t, repo, "one", "s1")
	two := addMarked(t, repo, "two", "s2")
	out, code := call(t, repo, `{"session_id":"s1","cwd":"`+repo+`"}`, stub(), "tidy", "--hook")
	if code != 0 || exists(one) || !exists(two) {
		t.Fatalf("code %d one %v two %v\n%s", code, exists(one), exists(two), out)
	}
	out, code = call(t, repo, "", stub(), "tidy", "--session", "s2")
	if code != 0 || exists(two) {
		t.Fatalf("session tidy %d\n%s", code, out)
	}
}

func TestHookBadStdin(t *testing.T) {
	repo := newRepo(t)
	for _, in := range []string{"", "not json", `{"session_id":"s"}`, `{"session_id":"s","cwd":"/nonexistent/x"}`} {
		out, code := call(t, repo, in, stub(), "tidy", "--hook")
		if code != 0 || out != "" {
			t.Fatalf("%q: code %d\n%s", in, code, out)
		}
	}
	p := stub()
	p.Remove = func(string, string) error { return errors.New("refused") }
	addMarked(t, repo, "x", "s")
	if _, code := call(t, repo, `{"session_id":"s","cwd":"`+repo+`"}`, p, "tidy", "--hook"); code != 0 {
		t.Fatalf("hook failure exit %d", code)
	}
}

func TestLocalURL(t *testing.T) {
	for url, want := range map[string]bool{
		"/srv/x.git": true, "../x": true, "file:///x": true, "x": true, "~/x": true,
		"https://h/x.git": false, "git@github.com:a/b.git": false, "ssh://h/x": false,
	} {
		if got := localURL(url); got != want {
			t.Fatalf("%s: %v", url, got)
		}
	}
}

func TestHelpAndUsage(t *testing.T) {
	out, code := call(t, t.TempDir(), "", stub(), "help")
	if code != 0 || !strings.Contains(out, "c3po mess tidy add") {
		t.Fatalf("%d\n%s", code, out)
	}
	if _, code := call(t, t.TempDir(), "", stub(), "nope"); code != 2 {
		t.Fatalf("unknown exit %d", code)
	}
	if _, code := call(t, t.TempDir(), "", stub()); code != 1 {
		t.Fatalf("outside a repo exit %d", code)
	}
}
