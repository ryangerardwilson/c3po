package vibetree

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// skipNames are VCS and dependency directories. They are not a language list.
var skipNames = map[string]bool{
	".git": true, ".hg": true, ".svn": true, ".jj": true,
	"node_modules": true, "vendor": true, "dist": true, "build": true,
	"target": true, "coverage": true, ".next": true, ".nuxt": true,
	".turbo": true, ".cache": true, ".venv": true, "venv": true,
	"__pycache__": true,
}

type item struct {
	path string
	size int
}

// measure scores one file. ok is false when the file is not a module.
type measure func(path string) (int, bool)

func resolve(cwd, path string) (string, error) {
	if path == "" {
		path = cwd
	} else if !filepath.IsAbs(path) {
		path = filepath.Join(cwd, path)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	st, err := os.Stat(abs)
	if err != nil {
		return "", err
	}
	if !st.IsDir() {
		return "", fmt.Errorf("%s is not a directory", abs)
	}
	return abs, nil
}

func collect(root string, value measure) ([]item, error) {
	if items, ok := gitItems(root, value); ok {
		return items, nil
	}
	return walkCollect(root, value)
}

func gitItems(root string, value measure) ([]item, bool) {
	top, err := gitOutput(root, "rev-parse", "--show-toplevel")
	if err != nil {
		return nil, false
	}
	raw, err := gitOutput(top, "ls-files", "-z", "--cached", "--others", "--exclude-standard")
	if err != nil {
		return nil, false
	}
	return itemsUnder(top, root, raw, value), true
}

func gitOutput(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func itemsUnder(top, root, raw string, value measure) []item {
	prefix := relPrefix(top, root)
	var out []item
	for _, p := range strings.Split(raw, "\x00") {
		p = strings.TrimSpace(p)
		if p == "" || !under(p, prefix) {
			continue
		}
		rel := trimPrefix(p, prefix)
		if rel == "" {
			continue
		}
		if n, ok := value(filepath.Join(top, filepath.FromSlash(p))); ok {
			out = append(out, item{path: rel, size: n})
		}
	}
	return out
}

func relPrefix(top, root string) string {
	rel, err := filepath.Rel(top, root)
	if err != nil || rel == "." {
		return ""
	}
	return filepath.ToSlash(rel)
}

func under(path, prefix string) bool {
	if prefix == "" {
		return true
	}
	return path == prefix || strings.HasPrefix(path, prefix+"/")
}

func trimPrefix(path, prefix string) string {
	if prefix == "" {
		return path
	}
	return strings.TrimPrefix(path, prefix+"/")
}

func walkCollect(root string, value measure) ([]item, error) {
	var out []item
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return skipWalk(root, path, d.Name())
		}
		rel, ok := relOf(root, path)
		if !ok {
			return nil
		}
		if n, ok := value(path); ok {
			out = append(out, item{path: rel, size: n})
		}
		return nil
	})
	return out, err
}

func skipWalk(root, path, name string) error {
	if path != root && skipNames[name] {
		return filepath.SkipDir
	}
	return nil
}

func relOf(root, path string) (string, bool) {
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." {
		return "", false
	}
	return filepath.ToSlash(rel), true
}

func fileLines(path string) (int, bool) {
	st, err := os.Lstat(path)
	if err != nil || !st.Mode().IsRegular() || st.Size() == 0 {
		return 0, false
	}
	f, err := os.Open(path)
	if err != nil {
		return 0, false
	}
	defer f.Close()
	return countText(f)
}

func cycloOf(path string) (int, bool) {
	src, ok := textFile(path)
	if !ok {
		return 0, false
	}
	return cyclomaticFile(path, src)
}

func textFile(path string) ([]byte, bool) {
	st, err := os.Lstat(path)
	if err != nil || !st.Mode().IsRegular() || st.Size() == 0 {
		return nil, false
	}
	src, err := os.ReadFile(path)
	if err != nil || bytes.IndexByte(src, 0) >= 0 {
		return nil, false
	}
	return src, true
}

func countText(r io.Reader) (int, bool) {
	buf := make([]byte, 32*1024)
	lines, last := 0, byte('\n')
	seen := false
	for {
		n, err := r.Read(buf)
		if n > 0 && !textChunk(buf[:n], &seen) {
			return 0, false
		}
		if n > 0 {
			lines += bytes.Count(buf[:n], []byte{'\n'})
			last = buf[n-1]
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, false
		}
	}
	if last != '\n' {
		lines++
	}
	return lines, lines > 0
}

func textChunk(buf []byte, seen *bool) bool {
	if *seen {
		return true
	}
	*seen = true
	return bytes.IndexByte(buf, 0) < 0
}
