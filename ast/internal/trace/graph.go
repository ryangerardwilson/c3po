package trace

import (
	"sort"

	"github.com/ryangerardwilson/c3po/ast/internal/config"
)

// Hit is one function the call index can name.
type Hit struct {
	ID, Path, Name, Kind string
	Line                 int
}

// Graph is the call index for one directory.
type Graph struct {
	ix *index
}

// Open indexes the calls under root.
func Open(root string, cfg config.Config) (*Graph, error) {
	ix, err := buildIndex(root, cfg)
	if err != nil {
		return nil, err
	}
	return &Graph{ix: ix}, nil
}

// Resolve returns every function named by query. A query of path:name
// narrows the file. The order is name, then path, then line.
func (g *Graph) Resolve(query string) []Hit {
	hits := g.ix.match(query)
	out := make([]Hit, len(hits))
	for i, s := range hits {
		out[i] = hitOf(s)
	}
	sortHits(out)
	return out
}

// Callees returns the resolved direct calls from id, one per target.
func (g *Graph) Callees(id string) []Hit {
	seen := map[string]bool{}
	var out []Hit
	for _, e := range g.ix.edges {
		if e.fromID != id || e.toID == "" || seen[e.toID] {
			continue
		}
		target, ok := g.ix.byID[e.toID]
		if !ok {
			continue
		}
		seen[e.toID] = true
		out = append(out, hitOf(target))
	}
	sortHits(out)
	return out
}

func hitOf(s symbol) Hit {
	return Hit{ID: s.id, Path: s.path, Name: s.name, Kind: s.kind, Line: s.line}
}

func sortHits(h []Hit) {
	sort.Slice(h, func(i, j int) bool {
		if h[i].Name != h[j].Name {
			return h[i].Name < h[j].Name
		}
		if h[i].Path != h[j].Path {
			return h[i].Path < h[j].Path
		}
		return h[i].Line < h[j].Line
	})
}
