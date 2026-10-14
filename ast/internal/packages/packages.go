package packages

import (
	"bufio"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type Package struct {
	ID       string   `json:"id"`
	Kind     string   `json:"kind"`
	Path     string   `json:"path"`
	Language []string `json:"language"`
	Install  string   `json:"install,omitempty"`
	Verify   []string `json:"verify,omitempty"`
}

type AffectedResult struct {
	Paths    []string  `json:"paths"`
	Packages []Package `json:"packages"`
	Unmapped []string  `json:"unmapped"`
}

func Discover(root string) ([]Package, error) {
	out := []Package{}
	apps, err := discoverApps(root)
	if err != nil {
		return nil, err
	}
	out = append(out, apps...)
	tests, err := discoverTests(root)
	if err != nil {
		return nil, err
	}
	out = append(out, tests...)
	out = append(out, discoverDesktop(root)...)
	seen := map[string]bool{}
	for _, p := range out {
		seen[p.ID] = true
	}
	cells, err := discoverCells(root, seen)
	if err != nil {
		return nil, err
	}
	out = append(out, cells...)
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}

func discoverApps(root string) ([]Package, error) {
	entries, err := os.ReadDir(filepath.Join(root, "Apps"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []Package
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		rel := filepath.ToSlash(filepath.Join("Apps", e.Name()))
		dir := filepath.Join(root, rel)
		hasGo := fileExists(filepath.Join(dir, "go.mod"))
		hasPkg := fileExists(filepath.Join(dir, "package.json"))
		hasInstall := fileExists(filepath.Join(dir, "install.sh"))
		if !hasGo && !hasPkg && !hasInstall {
			continue
		}
		pkg := Package{ID: rel, Kind: "app", Path: rel}
		if hasGo {
			pkg.Language = append(pkg.Language, "go")
			pkg.Verify = append(pkg.Verify, "go test -C "+rel+" ./...")
		}
		if hasPkg {
			pkg.Language = append(pkg.Language, "js")
			if npmHasTestScript(filepath.Join(dir, "package.json")) {
				pkg.Verify = append(pkg.Verify, "npm test --prefix "+rel)
			}
		}
		if fileExists(filepath.Join(dir, "pyproject.toml")) || fileExists(filepath.Join(dir, "requirements.txt")) {
			pkg.Language = append(pkg.Language, "py")
			if dirExists(filepath.Join(dir, "tests")) {
				pkg.Verify = append(pkg.Verify, "PYTHONPATH="+rel+" python3 -m unittest discover -s "+rel+"/tests -q")
			}
		}
		if hasInstall {
			pkg.Install = rel + "/install.sh from " + rel
		}
		out = append(out, pkg)
	}
	return out, nil
}

func discoverTests(root string) ([]Package, error) {
	entries, err := os.ReadDir(filepath.Join(root, "Tests"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []Package
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		rel := filepath.ToSlash(filepath.Join("Tests", e.Name()))
		if !fileExists(filepath.Join(root, rel, "run.sh")) {
			continue
		}
		out = append(out, Package{
			ID:       rel,
			Kind:     "test",
			Path:     rel,
			Language: []string{"sh"},
			Verify:   []string{rel + "/run.sh"},
		})
	}
	return out, nil
}

func discoverDesktop(root string) []Package {
	var out []Package
	if dirExists(filepath.Join(root, ".config", "hypr")) {
		out = append(out, Package{ID: ".config/hypr", Kind: "desktop", Path: ".config/hypr"})
	}
	if dirExists(filepath.Join(root, ".config", "quickshell")) {
		pkg := Package{ID: ".config/quickshell", Kind: "desktop", Path: ".config/quickshell"}
		if fileExists(filepath.Join(root, "Tests", "quickshell-desktop", "run.sh")) {
			pkg.Verify = []string{"Tests/quickshell-desktop/run.sh"}
		}
		out = append(out, pkg)
	}
	return out
}

func MapPaths(pkgs []Package, paths []string) AffectedResult {
	byPrefix := make([]Package, len(pkgs))
	copy(byPrefix, pkgs)
	sort.Slice(byPrefix, func(i, j int) bool { return len(byPrefix[i].Path) > len(byPrefix[j].Path) })

	seen := map[string]bool{}
	var hits []Package
	var unmapped []string
	normed := uniqueNormPaths(paths)
	for _, p := range normed {
		matched := false
		for _, pkg := range byPrefix {
			if pathInPackage(p, pkg.Path) {
				if !seen[pkg.ID] {
					hits = append(hits, pkg)
					seen[pkg.ID] = true
				}
				matched = true
				break
			}
		}
		if !matched {
			unmapped = append(unmapped, p)
		}
	}
	sort.Slice(hits, func(i, j int) bool { return hits[i].Path < hits[j].Path })
	return AffectedResult{Paths: normed, Packages: hits, Unmapped: unmapped}
}

func GitChangedPaths(root string) ([]string, error) {
	run := func(args ...string) ([]string, error) {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		out, err := cmd.Output()
		if err != nil {
			return nil, err
		}
		var paths []string
		sc := bufio.NewScanner(strings.NewReader(string(out)))
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line != "" {
				paths = append(paths, filepath.ToSlash(line))
			}
		}
		return paths, sc.Err()
	}
	diff, err := run("diff", "--name-only", "HEAD")
	if err != nil {
		// not a git repo or no HEAD
		return nil, nil
	}
	untracked, err := run("ls-files", "--others", "--exclude-standard")
	if err != nil {
		return diff, nil
	}
	return uniqueNormPaths(append(diff, untracked...)), nil
}

func pathInPackage(path, pkgPath string) bool {
	path = strings.TrimPrefix(filepath.ToSlash(path), "./")
	pkgPath = strings.TrimPrefix(filepath.ToSlash(pkgPath), "./")
	if path == pkgPath {
		return true
	}
	return strings.HasPrefix(path, pkgPath+"/")
}

func uniqueNormPaths(paths []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, p := range paths {
		p = strings.TrimSpace(p)
		p = strings.TrimPrefix(filepath.ToSlash(p), "./")
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

func fileExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}

func dirExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.IsDir()
}

func npmHasTestScript(path string) bool {
	b, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var pkg struct {
		Scripts map[string]string `json:"scripts"`
	}
	if json.Unmarshal(b, &pkg) != nil {
		return false
	}
	_, ok := pkg.Scripts["test"]
	return ok
}
