package measure

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/ryangerardwilson/c3po/ast/internal/config"
)

// Skip reasons reported by Scope.
const (
	SkipNamed      = "skipped directory name"
	SkipNestedRepo = "nested git repository"
	SkipExcluded   = "exclude glob"
	SkipGitIgnored = "gitignored"
	SkipSelfIgnore = "directory ignores itself"
	SkipNotIncl    = "no include glob"
)

// Scope is the set of files every walk sees: include/exclude globs, skipped
// directory names, nested repositories, the root .gitignore, and directories
// that ignore themselves. measure, map, show, search, explain, and trace all
// walk through it so they agree on what exists.
type Scope struct {
	root    string
	include []glob
	exclude []glob
	prose   []glob
	gi      *gitIgnore
}

// NewScope builds the walk scope for root under cfg.
func NewScope(root string, cfg config.Config) *Scope {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		rootAbs = root
	}
	return &Scope{
		root:    rootAbs,
		include: compileGlobs(cfg.Include),
		exclude: compileGlobs(cfg.Exclude),
		prose:   compileGlobs(cfg.Prose),
		gi:      loadGitIgnore(rootAbs),
	}
}

// Root is the absolute workspace root.
func (s *Scope) Root() string { return s.root }

// GitIgnoreActive reports whether a root ignore file was loaded.
func (s *Scope) GitIgnoreActive() bool { return s.gi != nil }

// IsProse reports whether rel matches a prose glob.
func (s *Scope) IsProse(rel string) bool { return matchAny(s.prose, filepath.ToSlash(rel)) }

func (s *Scope) rel(abs string) string {
	rel, err := filepath.Rel(s.root, abs)
	if err != nil {
		return filepath.ToSlash(abs)
	}
	return filepath.ToSlash(rel)
}

// SkipDir returns why a walk skips directory abs, or "" when it descends.
// The root itself is never skipped.
func (s *Scope) SkipDir(abs string) string {
	if abs == s.root {
		return ""
	}
	rel := s.rel(abs)
	switch {
	case SkipDirName(filepath.Base(abs)):
		return SkipNamed
	case isNestedGitRoot(abs, s.root):
		return SkipNestedRepo
	case matchAny(s.exclude, rel+"/") || matchAny(s.exclude, rel):
		return SkipExcluded
	case s.gi != nil && s.gi.ignored(rel, true):
		return SkipGitIgnored
	case selfIgnored(abs):
		return SkipSelfIgnore
	}
	return ""
}

// SkipFile returns why a walk skips regular file abs, or "" when it is seen.
func (s *Scope) SkipFile(abs string) string {
	rel := s.rel(abs)
	switch {
	case matchAny(s.exclude, rel):
		return SkipExcluded
	case s.gi != nil && s.gi.ignored(rel, false):
		return SkipGitIgnored
	case !matchAny(s.include, rel):
		return SkipNotIncl
	}
	return ""
}

// SkipAncestor returns why some directory between the root and abs (abs
// excluded) is skipped, and that directory, or "" when none is.
func (s *Scope) SkipAncestor(abs string) (string, string) {
	rel := s.rel(abs)
	if rel == "." || strings.HasPrefix(rel, "../") {
		return "", ""
	}
	dir := s.root
	parts := strings.Split(rel, "/")
	for _, p := range parts[:len(parts)-1] {
		dir = filepath.Join(dir, p)
		if why := s.SkipDir(dir); why != "" {
			return why, dir
		}
	}
	return "", ""
}

// Walk calls fn with the absolute and root-relative path of every file in
// scope under start, in lexical order. depth < 0 is unlimited; depth 0 is
// only start's own files. fn may return filepath.SkipAll. Walk returns how
// many nested repositories it skipped.
func (s *Scope) Walk(start string, depth int, fn func(abs, rel string) error) (int, error) {
	nested := 0
	err := filepath.WalkDir(start, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			if d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			if path == start {
				return nil
			}
			if why := s.SkipDir(path); why != "" {
				if why == SkipNestedRepo {
					nested++
				}
				return filepath.SkipDir
			}
			if depth >= 0 {
				if rel, err := filepath.Rel(start, path); err == nil && strings.Count(filepath.ToSlash(rel), "/")+1 > depth {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if !d.Type().IsRegular() || s.SkipFile(path) != "" {
			return nil
		}
		return fn(path, s.rel(path))
	})
	return nested, err
}

// selfIgnored reports whether dir holds a .gitignore whose only rule is "*",
// the marker tool caches (ruff, pytest, mypy) write into their own directory.
func selfIgnored(dir string) bool {
	f, err := os.Open(filepath.Join(dir, ".gitignore"))
	if err != nil {
		return false
	}
	defer f.Close()
	var rules []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		rules = append(rules, line)
		if len(rules) > 1 {
			return false
		}
	}
	return len(rules) == 1 && rules[0] == "*"
}
