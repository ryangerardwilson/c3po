package mess

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// worktree is one entry of git worktree list --porcelain.
type worktree struct {
	path     string
	branch   string // refs/heads/<name>, or "" when detached
	detached bool
	locked   bool
	bare     bool
}

// gitOut runs git in dir and returns its trimmed stdout.
func gitOut(dir string, args ...string) (string, error) {
	out, err := gitRaw(dir, args...)
	return strings.TrimSpace(out), err
}

func gitRaw(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return string(out), errors.New(msg)
		}
	}
	return string(out), err
}

// listWorktrees returns the main worktree first, then the linked ones.
func listWorktrees(dir string) ([]worktree, error) {
	out, err := gitOut(dir, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, fmt.Errorf("not a git repository: %s", dir)
	}
	var all []worktree
	for _, line := range strings.Split(out, "\n") {
		key, val, _ := strings.Cut(line, " ")
		if key == "worktree" {
			all = append(all, worktree{path: val})
			continue
		}
		if len(all) > 0 {
			all[len(all)-1].note(key, val)
		}
	}
	if len(all) == 0 {
		return nil, fmt.Errorf("no worktrees in %s", dir)
	}
	return all, nil
}

func (w *worktree) note(key, val string) {
	switch key {
	case "branch":
		w.branch = val
	case "detached":
		w.detached = true
	case "locked":
		w.locked = true
	case "bare":
		w.bare = true
	}
}

// adminDirs maps each linked worktree's path key to its git admin directory,
// <common>/worktrees/<name>.
func adminDirs(dir string) map[string]string {
	out := map[string]string{}
	common, err := gitOut(dir, "rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return out
	}
	entries, _ := os.ReadDir(filepath.Join(common, "worktrees"))
	for _, e := range entries {
		admin := filepath.Join(common, "worktrees", e.Name())
		raw, err := os.ReadFile(filepath.Join(admin, "gitdir"))
		if err != nil {
			continue
		}
		gitdir := strings.TrimSpace(string(raw))
		out[pathKey(filepath.Dir(gitdir))] = admin
	}
	return out
}

// pathKey resolves symlinks when it can, so two spellings of one path match.
func pathKey(p string) string {
	if real, err := filepath.EvalSymlinks(p); err == nil {
		return real
	}
	return filepath.Clean(p)
}

// markerName is the file tidy add writes in the worktree's admin directory.
const markerName = "c3po-tidy"

type marker struct {
	Session string
	Time    string
	Path    string
}

func readMarker(admin string) (marker, bool) {
	if admin == "" {
		return marker{}, false
	}
	raw, err := os.ReadFile(filepath.Join(admin, markerName))
	if err != nil {
		return marker{}, false
	}
	var m marker
	for _, line := range strings.Split(string(raw), "\n") {
		key, val, _ := strings.Cut(line, " ")
		switch key {
		case "session":
			m.Session = val
		case "time":
			m.Time = val
		case "path":
			m.Path = val
		}
	}
	return m, true
}

func writeMarker(admin, session, path string, now time.Time) error {
	body := fmt.Sprintf("session %s\ntime %s\npath %s\n", session, now.UTC().Format(time.RFC3339), path)
	return os.WriteFile(filepath.Join(admin, markerName), []byte(body), 0o644)
}
