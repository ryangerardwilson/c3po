package dirt

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// skipped names the directories the walk does not enter below the scan root.
var skipped = map[string]bool{
	"node_modules": true, ".venv": true, "venv": true, ".cache": true,
	"__pycache__": true, ".npm": true, ".cargo": true, "site-packages": true,
}

// gitScope is the inherited environment that would point git at another repository.
var gitScope = map[string]bool{
	"GIT_DIR": true, "GIT_WORK_TREE": true, "GIT_INDEX_FILE": true, "GIT_COMMON_DIR": true,
	"GIT_OBJECT_DIRECTORY": true, "GIT_ALTERNATE_OBJECT_DIRECTORIES": true, "GIT_PREFIX": true,
}

const workers = 8

type repo struct {
	path string
	// within is the scan path relative to path when the repository encloses
	// the scan path. It is empty when the whole repository is reported.
	within string
}

type verdict struct {
	tally tally
	err   error
}

type tally struct {
	modified  int
	added     int
	deleted   int
	untracked int
}

func (t tally) empty() bool {
	return t.modified == 0 && t.added == 0 && t.deleted == 0 && t.untracked == 0
}

func (t tally) String() string {
	var parts []string
	if t.modified > 0 {
		parts = append(parts, fmt.Sprintf("M%d", t.modified))
	}
	if t.added > 0 {
		parts = append(parts, fmt.Sprintf("+%d", t.added))
	}
	if t.deleted > 0 {
		parts = append(parts, fmt.Sprintf("-%d", t.deleted))
	}
	if t.untracked > 0 {
		parts = append(parts, fmt.Sprintf("??%d", t.untracked))
	}
	return strings.Join(parts, " ")
}

func report(root string, stdout, stderr io.Writer) int {
	repos := find(root)
	verdicts := judgeAll(repos)
	code := 0
	for i, r := range repos {
		v := verdicts[i]
		if v.err != nil {
			fmt.Fprintf(stderr, "dirt: %s: %s\n", r.path, v.err)
			code = 1
			continue
		}
		if !v.tally.empty() {
			fmt.Fprintf(stdout, "%s : %s\n", shortPath(r.path), v.tally)
		}
	}
	return code
}

// find returns the repositories under root, and the one that encloses it,
// sorted by path.
func find(root string) []repo {
	var repos []repo
	if top, ok := enclosing(root); ok {
		rel, err := filepath.Rel(top, root)
		if err == nil {
			repos = append(repos, repo{path: top, within: filepath.ToSlash(rel)})
		}
	}
	walk(root, &repos)
	sort.Slice(repos, func(i, j int) bool { return repos[i].path < repos[j].path })
	return repos
}

func walk(dir string, repos *[]repo) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		name := e.Name()
		if name == ".git" {
			if e.IsDir() || e.Type().IsRegular() {
				*repos = append(*repos, repo{path: dir})
			}
			continue
		}
		if e.IsDir() && !skipped[name] && !homeLocal(dir, name) {
			walk(filepath.Join(dir, name), repos)
		}
	}
}

// homeLocal is the user's ~/.local. A parent walk does not enter it. The
// directory is still scanned when it is the scan root, because the walk
// starts there. Any other directory named .local is entered.
func homeLocal(parent, name string) bool {
	if name != ".local" {
		return false
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return false
	}
	return filepath.Clean(filepath.Join(parent, name)) == filepath.Clean(filepath.Join(home, ".local"))
}

// enclosing returns the nearest ancestor of root that holds .git, when root
// is not itself a repository root.
func enclosing(root string) (string, bool) {
	if hasGit(root) {
		return "", false
	}
	for dir := filepath.Dir(root); ; dir = filepath.Dir(dir) {
		if hasGit(dir) {
			return dir, true
		}
		if dir == filepath.Dir(dir) {
			return "", false
		}
	}
}

func hasGit(dir string) bool {
	info, err := os.Lstat(filepath.Join(dir, ".git"))
	return err == nil && (info.IsDir() || info.Mode().IsRegular())
}

func judgeAll(repos []repo) []verdict {
	out := make([]verdict, len(repos))
	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup
	for i, r := range repos {
		sem <- struct{}{}
		wg.Go(func() {
			defer func() { <-sem }()
			out[i] = judge(r)
		})
	}
	wg.Wait()
	return out
}

// judge returns how much dirt r holds. An empty tally means r is clean.
func judge(r repo) verdict {
	status, err := gitStatus(r.path)
	if err != nil {
		return verdict{err: err}
	}
	return verdict{tally: dirtTally(status, r.within)}
}

// dirtTally counts porcelain lines that are not the ## branch line and, when
// within is set, name a path under it.
func dirtTally(status, within string) tally {
	var t tally
	for _, line := range strings.Split(status, "\n") {
		if line == "" || strings.HasPrefix(line, "##") {
			continue
		}
		if within != "" && !inside(line, within) {
			continue
		}
		switch kind(line) {
		case 'M':
			t.modified++
		case '+':
			t.added++
		case '-':
			t.deleted++
		case '?':
			t.untracked++
		}
	}
	return t
}

// kind is the one mark a porcelain line adds. 0 means the line is not dirt.
func kind(line string) byte {
	if len(line) < 2 {
		return 0
	}
	x, y := line[0], line[1]
	switch {
	case x == '?' || y == '?':
		return '?'
	case x == '!' || y == '!':
		return 0
	case x == 'D' || y == 'D':
		return '-'
	case x == 'A' || y == 'A' || x == 'C' || y == 'C':
		return '+'
	default:
		return 'M'
	}
}

// shortPath writes a repository under the home directory as ~/..., and that
// directory itself as ~. Any other path stays absolute.
func shortPath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return path
	}
	home = filepath.Clean(home)
	path = filepath.Clean(path)
	if path == home {
		return "~"
	}
	rel, err := filepath.Rel(home, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return path
	}
	return "~/" + filepath.ToSlash(rel)
}

func gitStatus(dir string) (string, error) {
	cmd := exec.Command("git", "--no-optional-locks", "-C", dir, "status", "--porcelain=v1", "-b", "--untracked-files=normal")
	cmd.Env = gitEnv()
	var errb bytes.Buffer
	cmd.Stderr = &errb
	out, err := cmd.Output()
	if err != nil {
		msg, _, _ := strings.Cut(strings.TrimSpace(errb.String()), "\n")
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("git status failed: %s", msg)
	}
	return string(out), nil
}

func gitEnv() []string {
	var env []string
	for _, kv := range os.Environ() {
		name, _, _ := strings.Cut(kv, "=")
		if !gitScope[name] {
			env = append(env, kv)
		}
	}
	return env
}

// inside reports whether a porcelain line names a path under within. A
// rename counts when either path is under it, and an untracked directory
// counts when it holds within.
func inside(line, within string) bool {
	if len(line) < 4 {
		return false
	}
	paths := []string{line[3:]}
	if strings.ContainsAny(line[:2], "RC") {
		if from, to, ok := strings.Cut(line[3:], " -> "); ok {
			paths = []string{from, to}
		}
	}
	for _, p := range paths {
		if covers(unquote(p), within) {
			return true
		}
	}
	return false
}

func covers(path, within string) bool {
	if path == within || strings.HasPrefix(path, within+"/") {
		return true
	}
	return strings.HasSuffix(path, "/") && strings.HasPrefix(within+"/", path)
}

func unquote(path string) string {
	if strings.HasPrefix(path, `"`) {
		if s, err := strconv.Unquote(path); err == nil {
			return s
		}
	}
	return path
}
