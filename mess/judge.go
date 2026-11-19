package mess

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Probe answers the questions that reach outside the worktree's files.
type Probe struct {
	// Remote returns the sha that branch points at on remote.
	Remote func(dir, remote, branch string) (string, error)
	// InUse reports whether a live process has its cwd inside dir.
	InUse func(dir string) (bool, error)
	// Remove runs git worktree remove without --force.
	Remove func(root, path string) error
}

// DefaultProbe asks git ls-remote and /proc, and removes with git.
func DefaultProbe() Probe {
	return Probe{Remote: lsRemote, InUse: procInUse, Remove: worktreeRemove}
}

// junk is the ignored first components a clean worktree may hold.
var junk = map[string]bool{
	"node_modules": true, ".next": true, "dist": true, "build": true, ".turbo": true,
	"coverage": true, ".venv": true, "__pycache__": true, ".ruff_cache": true, ".pytest_cache": true,
}

// judge returns the first failed condition, or "" when wt is eligible.
func judge(wt worktree, p Probe) string {
	if wt.locked {
		return "locked"
	}
	if _, err := os.Lstat(filepath.Join(wt.path, ".gitmodules")); err == nil {
		return "submodule"
	}
	if why := dirtyWhy(wt.path); why != "" {
		return why
	}
	if wt.detached || wt.branch == "" {
		return "detached"
	}
	if why := pushedWhy(wt, p); why != "" {
		return why
	}
	if busy, err := p.InUse(wt.path); err != nil || busy {
		return "in use"
	}
	return ""
}

func dirtyWhy(dir string) string {
	status, err := gitOut(dir, "status", "--porcelain")
	if err != nil || status != "" {
		return "dirty"
	}
	ignored, err := gitRaw(dir, "ls-files", "-z", "--others", "--ignored", "--exclude-standard", "--directory")
	if err != nil {
		return "dirty"
	}
	for _, name := range strings.Split(ignored, "\x00") {
		if name == "" {
			continue
		}
		first, _, _ := strings.Cut(name, "/")
		if !junk[first] {
			return "ignored file " + strings.TrimSuffix(name, "/")
		}
	}
	return ""
}

func pushedWhy(wt worktree, p Probe) string {
	remote, branch := upstream(wt)
	if remote == "." {
		return "remote is a local path"
	}
	url, err := gitOut(wt.path, "config", "--get", "remote."+remote+".url")
	if err != nil || url == "" {
		return "remote missing"
	}
	if localURL(url) {
		return "remote is a local path"
	}
	sha, err := p.Remote(wt.path, remote, branch)
	if err != nil || sha == "" {
		return "remote missing"
	}
	if _, err := gitOut(wt.path, "merge-base", "--is-ancestor", "HEAD", sha); err != nil {
		return "not contained"
	}
	return ""
}

// upstream is the branch's upstream remote and branch, else origin and the
// current branch.
func upstream(wt worktree) (string, string) {
	branch := strings.TrimPrefix(wt.branch, "refs/heads/")
	remote, _ := gitOut(wt.path, "config", "--get", "branch."+branch+".remote")
	if remote == "" {
		return "origin", branch
	}
	merge, _ := gitOut(wt.path, "config", "--get", "branch."+branch+".merge")
	if merge = strings.TrimPrefix(merge, "refs/heads/"); merge != "" {
		branch = merge
	}
	return remote, branch
}

// localURL reports whether a remote URL names a path on this machine.
func localURL(url string) bool {
	if strings.HasPrefix(url, "file:") {
		return true
	}
	if strings.Contains(url, "://") {
		return false
	}
	if strings.HasPrefix(url, "/") || strings.HasPrefix(url, ".") || strings.HasPrefix(url, "~") {
		return true
	}
	colon := strings.Index(url, ":")
	slash := strings.Index(url, "/")
	return colon < 0 || (slash >= 0 && slash < colon)
}

func lsRemote(dir, remote, branch string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "-C", dir, "ls-remote", "--exit-code", remote, "refs/heads/"+branch)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	sha, _, _ := strings.Cut(strings.TrimSpace(string(out)), "\t")
	if sha == "" {
		return "", errors.New("branch missing on remote")
	}
	return sha, nil
}

// procInUse looks for a process other than this one whose cwd is inside dir.
func procInUse(dir string) (bool, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return false, err
	}
	dir = pathKey(dir)
	self := strconv.Itoa(os.Getpid())
	seen := false
	for _, e := range entries {
		if _, err := strconv.Atoi(e.Name()); err != nil || e.Name() == self {
			continue
		}
		cwd, err := os.Readlink(filepath.Join("/proc", e.Name(), "cwd"))
		if err != nil {
			continue
		}
		seen = true
		if cwd == dir || strings.HasPrefix(cwd, dir+"/") {
			return true, nil
		}
	}
	if !seen {
		return false, errors.New("/proc shows no process directories")
	}
	return false, nil
}

func worktreeRemove(root, path string) error {
	_, err := gitOut(root, "worktree", "remove", path)
	return err
}
