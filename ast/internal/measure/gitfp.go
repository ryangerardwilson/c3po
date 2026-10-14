package measure

import (
	"crypto/sha256"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func runGit(root string, args ...string) []byte {
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	return out
}

// GitPorcelain returns porcelain status without stripping leading status spaces.
func GitPorcelain(root string) (string, error) {
	out := runGit(root, "status", "--porcelain=v1", "-uall")
	if out == nil {
		return "", os.ErrNotExist
	}
	return string(out), nil
}

// ParsePorcelainPaths returns path -> two-letter status from porcelain v1 text.
// Leading status spaces are preserved (do not trim lines).
func ParsePorcelainPaths(text string) map[string]string {
	m := map[string]string{}
	for _, line := range strings.Split(text, "\n") {
		if line == "" {
			continue
		}
		if len(line) < 4 {
			continue
		}
		st := line[0:2]
		path := line[3:]
		if i := strings.Index(path, " -> "); i >= 0 {
			path = path[i+4:]
		}
		path = strings.Trim(path, "\"")
		if path != "" {
			m[path] = st
		}
	}
	return m
}

func gitDirtyFingerprint(root string) []byte {
	out := runGit(root, "status", "--porcelain=v1", "-uall")
	if out == nil {
		if runGit(root, "rev-parse", "--is-inside-work-tree") != nil {
			// clean or empty status
			return append([]byte("git-clean\n"), sampleNestedMtimes(root)...)
		}
		return sampleNestedMtimes(root)
	}
	h := sha256.New()
	_, _ = h.Write([]byte("porcelain\n"))
	_, _ = h.Write(out)
	// Hash content of each dirty/untracked path so repeated edits while
	// porcelain status text is unchanged still invalidate the cache.
	for path := range ParsePorcelainPaths(string(out)) {
		abs := filepath.Join(root, path)
		data, err := os.ReadFile(abs)
		if err != nil {
			_, _ = h.Write([]byte("missing:" + path))
			continue
		}
		sum := sha256.Sum256(data)
		_, _ = h.Write([]byte(path))
		_, _ = h.Write(sum[:])
	}
	return h.Sum(nil)
}

func sampleNestedMtimes(root string) []byte {
	h := sha256.New()
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil || rel == "." {
			return nil
		}
		if depth := pathDepth(rel); depth > 3 {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() && SkipDirName(d.Name()) {
			return filepath.SkipDir
		}
		if d.IsDir() && path != root {
			if _, err := os.Stat(filepath.Join(path, ".git")); err == nil {
				return filepath.SkipDir
			}
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		_, _ = h.Write([]byte(rel))
		nsec := info.ModTime().UnixNano()
		var b [8]byte
		for i := 0; i < 8; i++ {
			b[i] = byte(nsec >> (8 * i))
		}
		_, _ = h.Write(b[:])
		return nil
	})
	return h.Sum(nil)
}

func pathDepth(rel string) int {
	if rel == "" || rel == "." {
		return 0
	}
	n := 1
	for i := 0; i < len(rel); i++ {
		if rel[i] == '/' || rel[i] == filepath.Separator {
			n++
		}
	}
	return n
}
