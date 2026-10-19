package vibetree

import (
	"os"
	"path/filepath"
	"strings"

	fileparse "github.com/ryangerardwilson/c3po/vibetree/internal/parse"
)

// Function is one scored function. File is relative to the directory
// that was read, with forward slashes.
type Function struct {
	File, Name, Language string
	Line, Cyclomatic     int
}

// Functions scores every function under root. A git work tree uses
// git ls-files. Otherwise the walk skips VCS and dependency directories.
func Functions(root string) ([]Function, error) {
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

func walkFunctions(root string) ([]Function, error) {
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

func scoreFiles(files []srcFile) []Function {
	var out []Function
	for _, f := range files {
		out = append(out, scoreFile(f.abs, f.rel)...)
	}
	return out
}

func scoreFile(abs, rel string) []Function {
	src, ok := textFile(abs)
	if !ok {
		return nil
	}
	fns, _ := fileparse.File(rel, language(rel), src)
	var out []Function
	for _, fn := range fns {
		if fn.Cyclomatic <= 0 {
			continue
		}
		out = append(out, Function{
			File: rel, Name: fn.Name, Language: fn.Language,
			Line: fn.Line, Cyclomatic: fn.Cyclomatic,
		})
	}
	return out
}
