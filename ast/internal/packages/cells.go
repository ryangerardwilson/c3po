package packages

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/ryangerardwilson/c3po/ast/internal/measure"
)

// discoverCells finds uv workspace members and standalone Go modules.
// A Go module is its own cell. It is not attached to the Python runtime.
func discoverCells(root string, seen map[string]bool) ([]Package, error) {
	var out []Package
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if path != root && (measure.SkipDirName(d.Name()) || isNestedGit(root, path)) {
				return filepath.SkipDir
			}
			return nil
		}
		base := d.Name()
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		dir := filepath.ToSlash(filepath.Dir(rel))
		if dir == "." {
			dir = ""
		}
		switch base {
		case "pyproject.toml":
			if dir == "" {
				out = append(out, uvMembers(root, rel)...)
			}
		case "go.mod":
			id := dir
			if id == "" {
				id = "go.mod"
			}
			if seen[id] || covered(seen, id) {
				return nil
			}
			seen[id] = true
			out = append(out, Package{
				ID: id, Kind: "module", Path: id, Language: []string{"go"},
				Verify: []string{"go test -C " + id + " ./..."},
			})
		}
		return nil
	})
	out = append(out, goWorkModules(root, seen)...)
	return out, err
}

// GoWorkUses lists module directories named by a root go.work.
// "." means the workspace root.
func GoWorkUses(root string) []string {
	b, err := os.ReadFile(filepath.Join(root, "go.work"))
	if err != nil {
		return nil
	}
	var dirs []string
	in := false
	for _, line := range strings.Split(string(b), "\n") {
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "//") {
			continue
		}
		if strings.HasPrefix(trim, "use") {
			rest := strings.TrimSpace(strings.TrimPrefix(trim, "use"))
			if rest == "" || rest == "(" {
				in = true
				continue
			}
			rest = strings.TrimSuffix(strings.TrimSpace(strings.TrimPrefix(rest, "(")), ")")
			if rest != "" {
				dirs = append(dirs, cleanUse(rest))
			}
			if strings.Contains(trim, ")") {
				in = false
			}
			continue
		}
		if in {
			if strings.HasPrefix(trim, ")") {
				in = false
				continue
			}
			dirs = append(dirs, cleanUse(strings.TrimSuffix(trim, ")")))
			if strings.Contains(trim, ")") {
				in = false
			}
		}
	}
	return dirs
}

func cleanUse(path string) string {
	path = strings.Trim(path, "\"` ")
	path = filepath.ToSlash(filepath.Clean(path))
	path = strings.TrimPrefix(path, "./")
	if path == "" {
		return "."
	}
	return path
}

func goWorkModules(root string, seen map[string]bool) []Package {
	var out []Package
	for _, dir := range GoWorkUses(root) {
		id := dir
		if id == "." {
			id = "go.mod"
		}
		if seen[id] || covered(seen, id) {
			continue
		}
		mod := filepath.Join(root, "go.mod")
		if dir != "." {
			mod = filepath.Join(root, filepath.FromSlash(dir), "go.mod")
		}
		if !fileExists(mod) {
			continue
		}
		seen[id] = true
		verifyID := id
		if verifyID == "go.mod" {
			verifyID = "."
		}
		out = append(out, Package{
			ID: id, Kind: "module", Path: id, Language: []string{"go"},
			Verify: []string{"go test -C " + verifyID + " ./..."},
		})
	}
	return out
}

func covered(seen map[string]bool, id string) bool {
	for have := range seen {
		if id == have || strings.HasPrefix(id, have+"/") {
			return true
		}
	}
	return false
}

func isNestedGit(root, path string) bool {
	if path == root {
		return false
	}
	st, err := os.Stat(filepath.Join(path, ".git"))
	return err == nil && st.IsDir()
}

func uvMembers(root, rel string) []Package {
	text, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		return nil
	}
	src := string(text)
	if !strings.Contains(src, "[tool.uv.workspace]") {
		return nil
	}
	var members []string
	in := false
	depth := 0
	for _, line := range strings.Split(src, "\n") {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "[") {
			in = trim == "[tool.uv.workspace]"
			continue
		}
		if !in {
			continue
		}
		depth += strings.Count(trim, "[") - strings.Count(trim, "]")
		if strings.Contains(trim, "members") || depth > 0 {
			members = append(members, quoted(trim)...)
		}
	}
	var out []Package
	for _, mem := range members {
		paths := expandMember(root, mem)
		for _, p := range paths {
			if !fileExists(filepath.Join(root, filepath.FromSlash(p), "pyproject.toml")) {
				continue
			}
			name := projectName(filepath.Join(root, filepath.FromSlash(p), "pyproject.toml"))
			pkg := Package{ID: p, Kind: "package", Path: p, Language: []string{"py"}, Install: name}
			out = append(out, pkg)
		}
	}
	return out
}

func expandMember(root, mem string) []string {
	if !strings.Contains(mem, "*") {
		return []string{filepath.ToSlash(mem)}
	}
	dir := filepath.Dir(mem)
	entries, err := os.ReadDir(filepath.Join(root, dir))
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			out = append(out, filepath.ToSlash(filepath.Join(dir, e.Name())))
		}
	}
	return out
}

func projectName(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	in := false
	for _, line := range strings.Split(string(b), "\n") {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "[") {
			in = trim == "[project]"
			continue
		}
		if in && strings.HasPrefix(trim, "name") && strings.Contains(trim, "=") {
			q := quoted(trim)
			if len(q) > 0 {
				return q[0]
			}
		}
	}
	return ""
}

func quoted(line string) []string {
	var out []string
	rest := line
	for {
		i := strings.Index(rest, `"`)
		if i < 0 {
			break
		}
		rest = rest[i+1:]
		j := strings.Index(rest, `"`)
		if j < 0 {
			break
		}
		out = append(out, rest[:j])
		rest = rest[j+1:]
	}
	return out
}

// PythonDeps lists declared dependency names for a pyproject. It does not
// invent a link to a Go module.
func PythonDeps(root, rel string) []string {
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel), "pyproject.toml"))
	if err != nil {
		return nil
	}
	var deps []string
	inDeps := false
	depth := 0
	for _, line := range strings.Split(string(b), "\n") {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "dependencies") && strings.Contains(trim, "=") {
			inDeps = true
		}
		if inDeps {
			depth += strings.Count(trim, "[") - strings.Count(trim, "]")
			for _, q := range quoted(trim) {
				name := q
				for _, cut := range []string{">=", "<=", "==", "~=", ">", "<", "["} {
					if i := strings.Index(name, cut); i >= 0 {
						name = name[:i]
					}
				}
				name = strings.TrimSpace(name)
				if name != "" {
					deps = append(deps, name)
				}
			}
			if depth <= 0 && strings.Contains(trim, "]") {
				inDeps = false
			}
		}
	}
	return deps
}
