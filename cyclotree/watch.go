package cyclotree

import (
	"fmt"
	"hash"
	"hash/fnv"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
)

func settle(s *screen, dir, query, stamp, pending string) (*screen, string, string) {
	next := sourceStamp(dir)
	if next == "" || next == stamp {
		return s, stamp, ""
	}
	if next != pending {
		return s, stamp, next
	}
	fresh, err := openTree(dir, query)
	if err != nil {
		return s, next, ""
	}
	return adoptScreen(s, fresh), next, ""
}

func sourceStamp(root string) string {
	h := fnv.New64a()
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		return stampOne(h, root, path, d, err)
	})
	if err != nil {
		return ""
	}
	return strconv.FormatUint(h.Sum64(), 16)
}

func stampOne(h hash.Hash, root, path string, d fs.DirEntry, err error) error {
	if err != nil || d == nil {
		return nil
	}
	if d.IsDir() {
		if path != root && skipStampDir(d.Name()) {
			return filepath.SkipDir
		}
		return nil
	}
	if !stampExt(d.Name()) {
		return nil
	}
	info, err := d.Info()
	if err != nil {
		return nil
	}
	rel, _ := filepath.Rel(root, path)
	fmt.Fprintf(h, "%s %d %d\n", filepath.ToSlash(rel), info.ModTime().UnixNano(), info.Size())
	return nil
}

var skipStampDirs = map[string]bool{
	".git": true, "node_modules": true, "vendor": true,
	"dist": true, "build": true, "target": true, ".cache": true,
}

func skipStampDir(name string) bool { return skipStampDirs[name] }

var stampExts = map[string]bool{
	".go": true, ".py": true, ".ts": true, ".tsx": true,
	".js": true, ".jsx": true, ".rs": true, ".mjs": true, ".cjs": true,
}

func stampExt(name string) bool {
	return stampExts[strings.ToLower(filepath.Ext(name))]
}

type cursorMark struct {
	key   string
	cycle bool
}

func adoptScreen(old, fresh *screen) *screen {
	if old == nil || fresh == nil {
		return fresh
	}
	open, seen := map[string]bool{}, map[string]bool{}
	collectPose(old.root, open, seen)
	mark := cursorMarkOf(old)
	applyPose(fresh, fresh.root, open, seen)
	fresh.layout()
	fresh.cursor = rowMatching(fresh, mark)
	fresh.shortcuts = old.shortcuts
	return fresh
}

func collectPose(n *node, open, seen map[string]bool) {
	stack := []*node{n}
	for len(stack) > 0 {
		cur := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if cur == nil {
			continue
		}
		key := poseKey(cur)
		seen[key] = true
		if cur.open {
			open[key] = true
		}
		stack = append(stack, cur.kids...)
	}
}

func applyPose(s *screen, n *node, open, seen map[string]bool) {
	stack := []*node{n}
	for len(stack) > 0 {
		cur := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if cur == nil {
			continue
		}
		if seen[poseKey(cur)] {
			setPose(s, cur, open[poseKey(cur)])
		}
		for i := len(cur.kids) - 1; i >= 0; i-- {
			stack = append(stack, cur.kids[i])
		}
	}
}

func setPose(s *screen, n *node, open bool) {
	if n == nil || n.cycle {
		return
	}
	if open {
		s.expand(n)
		return
	}
	n.open = false
}

func poseKey(n *node) string {
	if n == nil {
		return ""
	}
	return n.path + "\x00" + n.name
}

func cursorMarkOf(s *screen) cursorMark {
	if s == nil {
		return cursorMark{}
	}
	s.layout()
	if s.cursor < 0 || s.cursor >= len(s.rows) || s.rows[s.cursor].n == nil {
		return cursorMark{}
	}
	n := s.rows[s.cursor].n
	return cursorMark{key: poseKey(n), cycle: n.cycle}
}

func rowMatching(s *screen, mark cursorMark) int {
	if s == nil || mark.key == "" {
		return 0
	}
	fallback := -1
	for i, row := range s.rows {
		if poseKey(row.n) != mark.key {
			continue
		}
		if row.n != nil && row.n.cycle == mark.cycle {
			return i
		}
		if fallback < 0 {
			fallback = i
		}
	}
	if fallback >= 0 {
		return fallback
	}
	return 0
}
