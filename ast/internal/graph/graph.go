package graph

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"go/parser"
	"go/token"

	"github.com/ryangerardwilson/c3po/ast/internal/measure"
	"github.com/ryangerardwilson/c3po/ast/internal/packages"
)

type Node struct {
	ID   string `json:"id"`
	Kind string `json:"kind"`
	Path string `json:"path"`
}

type Edge struct {
	From string `json:"from"`
	To   string `json:"to"`
	Rel  string `json:"rel"`
}

type Graph struct {
	Nodes map[string]Node   `json:"-"`
	Out   map[string][]Edge `json:"-"`
	In    map[string][]Edge `json:"-"`
	mods  []module
}

func ID(kind, path string) string { return kind + ":" + path }

func (g *Graph) addNode(n Node) {
	if g.Nodes == nil {
		g.Nodes = map[string]Node{}
		g.Out = map[string][]Edge{}
		g.In = map[string][]Edge{}
	}
	if _, ok := g.Nodes[n.ID]; ok {
		return
	}
	g.Nodes[n.ID] = n
}

func (g *Graph) addEdge(from, to, rel string) {
	if from == to {
		return
	}
	if _, ok := g.Nodes[from]; !ok {
		return
	}
	if _, ok := g.Nodes[to]; !ok {
		if rel == "imports" && strings.HasPrefix(to, "gopkg:") {
			g.addNode(Node{ID: to, Kind: "gopkg", Path: strings.TrimPrefix(to, "gopkg:")})
		} else {
			return
		}
	}
	e := Edge{From: from, To: to, Rel: rel}
	for _, x := range g.Out[from] {
		if x.To == to && x.Rel == rel {
			return
		}
	}
	g.Out[from] = append(g.Out[from], e)
	g.In[to] = append(g.In[to], e)
}

type module struct {
	Path string
	Name string
}

func Build(root string, pkgs []packages.Package) (*Graph, error) {
	g := &Graph{}
	for _, p := range pkgs {
		g.addNode(Node{ID: ID(p.Kind, p.Path), Kind: p.Kind, Path: p.Path})
	}
	mods, err := findModules(root)
	if err != nil {
		return nil, err
	}
	g.mods = mods
	for _, m := range mods {
		if err := g.addModule(root, m, pkgs); err != nil {
			return nil, err
		}
	}
	g.addPythonDeps(root, pkgs)
	return g, nil
}

func (g *Graph) addPythonDeps(root string, pkgs []packages.Package) {
	byName := map[string]packages.Package{}
	for _, p := range pkgs {
		if p.Install != "" {
			byName[p.Install] = p
		}
	}
	for _, p := range pkgs {
		py := false
		for _, lang := range p.Language {
			if lang == "py" {
				py = true
			}
		}
		if !py {
			continue
		}
		for _, dep := range packages.PythonDeps(root, p.Path) {
			other, ok := byName[dep]
			if !ok || other.ID == p.ID {
				continue
			}
			g.addEdge(ID(p.Kind, p.Path), ID(other.Kind, other.Path), "depends")
		}
	}
}

func findModules(root string) ([]module, error) {
	var out []module
	add := func(rel string) error {
		b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel), "go.mod"))
		if err != nil {
			return err
		}
		name := moduleName(string(b))
		if name == "" {
			return nil
		}
		if rel == "." {
			rel = ""
		}
		out = append(out, module{Path: rel, Name: name})
		return nil
	}
	if fileExists(filepath.Join(root, "go.mod")) {
		if err := add("."); err != nil {
			return nil, err
		}
	}
	seen := map[string]bool{}
	for _, m := range out {
		seen[m.Path] = true
	}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if path != root && (measure.SkipDirName(d.Name()) || nestedGit(root, path)) {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Name() != "go.mod" {
			return nil
		}
		rel, err := filepath.Rel(root, filepath.Dir(path))
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if rel == "." {
			rel = ""
		}
		if seen[rel] {
			return nil
		}
		seen[rel] = true
		return add(rel)
	})
	if err != nil {
		return out, err
	}
	for _, rel := range packages.GoWorkUses(root) {
		key := rel
		addRel := rel
		if rel == "." {
			key = ""
			addRel = "."
		}
		if seen[key] {
			continue
		}
		if err := add(addRel); err != nil {
			return nil, err
		}
		seen[key] = true
	}
	return out, nil
}

func nestedGit(root, path string) bool {
	if path == root {
		return false
	}
	st, err := os.Stat(filepath.Join(path, ".git"))
	return err == nil && st.IsDir()
}

func moduleName(src string) string {
	for _, line := range strings.Split(src, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	return ""
}

func (g *Graph) addModule(root string, m module, pkgs []packages.Package) error {
	modAbs := root
	if m.Path != "" {
		modAbs = filepath.Join(root, filepath.FromSlash(m.Path))
	}
	gopkgs := map[string]struct{}{}
	var files []string
	err := filepath.WalkDir(modAbs, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if measure.SkipDirName(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".go") {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		files = append(files, rel)
		dir := filepath.ToSlash(filepath.Dir(rel))
		if dir == "." {
			dir = m.Path
		}
		gopkgs[dir] = struct{}{}
		return nil
	})
	if err != nil {
		return err
	}
	for dir := range gopkgs {
		g.addNode(Node{ID: ID("gopkg", dir), Kind: "gopkg", Path: dir})
		for _, p := range pkgs {
			if p.Path == dir || (p.Path != "" && strings.HasPrefix(dir, p.Path+"/")) {
				g.addEdge(ID(p.Kind, p.Path), ID("gopkg", dir), "contains")
			}
		}
	}
	fset := token.NewFileSet()
	for _, rel := range files {
		g.addNode(Node{ID: ID("file", rel), Kind: "file", Path: rel})
		dir := filepath.ToSlash(filepath.Dir(rel))
		if dir == "." {
			dir = m.Path
		}
		g.addEdge(ID("file", rel), ID("gopkg", dir), "in_package")
		abs := filepath.Join(root, filepath.FromSlash(rel))
		f, err := parser.ParseFile(fset, abs, nil, parser.ImportsOnly)
		if err != nil {
			continue
		}
		for _, is := range f.Imports {
			imp := strings.Trim(is.Path.Value, `"`)
			target, ok := resolveImport(g.mods, imp)
			if !ok {
				continue
			}
			g.addEdge(ID("file", rel), ID("gopkg", target), "imports")
			g.addEdge(ID("gopkg", dir), ID("gopkg", target), "imports")
		}
	}
	return nil
}

func resolveImport(mods []module, imp string) (string, bool) {
	var best module
	bestLen := -1
	for _, m := range mods {
		if imp == m.Name || strings.HasPrefix(imp, m.Name+"/") {
			if len(m.Name) > bestLen {
				best, bestLen = m, len(m.Name)
			}
		}
	}
	if bestLen < 0 {
		return "", false
	}
	rest := strings.TrimPrefix(imp, best.Name)
	rest = strings.TrimPrefix(rest, "/")
	if rest == "" {
		return best.Path, true
	}
	if best.Path == "" {
		return rest, true
	}
	return best.Path + "/" + rest, true
}

func fileExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}

func (g *Graph) Resolve(q string) (Node, bool) {
	q = strings.TrimPrefix(filepath.ToSlash(q), "./")
	if n, ok := g.Nodes[q]; ok {
		return n, true
	}
	for _, kind := range []string{"file", "gopkg", "app", "test", "desktop"} {
		if n, ok := g.Nodes[ID(kind, q)]; ok {
			return n, true
		}
	}
	var hits []Node
	for _, n := range g.Nodes {
		if n.Path == q || strings.HasSuffix(n.ID, ":"+q) {
			hits = append(hits, n)
		}
	}
	if len(hits) == 1 {
		return hits[0], true
	}
	return Node{}, false
}

func (g *Graph) Neighbors(id string) []Edge {
	seen := map[string]Edge{}
	var out []Edge
	add := func(e Edge) {
		k := e.From + "\x00" + e.To + "\x00" + e.Rel
		if _, ok := seen[k]; ok {
			return
		}
		seen[k] = e
		out = append(out, e)
	}
	for _, e := range g.Out[id] {
		add(e)
	}
	for _, e := range g.In[id] {
		add(e)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Rel != out[j].Rel {
			return out[i].Rel < out[j].Rel
		}
		if out[i].From != out[j].From {
			return out[i].From < out[j].From
		}
		return out[i].To < out[j].To
	})
	return out
}

func (g *Graph) RingHash(id string) string {
	n, ok := g.Nodes[id]
	if !ok {
		return ""
	}
	ids := []string{n.ID}
	for _, e := range g.Neighbors(id) {
		ids = append(ids, e.From, e.To, e.Rel)
	}
	sort.Strings(ids)
	sum := sha256.Sum256([]byte(strings.Join(ids, "\n")))
	return hex.EncodeToString(sum[:])
}

func (g *Graph) Degree(id string) int {
	return len(g.Neighbors(id))
}

// Snapshot returns stable node and edge arrays. Counts must match these slices.
func (g *Graph) Snapshot() (nodes []Node, edges []Edge) {
	nodes = []Node{}
	edges = []Edge{}
	if g == nil {
		return nodes, edges
	}
	for _, n := range g.Nodes {
		nodes = append(nodes, n)
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].ID < nodes[j].ID })
	for _, n := range nodes {
		es := append([]Edge{}, g.Out[n.ID]...)
		sort.Slice(es, func(i, j int) bool {
			if es[i].Rel != es[j].Rel {
				return es[i].Rel < es[j].Rel
			}
			if es[i].To != es[j].To {
				return es[i].To < es[j].To
			}
			return es[i].From < es[j].From
		})
		edges = append(edges, es...)
	}
	return nodes, edges
}

func (g *Graph) ImportComponents() [][]string {
	nodes := []string{}
	for id, n := range g.Nodes {
		if n.Kind == "gopkg" {
			nodes = append(nodes, id)
		}
	}
	sort.Strings(nodes)
	idx := map[string]int{}
	for i, id := range nodes {
		idx[id] = i
	}
	adj := make([][]int, len(nodes))
	for _, id := range nodes {
		for _, e := range g.Out[id] {
			if e.Rel != "imports" {
				continue
			}
			j, ok := idx[e.To]
			if !ok {
				continue
			}
			i := idx[id]
			adj[i] = append(adj[i], j)
			adj[j] = append(adj[j], i)
		}
	}
	seen := make([]bool, len(nodes))
	var comps [][]string
	for i := range nodes {
		if seen[i] {
			continue
		}
		stack := []int{i}
		seen[i] = true
		var comp []string
		for len(stack) > 0 {
			u := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			comp = append(comp, nodes[u])
			for _, v := range adj[u] {
				if !seen[v] {
					seen[v] = true
					stack = append(stack, v)
				}
			}
		}
		sort.Strings(comp)
		comps = append(comps, comp)
	}
	sort.Slice(comps, func(i, j int) bool { return comps[i][0] < comps[j][0] })
	return comps
}

func (g *Graph) CutVertices() []string {
	nodes := []string{}
	for id, n := range g.Nodes {
		if n.Kind == "gopkg" {
			nodes = append(nodes, id)
		}
	}
	sort.Strings(nodes)
	idx := map[string]int{}
	for i, id := range nodes {
		idx[id] = i
	}
	n := len(nodes)
	adj := make([][]int, n)
	for _, id := range nodes {
		for _, e := range g.Out[id] {
			if e.Rel != "imports" {
				continue
			}
			j, ok := idx[e.To]
			if !ok {
				continue
			}
			i := idx[id]
			adj[i] = append(adj[i], j)
			adj[j] = append(adj[j], i)
		}
	}
	disc := make([]int, n)
	low := make([]int, n)
	parent := make([]int, n)
	for i := range parent {
		parent[i] = -1
	}
	art := make([]bool, n)
	time := 0
	var dfs func(u int)
	dfs = func(u int) {
		time++
		disc[u] = time
		low[u] = time
		children := 0
		for _, v := range adj[u] {
			if disc[v] == 0 {
				children++
				parent[v] = u
				dfs(v)
				if low[v] < low[u] {
					low[u] = low[v]
				}
				if parent[u] != -1 && low[v] >= disc[u] {
					art[u] = true
				}
			} else if v != parent[u] && disc[v] < low[u] {
				low[u] = disc[v]
			}
		}
		if parent[u] == -1 && children > 1 {
			art[u] = true
		}
	}
	for i := range nodes {
		if disc[i] == 0 {
			dfs(i)
		}
	}
	var out []string
	for i, ok := range art {
		if ok {
			out = append(out, nodes[i])
		}
	}
	return out
}

func (g *Graph) ReverseImportReach(paths []string) []string {
	seed := map[string]struct{}{}
	for _, p := range paths {
		p = strings.TrimPrefix(filepath.ToSlash(p), "./")
		if n, ok := g.Resolve(p); ok {
			if n.Kind == "gopkg" {
				seed[n.ID] = struct{}{}
			}
			if n.Kind == "file" {
				dir := filepath.ToSlash(filepath.Dir(n.Path))
				seed[ID("gopkg", dir)] = struct{}{}
			}
			if n.Kind == "app" || n.Kind == "test" || n.Kind == "desktop" {
				for _, e := range g.Out[n.ID] {
					if e.Rel == "contains" {
						seed[e.To] = struct{}{}
					}
				}
			}
		} else if strings.HasSuffix(p, ".go") {
			seed[ID("gopkg", filepath.ToSlash(filepath.Dir(p)))] = struct{}{}
		}
	}
	seen := map[string]struct{}{}
	stack := make([]string, 0, len(seed))
	for id := range seed {
		if _, ok := g.Nodes[id]; !ok {
			continue
		}
		seen[id] = struct{}{}
		stack = append(stack, id)
	}
	for len(stack) > 0 {
		u := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		for _, e := range g.In[u] {
			if e.Rel != "imports" {
				continue
			}
			if _, ok := seen[e.From]; ok {
				continue
			}
			seen[e.From] = struct{}{}
			stack = append(stack, e.From)
		}
	}
	pkgSeen := map[string]struct{}{}
	var out []string
	for id := range seen {
		for _, e := range g.In[id] {
			if e.Rel != "contains" {
				continue
			}
			n := g.Nodes[e.From]
			if n.Kind != "app" && n.Kind != "test" && n.Kind != "desktop" {
				continue
			}
			if _, ok := pkgSeen[n.Path]; ok {
				continue
			}
			pkgSeen[n.Path] = struct{}{}
			out = append(out, n.Path)
		}
	}
	sort.Strings(out)
	return out
}

func (g *Graph) TopGopkgs(k int) []Node {
	var nodes []Node
	for _, n := range g.Nodes {
		if n.Kind == "gopkg" {
			nodes = append(nodes, n)
		}
	}
	sort.Slice(nodes, func(i, j int) bool {
		di, dj := g.Degree(nodes[i].ID), g.Degree(nodes[j].ID)
		if di != dj {
			return di > dj
		}
		return nodes[i].Path < nodes[j].Path
	})
	if k > 0 && len(nodes) > k {
		nodes = nodes[:k]
	}
	return nodes
}

func (g *Graph) Counts() (nodesByKind, edgesByRel map[string]int, nNodes, nEdges int) {
	nodesByKind = map[string]int{}
	edgesByRel = map[string]int{}
	for _, n := range g.Nodes {
		nodesByKind[n.Kind]++
		nNodes++
	}
	for _, es := range g.Out {
		for _, e := range es {
			edgesByRel[e.Rel]++
			nEdges++
		}
	}
	return
}

func MergeAffected(base packages.AffectedResult, extraPaths []string, all []packages.Package) packages.AffectedResult {
	byID := map[string]packages.Package{}
	for _, p := range all {
		byID[p.ID] = p
	}
	seen := map[string]bool{}
	var pkgs []packages.Package
	for _, p := range base.Packages {
		seen[p.ID] = true
		pkgs = append(pkgs, p)
	}
	for _, path := range extraPaths {
		for _, p := range all {
			if p.Path == path && !seen[p.ID] {
				seen[p.ID] = true
				pkgs = append(pkgs, p)
			}
		}
	}
	sort.Slice(pkgs, func(i, j int) bool { return pkgs[i].Path < pkgs[j].Path })
	mapped := map[string]bool{}
	for _, p := range pkgs {
		mapped[p.Path] = true
	}
	var unmapped []string
	for _, u := range base.Unmapped {
		hit := false
		for _, p := range pkgs {
			if u == p.Path || strings.HasPrefix(u, p.Path+"/") {
				hit = true
				break
			}
		}
		if !hit {
			unmapped = append(unmapped, u)
		}
	}
	_ = mapped
	base.Packages = pkgs
	base.Unmapped = unmapped
	return base
}
