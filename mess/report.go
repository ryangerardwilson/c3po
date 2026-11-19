package mess

import (
	"fmt"
	"io"
	"strings"
)

// Entry is one linked worktree in the report.
type Entry struct {
	Path    string
	Why     string
	Session string
}

// Report is the state of one repository's linked worktrees.
type Report struct {
	Root  string
	Clean []Entry
	Keep  []Entry
	Other []Entry
}

const (
	whyClean    = "pushed and clean"
	whyUnmarked = "unmarked linked worktree"
)

// Inspect reads the linked worktrees of the repository that contains dir.
// Nothing is changed.
func Inspect(dir string, p Probe) (Report, error) {
	all, err := listWorktrees(dir)
	if err != nil {
		return Report{}, err
	}
	rep := Report{Root: all[0].path}
	admins := adminDirs(dir)
	for _, wt := range all[1:] {
		if wt.bare {
			continue
		}
		m, ok := readMarker(admins[pathKey(wt.path)])
		if !ok {
			rep.Other = append(rep.Other, Entry{Path: wt.path, Why: whyUnmarked})
			continue
		}
		e := Entry{Path: wt.path, Session: m.Session}
		if e.Why = judge(wt, p); e.Why != "" {
			rep.Keep = append(rep.Keep, e)
			continue
		}
		e.Why = whyClean
		rep.Clean = append(rep.Clean, e)
	}
	return rep, nil
}

// Clean returns the clean marked worktrees of the repository that contains
// dir, or nil when dir is not in a repository.
func Clean(dir string) []string {
	rep, err := Inspect(dir, DefaultProbe())
	if err != nil {
		return nil
	}
	return rep.CleanPaths()
}

// CleanPaths lists the clean entries' paths.
func (r Report) CleanPaths() []string {
	var out []string
	for _, e := range r.Clean {
		out = append(out, e.Path)
	}
	return out
}

// Blocked lists what the agent cannot do, each as action and why.
func (r Report) Blocked() [][2]string {
	out := [][2]string{{"git worktree add", "use c3po mess tidy add"}}
	if len(r.Clean) > 0 {
		out = append(out, [2]string{"c3po mess tidy add", TidyFirst(r.CleanPaths())})
	}
	return out
}

// TidyFirst is the why of a blocked tidy add.
func TidyFirst(clean []string) string {
	return "run c3po mess tidy first  " + strings.Join(clean, " ")
}

// BlockedLine is the report line of one blocked action.
func BlockedLine(b [2]string) string {
	return "  " + b[0] + "  " + b[1]
}

// Write prints the report.
func (r Report) Write(w io.Writer) {
	blocked := r.Blocked()
	fmt.Fprintf(w, "mess %s\n", r.Root)
	fmt.Fprintf(w, "clean  %d\nkeep   %d\nother  %d\nblocked  %d\n", len(r.Clean), len(r.Keep), len(r.Other), len(blocked))
	if len(r.Clean) > 0 {
		fmt.Fprint(w, "\nclean\n")
		for _, e := range r.Clean {
			fmt.Fprintf(w, "  %s  %s  c3po mess tidy\n", e.Path, e.Why)
		}
	}
	writeGroup(w, "keep", r.Keep)
	writeGroup(w, "other", r.Other)
	fmt.Fprint(w, "\nblocked\n")
	for _, b := range blocked {
		fmt.Fprintln(w, BlockedLine(b))
	}
}

func writeGroup(w io.Writer, name string, entries []Entry) {
	if len(entries) == 0 {
		return
	}
	fmt.Fprintf(w, "\n%s\n", name)
	for _, e := range entries {
		fmt.Fprintf(w, "  %s  %s\n", e.Path, e.Why)
	}
}
