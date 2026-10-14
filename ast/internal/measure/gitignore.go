package measure

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// gitIgnore is a minimal .gitignore matcher for measure walks.
type gitIgnore struct {
	// patterns are applied in order; last match wins for negation.
	rules []giRule
}

type giRule struct {
	negated bool
	dirOnly bool
	pattern string
}

func loadGitIgnore(root string) *gitIgnore {
	var rules []giRule
	for _, name := range []string{
		filepath.Join(root, ".gitignore"),
		filepath.Join(root, ".git", "info", "exclude"),
	} {
		f, err := os.Open(name)
		if err != nil {
			continue
		}
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			neg := false
			if strings.HasPrefix(line, "!") {
				neg = true
				line = line[1:]
			}
			dirOnly := strings.HasSuffix(line, "/")
			if dirOnly {
				line = strings.TrimSuffix(line, "/")
			}
			line = strings.TrimPrefix(line, "/")
			if line == "" {
				continue
			}
			// Convert simple gitignore to our glob style.
			pat := line
			if !strings.Contains(pat, "/") {
				pat = "**/" + pat
			} else if !strings.HasPrefix(pat, "**/") && !strings.HasPrefix(pat, "*") {
				// path relative to root
			}
			rules = append(rules, giRule{negated: neg, dirOnly: dirOnly, pattern: pat})
		}
		_ = f.Close()
	}
	if len(rules) == 0 {
		return nil
	}
	return &gitIgnore{rules: rules}
}

// ignored returns true if rel should be ignored.
func (g *gitIgnore) ignored(rel string, isDir bool) bool {
	if g == nil {
		return false
	}
	rel = filepath.ToSlash(rel)
	matched := false
	for _, r := range g.rules {
		p := r.pattern
		hit := matchGlob(p, rel) ||
			matchGlob(p+"/**", rel) ||
			matchGlob("**/"+p, rel) ||
			matchGlob("**/"+p+"/**", rel) ||
			matchGlob(p, filepath.Base(rel))
		// directory rule "ignored/" matches path under ignored/
		if r.dirOnly {
			if rel == p || strings.HasPrefix(rel, p+"/") ||
				matchGlob("**/"+p, rel) || strings.Contains(rel, "/"+p+"/") ||
				strings.HasPrefix(rel, p+"/") {
				hit = true
			}
			// first path segment
			if segs := strings.Split(rel, "/"); len(segs) > 0 && segs[0] == p {
				hit = true
			}
		}
		if !hit {
			continue
		}
		if r.negated {
			matched = false
		} else {
			matched = true
		}
	}
	return matched
}
