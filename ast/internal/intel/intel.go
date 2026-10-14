package intel

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/ryangerardwilson/c3po/ast/internal/config"
	"github.com/ryangerardwilson/c3po/ast/internal/measure"
	"github.com/ryangerardwilson/c3po/ast/internal/trace"
)

// Entry is one outline line.
type Entry struct {
	Path  string `json:"path"`
	Kind  string `json:"kind"`
	Name  string `json:"name"`
	Line  int    `json:"line"`
	End   int    `json:"end"`
	Label string `json:"label"`
}

// MapOptions selects one page of a map.
// Depth < 0 means unlimited. Depth 0 is only the selected directory.
type MapOptions struct {
	Limit  int
	Offset int
	Depth  int
}

// MapResult is a page of outline entries.
type MapResult struct {
	Entries    []Entry `json:"entries"`
	Truncated  bool    `json:"truncated"`
	NextOffset *int    `json:"next_offset,omitempty"`
}

// Map outlines path (file or directory) under root.
func Map(root, relOrAbs string, cfg config.Config, maxEntries int) ([]Entry, error) {
	entries, _, err := MapReport(root, relOrAbs, cfg, maxEntries)
	return entries, err
}

func MapReport(root, relOrAbs string, cfg config.Config, maxEntries int) ([]Entry, bool, error) {
	res, err := MapQuery(root, relOrAbs, cfg, MapOptions{Limit: maxEntries, Depth: -1})
	if err != nil {
		return nil, false, err
	}
	return res.Entries, res.Truncated, nil
}

// MapQuery returns one stable page. Order is path, then line.
func MapQuery(root, relOrAbs string, cfg config.Config, opt MapOptions) (MapResult, error) {
	if opt.Limit <= 0 {
		opt.Limit = cfg.MaxMapEntries
	}
	if opt.Offset < 0 {
		opt.Offset = 0
	}
	target := root
	if relOrAbs != "" {
		p, err := resolve(root, relOrAbs)
		if err != nil {
			return MapResult{}, err
		}
		target = p
	}
	files, err := mapPaths(root, target, cfg, opt.Depth)
	if err != nil {
		return MapResult{}, err
	}
	need := opt.Offset + opt.Limit
	var collected []Entry
	truncated := false
	for _, rel := range files {
		es, err := mapFile(root, filepath.Join(root, filepath.FromSlash(rel)), rel)
		if err != nil {
			return MapResult{}, err
		}
		for _, e := range es {
			if len(collected) == need {
				truncated = true
				break
			}
			collected = append(collected, e)
		}
		if truncated {
			break
		}
	}
	page := []Entry{}
	if opt.Offset < len(collected) {
		page = collected[opt.Offset:]
	}
	res := MapResult{Entries: page, Truncated: truncated}
	if truncated {
		next := opt.Offset + len(page)
		res.NextOffset = &next
	}
	return res, nil
}

func mapPaths(root, target string, cfg config.Config, depth int) ([]string, error) {
	st, err := os.Stat(target)
	if err != nil {
		return nil, err
	}
	if !st.IsDir() {
		rel, _ := filepath.Rel(root, target)
		return []string{filepath.ToSlash(rel)}, nil
	}
	var files []string
	_, err = measure.NewScope(root, cfg).Walk(target, depth, func(_, rel string) error {
		files = append(files, rel)
		return nil
	})
	sort.Strings(files)
	return files, err
}

// Show finds a symbol by name or path:name and returns its body.
func Show(root, target string, cfg config.Config) (string, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return "", fmt.Errorf("show requires a target")
	}
	pathHint := ""
	name := target
	if i := strings.LastIndex(target, ":"); i >= 0 {
		pathHint = target[:i]
		name = target[i+1:]
	}

	var matches []Entry
	var bodies []string

	collect := func(path, rel string) {
		es, err := mapFile(root, path, rel)
		if err != nil {
			return
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return
		}
		lines := strings.Split(string(src), "\n")
		for _, e := range es {
			if e.Name != name {
				continue
			}
			if pathHint != "" {
				hint := filepath.ToSlash(pathHint)
				if rel != hint && !strings.HasSuffix(rel, "/"+hint) && rel != strings.TrimPrefix(hint, "./") {
					if !strings.Contains(rel, hint) {
						continue
					}
				}
			}
			body := sliceLines(lines, e.Line, e.End)
			matches = append(matches, e)
			bodies = append(bodies, body)
		}
	}

	if pathHint != "" {
		p, err := resolve(root, pathHint)
		if err == nil {
			rel, _ := filepath.Rel(root, p)
			collect(p, filepath.ToSlash(rel))
		}
	}

	if len(matches) == 0 {
		_, _ = measure.NewScope(root, cfg).Walk(root, -1, func(abs, rel string) error {
			collect(abs, rel)
			if len(matches) > 20 {
				return filepath.SkipAll
			}
			return nil
		})
	}

	if len(matches) == 0 {
		return "", fmt.Errorf("symbol %q not found", name)
	}
	if len(matches) > 1 {
		var b strings.Builder
		fmt.Fprintf(&b, "ambiguous symbol %q (%d matches):\n", name, len(matches))
		for _, m := range matches {
			fmt.Fprintf(&b, "  %s:%s (line %d)\n", m.Path, m.Name, m.Line)
		}
		fmt.Fprintf(&b, "disambiguate with path:symbol")
		return "", fmt.Errorf("%s", b.String())
	}
	return bodies[0], nil
}

// Hit is a search result.
type Hit struct {
	Path    string
	Line    int
	Content string
}

// SearchOpts filters and pages search.
type SearchOpts struct {
	PathPrefix string // relative dir or file under root
	Glob       string // optional extra glob like *.sh
	Offset     int    // hits to skip
	Regex      bool   // query is a Go regular expression
	IgnoreCase bool
}

// SearchResult is one page of hits. Total counts every hit up to
// maxSearchTotal; TotalCapped says the count stopped there.
type SearchResult struct {
	Hits        []Hit
	Offset      int
	Total       int
	TotalCapped bool
}

// More reports whether hits exist past this page.
func (r SearchResult) More() bool {
	return r.TotalCapped || r.Offset+len(r.Hits) < r.Total
}

// maxSearchTotal bounds how far a search keeps counting past its page.
const maxSearchTotal = 5000

// Search finds literal substring matches, capped.
func Search(root, query string, cfg config.Config) ([]Hit, bool, error) {
	res, err := SearchPage(root, query, cfg, SearchOpts{})
	return res.Hits, res.More(), err
}

// SearchPage returns one page of matches in walk order and the total count.
func SearchPage(root, query string, cfg config.Config, opts SearchOpts) (SearchResult, error) {
	if query == "" {
		return SearchResult{}, fmt.Errorf("search requires a query")
	}
	max := cfg.Enforcement.MaxSearchHits
	if max <= 0 {
		max = 50
	}
	if opts.Offset < 0 {
		opts.Offset = 0
	}
	match := func(line string) bool { return strings.Contains(line, query) }
	switch {
	case opts.Regex:
		expr := query
		if opts.IgnoreCase {
			expr = "(?i)" + expr
		}
		re, err := regexp.Compile(expr)
		if err != nil {
			return SearchResult{}, fmt.Errorf("invalid --regex: %w", err)
		}
		match = re.MatchString
	case opts.IgnoreCase:
		lower := strings.ToLower(query)
		match = func(line string) bool { return strings.Contains(strings.ToLower(line), lower) }
	}
	start := root
	if opts.PathPrefix != "" {
		p := opts.PathPrefix
		if !filepath.IsAbs(p) {
			p = filepath.Join(root, p)
		}
		start = filepath.Clean(p)
	}
	res := SearchResult{Offset: opts.Offset}
	_, err := measure.NewScope(root, cfg).Walk(start, -1, func(abs, rel string) error {
		if opts.Glob != "" && !matchSimpleGlob(opts.Glob, rel) {
			return nil
		}
		data, err := os.ReadFile(abs)
		if err != nil || strings.Contains(string(data), "\x00") {
			return nil
		}
		for i, line := range strings.Split(string(data), "\n") {
			if !match(line) {
				continue
			}
			if res.Total >= opts.Offset && len(res.Hits) < max {
				res.Hits = append(res.Hits, Hit{Path: rel, Line: i + 1, Content: truncateSnippet(strings.TrimRight(line, "\r"), 120)})
			}
			res.Total++
			if res.Total >= maxSearchTotal {
				res.TotalCapped = true
				return filepath.SkipAll
			}
		}
		return nil
	})
	return res, err
}

// Explain describes why a path is visible or not. A directory reports how
// many files under it the walk scope sees.
func Explain(root, relOrAbs string, cfg config.Config) (string, error) {
	abs, err := resolve(root, relOrAbs)
	if err != nil {
		return "", err
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(rootAbs, abs)
	if err != nil {
		return "", err
	}
	rel = filepath.ToSlash(rel)
	var b strings.Builder
	fmt.Fprintf(&b, "path: %s\n", rel)
	fmt.Fprintf(&b, "abs: %s\n", abs)
	st, err := os.Stat(abs)
	if err != nil {
		fmt.Fprintf(&b, "exists: false\n")
		fmt.Fprintf(&b, "error: %v\n", err)
		return b.String(), nil
	}
	fmt.Fprintf(&b, "exists: true\n")
	fmt.Fprintf(&b, "is_dir: %v\n", st.IsDir())
	sc := measure.NewScope(rootAbs, cfg)
	why, at := sc.SkipAncestor(abs)
	if why == "" && st.IsDir() {
		why, at = sc.SkipDir(abs), abs
	}
	if why == measure.SkipNestedRepo {
		fmt.Fprintf(&b, "under_nested_git: %s\n", at)
	} else {
		fmt.Fprintf(&b, "under_nested_git: false\n")
	}
	if st.IsDir() {
		visible := 0
		if why == "" {
			_, _ = sc.Walk(abs, -1, func(string, string) error {
				visible++
				return nil
			})
		}
		if why != "" {
			fmt.Fprintf(&b, "skip_reason: %s (%s)\n", why, at)
		}
		fmt.Fprintf(&b, "visible_files: %d\n", visible)
		fmt.Fprintf(&b, "map_visible: %v\n", visible > 0)
		fmt.Fprintf(&b, "search_visible: %v\n", visible > 0)
		return b.String(), nil
	}
	fmt.Fprintf(&b, "include_match: %v\n", pathIncluded(cfg, rel))
	if bin, err := isBin(abs); err == nil {
		fmt.Fprintf(&b, "binary: %v\n", bin)
	}
	if why == "" {
		why = sc.SkipFile(abs)
		at = abs
	}
	if why != "" {
		fmt.Fprintf(&b, "skip_reason: %s (%s)\n", why, at)
	}
	// A file named directly is always mappable; search needs it in scope.
	fmt.Fprintf(&b, "map_visible: true\n")
	fmt.Fprintf(&b, "search_visible: %v\n", why == "")
	return b.String(), nil
}

func isBin(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()
	buf := make([]byte, 4096)
	n, _ := f.Read(buf)
	return strings.Contains(string(buf[:n]), "\x00"), nil
}

func matchSimpleGlob(pattern, name string) bool {
	// reuse filepath match on full path segments
	ok, _ := filepath.Match(pattern, name)
	if ok {
		return true
	}
	ok, _ = filepath.Match(pattern, filepath.Base(name))
	return ok
}

func resolve(root, path string) (string, error) {
	if filepath.IsAbs(path) {
		return filepath.Clean(path), nil
	}
	return filepath.Clean(filepath.Join(root, path)), nil
}

func pathIncluded(cfg config.Config, rel string) bool {
	return measure.PathMatches(cfg, rel)
}

func mapFile(root, abs, rel string) ([]Entry, error) {
	base := filepath.Base(rel)
	low := strings.ToLower(rel)
	if strings.HasSuffix(rel, ".go") {
		return mapGo(abs, rel)
	}
	ext := strings.ToLower(filepath.Ext(rel))
	if ext == ".py" || ext == ".ts" || ext == ".tsx" || ext == ".js" || ext == ".jsx" {
		if ol := trace.FileOutline(abs, rel); len(ol) > 0 {
			var entries []Entry
			for _, o := range ol {
				entries = append(entries, Entry{
					Path: rel, Kind: o.Kind, Name: o.Name, Line: o.Line, End: o.End,
					Label: fmt.Sprintf("%s %s", o.Kind, o.Name),
				})
			}
			return entries, nil
		}
	}
	if strings.HasSuffix(rel, ".sh") || strings.HasSuffix(rel, ".bash") ||
		base == ".bashrc" || base == ".bash_profile" || base == ".profile" {
		return mapBash(abs, rel)
	}
	if strings.HasSuffix(low, ".md") {
		return mapMarkdown(abs, rel)
	}
	if strings.HasSuffix(low, ".json") {
		return mapJSON(abs, rel)
	}
	if strings.HasSuffix(low, ".toml") {
		return mapTOML(abs, rel)
	}
	if strings.HasSuffix(low, ".yml") || strings.HasSuffix(low, ".yaml") {
		return mapYAML(abs, rel)
	}
	if strings.HasSuffix(low, ".qml") {
		return mapQML(abs, rel)
	}
	return mapHeuristic(abs, rel)
}

var (
	reMDHeading = regexp.MustCompile(`(?m)^(#{1,6})[ \t]+(.+?)\s*$`)
	reMDReq     = regexp.MustCompile(`(?m)^###[ \t]+Requirement:[ \t]+(.+)$`)
	reMDScen    = regexp.MustCompile(`(?m)^####[ \t]+Scenario:[ \t]+(.+)$`)
	reTOMLSec   = regexp.MustCompile(`(?m)^\[([^\]]+)\]\s*$`)
	reYAMLKey   = regexp.MustCompile(`(?m)^([A-Za-z0-9_.-]+):\s`)
	reQMLType   = regexp.MustCompile(`(?m)^[ \t]*([A-Z][A-Za-z0-9_]*)\s*\{`)
	reQMLFunc   = regexp.MustCompile(`(?m)^[ \t]*function\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(`)
	reQMLProp   = regexp.MustCompile(`(?m)^[ \t]*property\s+\w+\s+([A-Za-z_][A-Za-z0-9_]*)`)
)

func mapMarkdown(abs, rel string) ([]Entry, error) {
	data, err := os.ReadFile(abs)
	if err != nil {
		return nil, err
	}
	text := string(data)
	var entries []Entry
	for _, m := range reMDReq.FindAllStringSubmatchIndex(text, -1) {
		name := text[m[2]:m[3]]
		line := 1 + strings.Count(text[:m[0]], "\n")
		entries = append(entries, Entry{Path: rel, Kind: "requirement", Name: name, Line: line, End: line + 5, Label: "requirement " + name})
	}
	for _, m := range reMDScen.FindAllStringSubmatchIndex(text, -1) {
		name := text[m[2]:m[3]]
		line := 1 + strings.Count(text[:m[0]], "\n")
		entries = append(entries, Entry{Path: rel, Kind: "scenario", Name: name, Line: line, End: line + 3, Label: "scenario " + name})
	}
	if len(entries) == 0 {
		for _, m := range reMDHeading.FindAllStringSubmatchIndex(text, -1) {
			level := m[3] - m[2]
			name := strings.TrimSpace(text[m[4]:m[5]])
			line := 1 + strings.Count(text[:m[0]], "\n")
			entries = append(entries, Entry{Path: rel, Kind: "heading", Name: name, Line: line, End: line, Label: fmt.Sprintf("h%d %s", level, name)})
			if len(entries) >= 80 {
				break
			}
		}
	}
	if len(entries) == 0 {
		return mapHeuristic(abs, rel)
	}
	return entries, nil
}

func mapJSON(abs, rel string) ([]Entry, error) {
	data, err := os.ReadFile(abs)
	if err != nil {
		return nil, err
	}
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return mapHeuristic(abs, rel)
	}
	var entries []Entry
	var walk func(prefix string, node any, depth int)
	walk = func(prefix string, node any, depth int) {
		if depth > 3 || len(entries) >= 100 {
			return
		}
		switch t := node.(type) {
		case map[string]any:
			for k, child := range t {
				path := k
				if prefix != "" {
					path = prefix + "." + k
				}
				entries = append(entries, Entry{Path: rel, Kind: "key", Name: path, Line: 1, End: 1, Label: "key " + path})
				walk(path, child, depth+1)
			}
		case []any:
			if prefix != "" {
				entries = append(entries, Entry{Path: rel, Kind: "array", Name: prefix, Line: 1, End: 1, Label: "array " + prefix})
			}
		}
	}
	walk("", v, 0)
	if len(entries) == 0 {
		return mapHeuristic(abs, rel)
	}
	return entries, nil
}

func mapTOML(abs, rel string) ([]Entry, error) {
	data, err := os.ReadFile(abs)
	if err != nil {
		return nil, err
	}
	text := string(data)
	var entries []Entry
	for _, m := range reTOMLSec.FindAllStringSubmatchIndex(text, -1) {
		name := text[m[2]:m[3]]
		line := 1 + strings.Count(text[:m[0]], "\n")
		entries = append(entries, Entry{Path: rel, Kind: "section", Name: name, Line: line, End: line + 10, Label: "section " + name})
	}
	depth := 0
	for i, line := range strings.Split(text, "\n") {
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "#") {
			continue
		}
		if depth == 0 && strings.HasPrefix(trim, "[") {
			continue
		}
		if depth == 0 {
			if idx := strings.IndexByte(line, '='); idx > 0 {
				key := strings.TrimSpace(line[:idx])
				if key != "" && !strings.Contains(key, " ") && !strings.Contains(key, "\"") {
					entries = append(entries, Entry{Path: rel, Kind: "key", Name: key, Line: i + 1, End: i + 1, Label: "key " + key})
				}
			}
		} else {
			for _, q := range tomlQuoted(trim) {
				entries = append(entries, Entry{Path: rel, Kind: "value", Name: q, Line: i + 1, End: i + 1, Label: "value " + q})
			}
		}
		depth += strings.Count(trim, "[") - strings.Count(trim, "]")
		if depth < 0 {
			depth = 0
		}
	}
	if len(entries) == 0 {
		return mapHeuristic(abs, rel)
	}
	return entries, nil
}

func mapYAML(abs, rel string) ([]Entry, error) {
	data, err := os.ReadFile(abs)
	if err != nil {
		return nil, err
	}
	text := string(data)
	if strings.Contains(rel, ".github/workflows/") {
		if entries := mapWorkflow(rel, text); len(entries) > 0 {
			return entries, nil
		}
	}
	var entries []Entry
	for i, line := range strings.Split(text, "\n") {
		if m := reYAMLKey.FindStringSubmatch(line); len(m) == 2 && !strings.HasPrefix(strings.TrimLeft(line, " \t"), "-") {
			// only indent 0-2 for top-ish keys
			indent := len(line) - len(strings.TrimLeft(line, " \t"))
			if indent <= 2 {
				entries = append(entries, Entry{Path: rel, Kind: "key", Name: m[1], Line: i + 1, End: i + 1, Label: "key " + m[1]})
			}
		}
		if len(entries) >= 80 {
			break
		}
	}
	if len(entries) == 0 {
		return mapHeuristic(abs, rel)
	}
	return entries, nil
}

func mapQML(abs, rel string) ([]Entry, error) {
	data, err := os.ReadFile(abs)
	if err != nil {
		return nil, err
	}
	text := string(data)
	var entries []Entry
	for _, m := range reQMLType.FindAllStringSubmatchIndex(text, -1) {
		name := text[m[2]:m[3]]
		line := 1 + strings.Count(text[:m[0]], "\n")
		entries = append(entries, Entry{Path: rel, Kind: "type", Name: name, Line: line, End: line + 15, Label: "type " + name})
	}
	for _, m := range reQMLFunc.FindAllStringSubmatchIndex(text, -1) {
		name := text[m[2]:m[3]]
		line := 1 + strings.Count(text[:m[0]], "\n")
		entries = append(entries, Entry{Path: rel, Kind: "function", Name: name, Line: line, End: line + 10, Label: "function " + name})
	}
	for _, m := range reQMLProp.FindAllStringSubmatchIndex(text, -1) {
		name := text[m[2]:m[3]]
		line := 1 + strings.Count(text[:m[0]], "\n")
		entries = append(entries, Entry{Path: rel, Kind: "property", Name: name, Line: line, End: line, Label: "property " + name})
	}
	if len(entries) == 0 {
		return mapHeuristic(abs, rel)
	}
	return entries, nil
}

var (
	reBashFunc1 = regexp.MustCompile(`(?m)^[ \t]*(?:function[ \t]+)?([A-Za-z_][A-Za-z0-9_]*)[ \t]*\([ \t]*\)[ \t]*\{`)
	reBashFunc2 = regexp.MustCompile(`(?m)^[ \t]*function[ \t]+([A-Za-z_][A-Za-z0-9_]*)[ \t]*\{`)
	reBashAlias = regexp.MustCompile(`(?m)^[ \t]*alias[ \t]+([A-Za-z_][A-Za-z0-9_]*)=`)
)

func mapBash(abs, rel string) ([]Entry, error) {
	data, err := os.ReadFile(abs)
	if err != nil {
		return nil, err
	}
	text := string(data)
	lines := strings.Split(text, "\n")
	var entries []Entry
	seen := map[string]bool{}
	add := func(kind, name string, line int) {
		key := kind + ":" + name + ":" + fmt.Sprint(line)
		if seen[key] {
			return
		}
		seen[key] = true
		end := line + 20
		if end > len(lines) {
			end = len(lines)
		}
		entries = append(entries, Entry{
			Path: rel, Kind: kind, Name: name, Line: line, End: end,
			Label: fmt.Sprintf("%s %s", kind, name),
		})
	}
	for i, line := range lines {
		ln := i + 1
		if m := reBashFunc1.FindStringSubmatch(line); len(m) == 2 {
			add("function", m[1], ln)
			continue
		}
		if m := reBashFunc2.FindStringSubmatch(line); len(m) == 2 {
			add("function", m[1], ln)
			continue
		}
		if m := reBashAlias.FindStringSubmatch(line); len(m) == 2 {
			add("alias", m[1], ln)
		}
	}
	if len(entries) == 0 {
		return mapHeuristic(abs, rel)
	}
	return entries, nil
}

func mapGo(abs, rel string) ([]Entry, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, abs, nil, parser.ParseComments)
	if err != nil {
		return mapHeuristic(abs, rel)
	}
	var entries []Entry
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Name == nil {
				continue
			}
			pos := fset.Position(d.Pos())
			end := fset.Position(d.End())
			name := d.Name.Name
			kind := "func"
			if d.Recv != nil {
				kind = "method"
				if rt := exprRecv(d); rt != "" {
					name = rt + "." + name
				}
			}
			entries = append(entries, Entry{
				Path: rel, Kind: kind, Name: name,
				Line: pos.Line, End: end.Line,
				Label: fmt.Sprintf("%s %s", kind, name),
			})
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					pos := fset.Position(s.Pos())
					end := fset.Position(s.End())
					entries = append(entries, Entry{
						Path: rel, Kind: "type", Name: s.Name.Name,
						Line: pos.Line, End: end.Line,
						Label: fmt.Sprintf("type %s", s.Name.Name),
					})
				case *ast.ValueSpec:
					for _, n := range s.Names {
						pos := fset.Position(n.Pos())
						end := fset.Position(n.End())
						kind := "var"
						if d.Tok == token.CONST {
							kind = "const"
						}
						entries = append(entries, Entry{
							Path: rel, Kind: kind, Name: n.Name,
							Line: pos.Line, End: end.Line,
							Label: fmt.Sprintf("%s %s", kind, n.Name),
						})
					}
				}
			}
		}
	}
	return entries, nil
}

var (
	reFuncTS  = regexp.MustCompile(`(?m)^(?:export\s+)?(?:async\s+)?function\s+([A-Za-z0-9_$]+)`)
	reClass   = regexp.MustCompile(`(?m)^(?:export\s+)?(?:abstract\s+)?class\s+([A-Za-z0-9_$]+)`)
	reDefPy   = regexp.MustCompile(`(?m)^(?:async\s+)?def\s+([A-Za-z0-9_]+)`)
	reClassPy = regexp.MustCompile(`(?m)^class\s+([A-Za-z0-9_]+)`)
	reFnRust  = regexp.MustCompile(`(?m)^(?:pub\s+)?(?:async\s+)?fn\s+([A-Za-z0-9_]+)`)
)

func exprRecv(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return ""
	}
	e := fn.Recv.List[0].Type
	for {
		switch t := e.(type) {
		case *ast.Ident:
			return t.Name
		case *ast.StarExpr:
			e = t.X
		default:
			return ""
		}
	}
}

func mapHeuristic(abs, rel string) ([]Entry, error) {
	data, err := os.ReadFile(abs)
	if err != nil {
		return nil, err
	}
	if strings.Contains(string(data), "\x00") {
		return nil, nil
	}
	text := string(data)
	lines := strings.Split(text, "\n")
	var entries []Entry
	add := func(kind, name string, line int) {
		end := line
		// crude: extend until blank line or next top-level-ish
		for i := line; i < len(lines); i++ {
			end = i + 1
			if i > line-1 && strings.TrimSpace(lines[i]) == "" && i+1 < len(lines) {
				// keep going a bit for short blocks
				if i > line+2 {
					break
				}
			}
		}
		// cap body window
		if end > line+80 {
			end = line + 80
		}
		entries = append(entries, Entry{
			Path: rel, Kind: kind, Name: name, Line: line, End: end,
			Label: fmt.Sprintf("%s %s", kind, name),
		})
	}
	for i, line := range lines {
		ln := i + 1
		if m := reFuncTS.FindStringSubmatch(line); len(m) == 2 {
			add("function", m[1], ln)
			continue
		}
		if m := reClass.FindStringSubmatch(line); len(m) == 2 {
			add("class", m[1], ln)
			continue
		}
		if m := reDefPy.FindStringSubmatch(line); len(m) == 2 {
			add("def", m[1], ln)
			continue
		}
		if m := reClassPy.FindStringSubmatch(line); len(m) == 2 {
			add("class", m[1], ln)
			continue
		}
		if m := reFnRust.FindStringSubmatch(line); len(m) == 2 {
			add("fn", m[1], ln)
			continue
		}
	}
	if len(entries) == 0 {
		// file-level placeholder
		entries = append(entries, Entry{
			Path: rel, Kind: "file", Name: filepath.Base(rel),
			Line: 1, End: min(len(lines), 40),
			Label: "file " + filepath.Base(rel),
		})
	}
	return entries, nil
}

func sliceLines(lines []string, start, end int) string {
	if start < 1 {
		start = 1
	}
	if end < start {
		end = start
	}
	if start > len(lines) {
		return ""
	}
	if end > len(lines) {
		end = len(lines)
	}
	return strings.Join(lines[start-1:end], "\n")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func truncateSnippet(s string, max int) string {
	if max <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "…"
}

func tomlQuoted(line string) []string {
	var out []string
	rest := line
	for {
		i := strings.Index(rest, "\"")
		if i < 0 {
			break
		}
		rest = rest[i+1:]
		j := strings.Index(rest, "\"")
		if j < 0 {
			break
		}
		out = append(out, rest[:j])
		rest = rest[j+1:]
	}
	return out
}

func mapWorkflow(rel, text string) []Entry {
	var entries []Entry
	lines := strings.Split(text, "\n")
	inJobs := false
	var job string
	for i, line := range lines {
		indent := len(line) - len(strings.TrimLeft(line, " "))
		trim := strings.TrimSpace(line)
		if indent == 0 && strings.HasPrefix(trim, "name:") {
			entries = append(entries, Entry{Path: rel, Kind: "workflow", Name: strings.TrimSpace(strings.TrimPrefix(trim, "name:")), Line: i + 1, End: i + 1, Label: "workflow " + strings.TrimSpace(strings.TrimPrefix(trim, "name:"))})
		}
		if indent == 0 && trim == "jobs:" {
			inJobs = true
			continue
		}
		if indent == 0 && trim != "jobs:" && strings.HasSuffix(trim, ":") && !strings.HasPrefix(trim, "-") {
			if trim != "jobs:" {
				inJobs = false
			}
		}
		if !inJobs {
			continue
		}
		if indent == 2 && strings.HasSuffix(trim, ":") && !strings.HasPrefix(trim, "-") {
			job = strings.TrimSuffix(trim, ":")
			entries = append(entries, Entry{Path: rel, Kind: "job", Name: job, Line: i + 1, End: i + 1, Label: "job " + job})
		}
		if job != "" && indent >= 4 && strings.HasPrefix(trim, "needs:") {
			for _, name := range splitList(strings.TrimSpace(strings.TrimPrefix(trim, "needs:"))) {
				entries = append(entries, Entry{Path: rel, Kind: "dependency", Name: job + "->" + name, Line: i + 1, End: i + 1, Label: "needs " + name})
			}
		}
		if job != "" && indent >= 4 && strings.HasPrefix(trim, "if:") {
			cond := strings.TrimSpace(strings.TrimPrefix(trim, "if:"))
			entries = append(entries, Entry{Path: rel, Kind: "condition", Name: job, Line: i + 1, End: i + 1, Label: "if " + cond})
		}
		if indent >= 4 && strings.HasPrefix(trim, "- name:") {
			step := strings.TrimSpace(strings.TrimPrefix(trim, "- name:"))
			entries = append(entries, Entry{Path: rel, Kind: "step", Name: job + "/" + step, Line: i + 1, End: i + 1, Label: "step " + step})
		}
	}
	return entries
}

func splitList(rest string) []string {
	rest = strings.Trim(rest, "[] ")
	if rest == "" {
		return nil
	}
	var out []string
	for _, part := range strings.Split(rest, ",") {
		part = strings.Trim(part, " \"'")
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
