package cyclotree

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	fileparse "github.com/ryangerardwilson/c3po/cyclotree/internal/parse"
)

// scoredFn is one scored function. File is relative to the directory
// that was read, with forward slashes.
type scoredFn struct {
	File, Name, Language string
	Line, Cyclomatic     int
}

// scoreFunctions scores every function under root. A git work tree uses
// git ls-files. Otherwise the walk skips VCS and dependency directories.
func scoreFunctions(root string) ([]scoredFn, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if files, ok := gitSource(abs); ok {
		return scoreFiles(files), nil
	}
	return walkFunctions(abs)
}

type srcFile struct {
	abs, rel string
}

func gitSource(root string) ([]srcFile, bool) {
	top, err := gitOutput(root, "rev-parse", "--show-toplevel")
	if err != nil {
		return nil, false
	}
	raw, err := gitOutput(top, "ls-files", "-z", "--cached", "--others", "--exclude-standard")
	if err != nil {
		return nil, false
	}
	return gitListed(top, root, raw), true
}

func gitListed(top, root, raw string) []srcFile {
	prefix := relPrefix(top, root)
	var out []srcFile
	for _, p := range strings.Split(raw, "\x00") {
		p = strings.TrimSpace(p)
		if p == "" || !under(p, prefix) || language(p) == "" {
			continue
		}
		rel := trimPrefix(p, prefix)
		if rel == "" {
			continue
		}
		out = append(out, srcFile{abs: filepath.Join(top, filepath.FromSlash(p)), rel: rel})
	}
	return out
}

func walkFunctions(root string) ([]scoredFn, error) {
	var files []srcFile
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return skipWalk(root, path, d.Name())
		}
		rel, ok := relOf(root, path)
		if !ok || language(rel) == "" {
			return nil
		}
		files = append(files, srcFile{abs: path, rel: rel})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return scoreFiles(files), nil
}

func scoreFiles(files []srcFile) []scoredFn {
	var out []scoredFn
	for _, f := range files {
		out = append(out, scoreFile(f.abs, f.rel)...)
	}
	return out
}

func scoreFile(abs, rel string) []scoredFn {
	src, ok := textFile(abs)
	if !ok {
		return nil
	}
	fns, _ := fileparse.File(rel, language(rel), src)
	var out []scoredFn
	for _, fn := range fns {
		if fn.Cyclomatic <= 0 {
			continue
		}
		out = append(out, scoredFn{
			File: rel, Name: fn.Name, Language: fn.Language,
			Line: fn.Line, Cyclomatic: fn.Cyclomatic,
		})
	}
	return out
}

func language(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go":
		return "go"
	case ".py":
		return "python"
	case ".rs":
		return "rust"
	case ".ts", ".tsx":
		return "typescript"
	case ".js", ".jsx", ".mjs", ".cjs":
		return "javascript"
	default:
		return ""
	}
}

// skipNames are VCS and dependency directories. They are not a language list.
var skipNames = map[string]bool{
	".git": true, ".hg": true, ".svn": true, ".jj": true,
	"node_modules": true, "vendor": true, "dist": true, "build": true,
	"target": true, "coverage": true, ".next": true, ".nuxt": true,
	".turbo": true, ".cache": true, ".venv": true, "venv": true,
	"__pycache__": true,
}

func gitOutput(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
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
