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
	negated  bool
	dirOnly  bool
	anchored bool
	pattern  string
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
			rule, ok := parseGitRule(sc.Text())
			if ok {
				rules = append(rules, rule)
			}
		}
		_ = f.Close()
	}
	if len(rules) == 0 {
		return nil
	}
	return &gitIgnore{rules: rules}
}

func parseGitRule(line string) (giRule, bool) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return giRule{}, false
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
	anchored := strings.HasPrefix(line, "/")
	line = strings.TrimPrefix(line, "/")
	if line == "" {
		return giRule{}, false
	}
	pat := line
	if !anchored && !strings.Contains(pat, "/") {
		pat = "**/" + pat
	}
	return giRule{negated: neg, dirOnly: dirOnly, anchored: anchored, pattern: pat}, true
}

// ignored returns true if rel should be ignored.
func (g *gitIgnore) ignored(rel string, _ bool) bool {
	if g == nil {
		return false
	}
	rel = filepath.ToSlash(rel)
	matched := false
	for _, r := range g.rules {
		if !ruleHit(r, rel) {
			continue
		}
		matched = !r.negated
	}
	return matched
}

func ruleHit(r giRule, rel string) bool {
	if r.anchored {
		return anchoredHit(r.pattern, rel)
	}
	return anywhereHit(r, rel)
}

func anchoredHit(p, rel string) bool {
	return matchGlob(p, rel) || matchGlob(p+"/**", rel) || strings.HasPrefix(rel, p+"/")
}

func anywhereHit(r giRule, rel string) bool {
	if nameHit(r.pattern, rel) {
		return true
	}
	if !r.dirOnly {
		return false
	}
	return dirHit(r.pattern, rel)
}

func nameHit(p, rel string) bool {
	return matchGlob(p, rel) ||
		matchGlob(p+"/**", rel) ||
		matchGlob("**/"+p, rel) ||
		matchGlob("**/"+p+"/**", rel) ||
		matchGlob(p, filepath.Base(rel))
}

func dirHit(p, rel string) bool {
	if rel == p || strings.HasPrefix(rel, p+"/") || strings.Contains(rel, "/"+p+"/") {
		return true
	}
	if matchGlob("**/"+p, rel) {
		return true
	}
	segs := strings.Split(rel, "/")
	return len(segs) > 0 && segs[0] == p
}
