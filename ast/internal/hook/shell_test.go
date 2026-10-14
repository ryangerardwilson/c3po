package hook

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/ryangerardwilson/c3po/ast/internal/config"
)

func TestShellWrites(t *testing.T) {
	home, _ := os.UserHomeDir()
	cases := []struct {
		cmd  string
		want []string
	}{
		{"ls -la", nil},
		{"rgw-ast status --json 2>&1", nil},
		{"go test ./... 2>/dev/null | head >&2", nil},
		{"grep -n 'a > b; rm x' file", nil},
		{"echo x > out.txt", []string{"/w/out.txt"}},
		{"echo x >> /tmp/log 2>&1", []string{"/tmp/log"}},
		{"cmd &> all.log", []string{"/w/all.log"}},
		{"echo hi # > not-a-write", nil},
		{"rm -rf build dist", []string{"/w/build", "/w/dist"}},
		{"cp -r a.go /tmp/b.go", []string{"/tmp/b.go"}},
		{"cp -t /tmp/out a b", []string{"/tmp/out"}},
		{"rsync -a src/ /tmp/dst/ --exclude x.sh", []string{"/tmp/dst"}},
		{"install -m 0644 a /tmp/b", []string{"/tmp/b"}},
		{"mv a b", []string{"/w/a", "/w/b"}},
		{"ls | tee copy.txt", []string{"/w/copy.txt"}},
		{"chmod +x run.sh", []string{"/w/run.sh"}},
		{"sed -n '1,5p' f.go", nil},
		{"sed -i 's/a/b/' f.go", []string{"/w/f.go"}},
		{"sed -i.bak -e 's/a/b/' f.go", []string{"/w/f.go"}},
		{"perl -pi -e 's/a/b/' f.go", []string{"/w/f.go"}},
		{"dd if=/dev/zero of=disk.img", []string{"/w/disk.img"}},
		{"truncate -s 0 log.txt", []string{"/w/log.txt"}},
		{"cd /tmp && rm x", []string{"/tmp/x"}},
		{"cd sub; touch y", []string{"/w/sub/y"}},
		{"cd && touch z", []string{filepath.Join(home, "z")}},
		{"touch ~/note $HOME/a ${HOME}/b", []string{filepath.Join(home, "note"), filepath.Join(home, "a"), filepath.Join(home, "b")}},
		{`rm "$TARGET"`, []string{"/w"}},
		{"S=/tmp/sp; rm -rf $S/x && echo y > \"${S}/z\"", []string{"/tmp/sp/x", "/tmp/sp/z"}},
		{"export OUT=$HOME/o; touch $OUT", []string{filepath.Join(home, "o")}},
		{"D=$(pwd); rm $D/x", []string{"/w"}},
		{"rm ${S:-x}", []string{"/w"}},
		{"bash -c 'rm a.go'", []string{"/w/a.go"}},
		{"sh -lc \"echo x > f\"", []string{"/w/f"}},
		{"eval rm a.go", []string{"/w/a.go"}},
		{"echo $(rm a.go)", []string{"/w/a.go"}},
		{"echo `touch b`", []string{"/w/b"}},
		{"diff <(sort a) <(sort b)", nil},
		{"FOO=1 BAR=\"x y\" rm z", []string{"/w/z"}},
		{"sudo -u root rm /etc/x", []string{"/etc/x"}},
		{"timeout 5 rm q", []string{"/w/q"}},
		{"find . -name '*.go' -print", nil},
		{"find src -name '*.tmp' -delete", []string{"/w/src"}},
		{"find . -name x -exec rm {} \\;", []string{"/w"}},
		{"find . -name x -exec grep y {} +", nil},
		{"git ls-files | xargs rm", []string{"/w"}},
		{"git ls-files | xargs grep x", nil},
		{"git status && git diff", nil},
		{"git -C /r apply p.diff", []string{"/r"}},
		{"git checkout -- a.go", []string{"/w"}},
		{"git checkout main", nil},
		{"patch -p1", []string{"/w"}},
		{"rgw-ast create f --expect-absent --stdin <<'EOF'\nrm -rf / > x\nEOF", nil},
		{"cat <<EOF > out\nhello $(date)\nEOF", []string{"/w", "/w/out"}},
		{"cat <<EOF\nplain $USER\nEOF", nil},
		{"git commit -m \"$(cat <<'EOF'\nDon't (break) \"this\"\nEOF\n)\"", nil},
		{"for f in *.go; do rm $f; done", []string{"/w"}},
		{"echo 'unterminated", []string{"/w"}},
		{"rgw-ast status; printf x > f", []string{"/w/f"}},
	}
	for _, c := range cases {
		t.Run(c.cmd, func(t *testing.T) {
			got := shellWrites(c.cmd, "/w")
			if !reflect.DeepEqual(got, c.want) {
				t.Fatalf("shellWrites(%q) = %v, want %v", c.cmd, got, c.want)
			}
		})
	}
}

// enforcedRepo returns a git worktree that is enforced at threshold 1 and
// ignores ignored/.
func enforcedRepo(t *testing.T) (string, config.Config) {
	t.Helper()
	repo := t.TempDir()
	if out, err := exec.Command("git", "-C", repo, "init", "-q").CombinedOutput(); err != nil {
		t.Skipf("git unavailable: %v %s", err, out)
	}
	for name, body := range map[string]string{
		"a.go":       "package a\n//1\n//2\n",
		".gitignore": "ignored/\n",
	} {
		if err := os.WriteFile(filepath.Join(repo, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cfg := config.Default()
	cfg.ThresholdLOC = 1
	cfg.Enforcement.Mode = "auto"
	cfg.Cache.Dir = t.TempDir()
	return repo, cfg
}

func decide(t *testing.T, cfg config.Config, req Request) string {
	t.Helper()
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := Run(bytes.NewReader(data), &out, cfg, "/", ""); err != nil {
		t.Fatal(err)
	}
	var dec Decision
	if err := json.Unmarshal(out.Bytes(), &dec); err != nil {
		t.Fatal(err, out.String())
	}
	return dec.PermissionDecision
}

func TestHookJudgesTargetNotSessionDirectory(t *testing.T) {
	repo, cfg := enforcedRepo(t)
	outside := t.TempDir()
	cases := []struct {
		name string
		req  Request
		want string
	}{
		{"write outside every repository", Request{Cwd: repo, ToolName: "Write", ToolInput: map[string]any{"file_path": filepath.Join(outside, "n.txt")}}, "allow"},
		{"edit into enforced repo from elsewhere", Request{Cwd: outside, ToolName: "Edit", ToolInput: map[string]any{"file_path": filepath.Join(repo, "a.go")}}, "deny"},
		{"relative write resolves against event cwd", Request{Cwd: repo, ToolName: "Write", ToolInput: map[string]any{"file_path": "a.go"}}, "deny"},
		{"write of a new file is a create", Request{Cwd: repo, ToolName: "Write", ToolInput: map[string]any{"file_path": "b.go"}}, "allow"},
		{"edit of a new file is denied", Request{Cwd: repo, ToolName: "Edit", ToolInput: map[string]any{"file_path": "b.go"}}, "deny"},
		{"ignored target", Request{Cwd: repo, ToolName: "Write", ToolInput: map[string]any{"file_path": filepath.Join(repo, "ignored", "x")}}, "allow"},
		{"notebook edit", Request{Cwd: outside, ToolName: "NotebookEdit", ToolInput: map[string]any{"notebook_path": filepath.Join(repo, "n.ipynb")}}, "deny"},
		{"descriptor duplication", Request{Cwd: repo, ToolName: "Bash", ToolInput: map[string]any{"command": "rgw-ast status --json 2>&1"}}, "allow"},
		{"heredoc into rgw-ast", Request{Cwd: repo, ToolName: "Bash", ToolInput: map[string]any{"command": "rgw-ast create f --expect-absent --stdin <<'EOF'\nx > y\nEOF"}}, "allow"},
		{"copy out of repo", Request{Cwd: repo, ToolName: "Bash", ToolInput: map[string]any{"command": "cp a.go " + filepath.Join(outside, "a.go")}}, "allow"},
		{"redirect into repo", Request{Cwd: repo, ToolName: "Bash", ToolInput: map[string]any{"command": "echo x > a.go"}}, "deny"},
		{"redirect into repo from elsewhere", Request{Cwd: outside, ToolName: "Bash", ToolInput: map[string]any{"command": "cd " + repo + " && echo x > a.go"}}, "deny"},
		{"nested shell write", Request{Cwd: repo, ToolName: "Bash", ToolInput: map[string]any{"command": "bash -c 'rm a.go'"}}, "deny"},
		{"read-only pipeline", Request{Cwd: repo, ToolName: "Bash", ToolInput: map[string]any{"command": "rgw-ast read a.go --lines 1-5 | head -n 2"}}, "allow"},
		{"write to ignored path by shell", Request{Cwd: repo, ToolName: "Bash", ToolInput: map[string]any{"command": "mkdir -p ignored && echo x > ignored/log"}}, "allow"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := decide(t, cfg, c.req); got != c.want {
				t.Fatalf("got %s, want %s", got, c.want)
			}
		})
	}
}
