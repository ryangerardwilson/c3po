package mess

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// tidy removes the clean worktrees, only those of session when it is set,
// then prints what remains.
func tidy(cwd string, session *string, stdout, stderr io.Writer, p Probe) int {
	rep, err := Inspect(cwd, p)
	if err != nil {
		fmt.Fprintf(stderr, "mess: %s\n", err)
		return 1
	}
	failed := map[string]bool{}
	for _, e := range rep.Clean {
		if session != nil && e.Session != *session {
			continue
		}
		err := p.Remove(rep.Root, e.Path)
		if _, statErr := os.Lstat(e.Path); statErr == nil {
			failed[e.Path] = true
			fmt.Fprintf(stderr, "mess: %s still exists after git worktree remove: %v\n", e.Path, err)
			continue
		}
		fmt.Fprintf(stdout, "removed %s\n", e.Path)
	}
	rest, err := Inspect(rep.Root, p)
	if err != nil {
		fmt.Fprintf(stderr, "mess: %s\n", err)
		return 1
	}
	rest.demote(failed)
	rest.Write(stdout)
	if len(failed) > 0 {
		return 1
	}
	return 0
}

// demote moves a clean entry whose removal failed to keep.
func (r *Report) demote(failed map[string]bool) {
	if len(failed) == 0 {
		return
	}
	var clean []Entry
	for _, e := range r.Clean {
		if failed[e.Path] {
			e.Why = "remove failed"
			r.Keep = append(r.Keep, e)
			continue
		}
		clean = append(clean, e)
	}
	r.Clean = clean
}

// add creates one marked linked worktree unless a clean entry exists.
func add(cwd string, o tidyOpts, stdout, stderr io.Writer, p Probe) int {
	rep, err := Inspect(cwd, p)
	if err != nil {
		fmt.Fprintf(stderr, "mess: %s\n", err)
		return 1
	}
	if len(rep.Clean) > 0 {
		fmt.Fprintln(stderr, BlockedLine(rep.Blocked()[1]))
		return 1
	}
	path := o.path
	if !filepath.IsAbs(path) {
		path = filepath.Join(cwd, path)
	}
	path = filepath.Clean(path)
	args := []string{"-C", cwd, "worktree", "add"}
	if o.branch != "" {
		args = append(args, "-b", o.branch)
	}
	cmd := exec.Command("git", append(args, path)...)
	cmd.Stdout, cmd.Stderr = stderr, stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(stderr, "mess: git worktree add: %s\n", err)
		return 1
	}
	return mark(cwd, path, sessionOf(o), stdout, stderr)
}

func sessionOf(o tidyOpts) string {
	if o.session != nil {
		return *o.session
	}
	return os.Getenv("C3PO_SESSION_ID")
}

func mark(cwd, path, session string, stdout, stderr io.Writer) int {
	admin := adminDirs(cwd)[pathKey(path)]
	if admin == "" {
		fmt.Fprintf(stderr, "mess: no git admin directory for %s\n", path)
		return 1
	}
	if err := writeMarker(admin, session, path, time.Now()); err != nil {
		fmt.Fprintf(stderr, "mess: marker: %s\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "added %s\n", path)
	return 0
}

// hookTidy reads a SessionEnd event and tidies that session. It always
// exits 0 so the host turn is never blocked.
func hookTidy(stdin io.Reader, stdout io.Writer, p Probe) int {
	var ev struct {
		SessionID string `json:"session_id"`
		Cwd       string `json:"cwd"`
	}
	data, err := io.ReadAll(stdin)
	if err != nil || json.Unmarshal(data, &ev) != nil || ev.SessionID == "" || ev.Cwd == "" {
		return 0
	}
	if _, err := listWorktrees(ev.Cwd); err != nil {
		return 0
	}
	tidy(ev.Cwd, &ev.SessionID, stdout, io.Discard, p)
	return 0
}
