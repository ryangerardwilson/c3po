package ast

import (
	"github.com/ryangerardwilson/c3po/ast/internal/config"
	"github.com/ryangerardwilson/c3po/ast/internal/trace"
)

// Fn is one function in the call index.
type Fn struct {
	ID, Path, Name, Kind string
	Line                 int
}

// Calls is the call index for one directory.
type Calls struct {
	g *trace.Graph
}

// OpenCalls indexes the calls under root.
func OpenCalls(root string) (*Calls, error) {
	g, err := trace.Open(root, config.Default())
	if err != nil {
		return nil, err
	}
	return &Calls{g: g}, nil
}

// Resolve returns every function named by query.
func (c *Calls) Resolve(query string) []Fn {
	return copyFns(c.g.Resolve(query))
}

// Callees returns the resolved direct calls from id, one per target.
func (c *Calls) Callees(id string) []Fn {
	return copyFns(c.g.Callees(id))
}

func copyFns(in []trace.Hit) []Fn {
	out := make([]Fn, len(in))
	for i, h := range in {
		out[i] = Fn{ID: h.ID, Path: h.Path, Name: h.Name, Kind: h.Kind, Line: h.Line}
	}
	return out
}
