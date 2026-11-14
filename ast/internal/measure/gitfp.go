package measure

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
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
			return append([]byte("git-clean\n"), sampleNestedMtimes(root, 3)...)
		}
		// Without git there is no status to say what changed, so every file
		// the count could read is sampled.
		return sampleNestedMtimes(root, -1)
	}
	h := sha256.New()
	_, _ = h.Write([]byte("porcelain\n"))
	_, _ = h.Write(out)
	// Hash content of each dirty/untracked path so repeated edits while
	// porcelain status text is unchanged still invalidate the cache.
	// Paths are hashed in sorted order: map order would make the fingerprint
	// differ between runs on the same tree.
	dirty := ParsePorcelainPaths(string(out))
	paths := make([]string, 0, len(dirty))
	for path := range dirty {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		sum, err := hashDirtyFile(filepath.Join(root, path))
		if err != nil {
			_, _ = h.Write([]byte("missing:" + path))
			continue
		}
		_, _ = h.Write([]byte(path))
		_, _ = h.Write(sum)
	}
	return h.Sum(nil)
}

// hashDirtyFile streams the SHA-256 of one file. A directory is an error,
// not the digest of empty input.
func hashDirtyFile(abs string) ([]byte, error) {
	f, err := os.Open(abs)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("%s is a directory", abs)
	}
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}

// sampleNestedMtimes hashes the path, size, and mtime of entries under root
// down to maxDepth levels. A negative maxDepth has no limit.
func sampleNestedMtimes(root string, maxDepth int) []byte {
	h := sha256.New()
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil || rel == "." {
			return nil
		}
		if depth := pathDepth(rel); maxDepth >= 0 && depth > maxDepth {
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
		nsec, size := info.ModTime().UnixNano(), info.Size()
		var b [16]byte
		for i := 0; i < 8; i++ {
			b[i] = byte(nsec >> (8 * i))
			b[8+i] = byte(size >> (8 * i))
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
