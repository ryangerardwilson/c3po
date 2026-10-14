package discover

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ryangerardwilson/c3po/complexity/internal/config"
	"github.com/ryangerardwilson/c3po/complexity/internal/glob"
)

// File is one source file selected for analysis.
type File struct {
	Path     string
	Abs      string
	Language string
	Ext      string
}

// Options narrows a walk.
type Options struct {
	IncludeTests bool
	Languages    map[string]bool
	ExtraExclude []string
}

// Walk returns supported files under root. It prunes dependency directories,
// skips gitignored paths, and does not enter a nested git checkout.
func Walk(root string, cfg config.Config, opt Options) ([]File, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	w := walker{root: root, cfg: cfg, opt: opt}
	err = filepath.WalkDir(root, w.visit)
	if err != nil {
		return nil, err
	}
	return dropIgnored(root, w.found), nil
}

type walker struct {
	root  string
	cfg   config.Config
	opt   Options
	found []File
}

func (w *walker) visit(path string, d os.DirEntry, err error) error {
	if err != nil {
		return nil
	}
	if d.IsDir() {
		return skipDir(w.root, path, d.Name())
	}
	if !d.Type().IsRegular() {
		return nil
	}
	return w.keep(path, d)
}

func (w *walker) keep(path string, d os.DirEntry) error {
	rel, err := filepath.Rel(w.root, path)
	if err != nil {
		return nil
	}
	rel = filepath.ToSlash(rel)
	lang, ext := Language(rel)
	if lang == "" || excluded(rel, w.cfg, w.opt) {
		return nil
	}
	if !wantLang(lang, w.opt) {
		return nil
	}
	info, err := d.Info()
	if err != nil || info.Size() > 1_000_000 {
		return nil
	}
	w.found = append(w.found, File{Path: rel, Abs: path, Language: lang, Ext: ext})
	return nil
}

func wantLang(lang string, opt Options) bool {
	if len(opt.Languages) == 0 {
		return true
	}
	return opt.Languages[lang] || opt.Languages[family(lang)]
}

func skipDir(root, path, name string) error {
	if path != root && config.PruneDir(name) {
		return filepath.SkipDir
	}
	if path != root && isGitCheckout(path) {
		return filepath.SkipDir
	}
	return nil
}

func isGitCheckout(path string) bool {
	_, err := os.Stat(filepath.Join(path, ".git"))
	return err == nil
}

func excluded(rel string, cfg config.Config, opt Options) bool {
	for _, p := range cfg.Exclude.Patterns {
		if glob.Match(p, rel) {
			return true
		}
	}
	for _, p := range opt.ExtraExclude {
		if glob.Match(p, rel) {
			return true
		}
	}
	if opt.IncludeTests || cfg.Exclude.IncludeTests {
		return false
	}
	for _, p := range cfg.Exclude.Tests {
		if glob.Match(p, rel) {
			return true
		}
	}
	return false
}

func family(lang string) string {
	if lang == "javascript" {
		return "typescript"
	}
	return lang
}

// Language maps a slash-separated path to a language id.
func Language(path string) (string, string) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go":
		return "go", ext
	case ".py":
		return "python", ext
	case ".rs":
		return "rust", ext
	case ".ts", ".tsx":
		return "typescript", ext
	case ".js", ".jsx", ".mjs", ".cjs":
		return "javascript", ext
	case ".prisma":
		return "prisma", ext
	case ".sql":
		return "sql", ext
	default:
		return "", ext
	}
}

func dropIgnored(root string, files []File) []File {
	if len(files) == 0 || !isGitCheckout(root) {
		return files
	}
	ignored := gitIgnored(root, files)
	if ignored == nil {
		return files
	}
	out := files[:0]
	for _, f := range files {
		if !ignored[f.Path] {
			out = append(out, f)
		}
	}
	return out
}

// Skip reports whether a slash-separated path is outside the analysis set.
func Skip(rel string, cfg config.Config, includeTests bool) bool {
	return excluded(rel, cfg, Options{IncludeTests: includeTests})
}

// HasCode reports whether root contains an in-scope source file.
func HasCode(root string, cfg config.Config) bool {
	root, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	found := false
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if found {
			return filepath.SkipAll
		}
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return skipDir(root, path, d.Name())
		}
		if !d.Type().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if !codeLang(Language(rel)) || excluded(rel, cfg, Options{}) {
			return nil
		}
		info, err := d.Info()
		if err != nil || info.Size() > 1_000_000 {
			return nil
		}
		found = true
		return filepath.SkipAll
	})
	return found
}

func codeLang(lang, _ string) bool {
	switch lang {
	case "go", "python", "rust", "typescript", "javascript":
		return true
	default:
		return false
	}
}

func gitIgnored(root string, files []File) map[string]bool {
	cmd := exec.Command("git", "-C", root, "check-ignore", "-z", "--stdin")
	var in bytes.Buffer
	for _, f := range files {
		in.WriteString(f.Path)
		in.WriteByte(0)
	}
	cmd.Stdin = &in
	out, err := cmd.Output()
	if err != nil && len(out) == 0 {
		return map[string]bool{}
	}
	set := map[string]bool{}
	for _, p := range bytes.Split(out, []byte{0}) {
		if len(p) > 0 {
			set[filepath.ToSlash(string(p))] = true
		}
	}
	return set
}
