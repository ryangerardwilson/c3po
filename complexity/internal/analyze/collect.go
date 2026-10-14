package analyze

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ryangerardwilson/c3po/complexity/internal/config"
	"github.com/ryangerardwilson/c3po/complexity/internal/discover"
)

func collect(path string, opt Options) ([]discover.File, string, error) {
	shown := userPath(path)
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, "", err
	}
	st, err := os.Stat(abs)
	if err != nil {
		return nil, "", fmt.Errorf("path not found: %s", shown)
	}
	root, _ := config.GitRoot(abs)
	if opt.Diff != "" {
		if root == "" {
			return nil, "", fmt.Errorf("%s is not a git checkout", shown)
		}
		files, err := diffFiles(root, abs, opt)
		return files, root, err
	}
	if !st.IsDir() {
		return oneFile(abs, root, opt)
	}
	files, err := discover.Walk(abs, opt.Config, discover.Options{
		IncludeTests: opt.IncludeTests || opt.Config.Exclude.IncludeTests,
		Languages:    langSet(opt),
		ExtraExclude: opt.ExtraExclude,
	})
	if err != nil {
		return nil, root, err
	}
	return rebase(files, root), root, nil
}

func oneFile(abs, root string, opt Options) ([]discover.File, string, error) {
	rel := abs
	if root != "" {
		if r, err := filepath.Rel(root, abs); err == nil {
			rel = filepath.ToSlash(r)
		}
	}
	lang, ext := discover.Language(rel)
	if lang == "" || discover.Skip(rel, opt.Config, opt.IncludeTests) {
		return nil, root, nil
	}
	if !want(lang, langSet(opt)) {
		return nil, root, nil
	}
	return []discover.File{{Path: rel, Abs: abs, Language: lang, Ext: ext}}, root, nil
}

func diffFiles(root, under string, opt Options) ([]discover.File, error) {
	names, err := changedFiles(root, opt.Diff)
	if err != nil {
		return nil, err
	}
	langs := langSet(opt)
	var out []discover.File
	for name := range names {
		abs := filepath.Join(root, filepath.FromSlash(name))
		if !underPath(under, abs) {
			continue
		}
		lang, ext := discover.Language(name)
		if lang == "" || discover.Skip(name, opt.Config, opt.IncludeTests) || !want(lang, langs) {
			continue
		}
		st, err := os.Stat(abs)
		if err != nil || st.IsDir() || st.Size() > 1_000_000 {
			continue
		}
		out = append(out, discover.File{Path: filepath.ToSlash(name), Abs: abs, Language: lang, Ext: ext})
	}
	return out, nil
}

func rebase(files []discover.File, root string) []discover.File {
	if root == "" {
		return files
	}
	for i, f := range files {
		rel, err := filepath.Rel(root, f.Abs)
		if err == nil {
			files[i].Path = filepath.ToSlash(rel)
		}
	}
	return files
}

func underPath(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel == "." || !strings.HasPrefix(rel, "..")
}

func langSet(opt Options) map[string]bool {
	if opt.SQLOnly {
		return map[string]bool{"sql": true}
	}
	set := map[string]bool{}
	if len(opt.Langs) > 0 {
		for k, ok := range opt.Langs {
			if ok {
				set[k] = true
			}
		}
	} else {
		for _, l := range opt.Config.Languages.Include {
			set[l] = true
		}
	}
	if opt.ORMOnly {
		if len(opt.Langs) > 0 {
			delete(set, "sql")
			return set
		}
		return map[string]bool{"prisma": true, "python": true, "go": true, "typescript": true, "javascript": true, "rust": true}
	}
	if len(opt.Langs) == 0 && !opt.SQLOnly {
		set["prisma"] = true
		set["sql"] = true
		set["javascript"] = true
	}
	return set
}

func want(lang string, set map[string]bool) bool {
	if len(set) == 0 {
		return true
	}
	if set[lang] {
		return true
	}
	if lang == "javascript" && set["typescript"] {
		return true
	}
	return false
}
