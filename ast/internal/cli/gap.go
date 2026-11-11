package cli

import (
	"errors"
	"fmt"
	"os"
	"sort"

	"github.com/ryangerardwilson/c3po/ast/internal/config"
	"github.com/ryangerardwilson/c3po/ast/internal/measure"
	"github.com/ryangerardwilson/c3po/cyclotree/score"
)

// errGapOver is gap --check with a function at or above max_cc. Run exits 1
// and prints no Error line; the report is the output.
var errGapOver = errors.New("functions at or above max_cc")

// gapFn is one scored function at or above max_cc.
type gapFn struct {
	Path       string `json:"path"`
	Name       string `json:"name"`
	Language   string `json:"language"`
	Line       int    `json:"line"`
	Cyclomatic int    `json:"cyclomatic"`
}

type gapReport struct {
	Root      string  `json:"root"`
	MaxCC     int     `json:"max_cc"`
	Over      int     `json:"over"`
	Functions []gapFn `json:"functions"`
}

const gapUse = "Use: c3po ast gap [--json] [--check]"

func (r Runner) cmdGap(args []string) error {
	asJSON, check, err := parseGapArgs(args)
	if err != nil {
		return err
	}
	cfg, _, wsRoot, err := r.workspace()
	if err != nil {
		return err
	}
	maxCC := ccMax(cfg.MaxCC)
	root, fns, err := gapScan(cfg, wsRoot, maxCC)
	if err != nil {
		return err
	}
	if asJSON {
		err = writeJSON(r.Out, gapReport{Root: root, MaxCC: maxCC, Over: len(fns), Functions: fns})
	} else {
		r.printGap(root, maxCC, fns)
	}
	if err == nil && check && len(fns) > 0 {
		return errGapOver
	}
	return err
}

func parseGapArgs(args []string) (asJSON, check bool, err error) {
	for _, a := range args {
		switch a {
		case "--json":
			asJSON = true
		case "--check":
			check = true
		default:
			return false, false, usageError{gapUse}
		}
	}
	return asJSON, check, nil
}

// gapScan scores every file the measure scope walks and keeps the functions
// whose cyclomatic number is at or above maxCC, highest first, then by path
// and name.
func gapScan(cfg config.Config, wsRoot string, maxCC int) (string, []gapFn, error) {
	sc := measure.NewScope(wsRoot, cfg)
	fns := []gapFn{}
	_, err := sc.Walk(sc.Root(), -1, func(abs, rel string) error {
		if score.Language(rel) == "" {
			return nil
		}
		src, err := os.ReadFile(abs)
		if err != nil {
			return err
		}
		for _, fn := range score.File(rel, src) {
			if fn.Cyclomatic >= maxCC {
				fns = append(fns, gapFn{Path: rel, Name: fn.Name, Language: fn.Language, Line: fn.Line, Cyclomatic: fn.Cyclomatic})
			}
		}
		return nil
	})
	if err != nil {
		return "", nil, err
	}
	sort.Slice(fns, func(i, j int) bool { return gapLess(fns[i], fns[j]) })
	return sc.Root(), fns, nil
}

func gapLess(a, b gapFn) bool {
	if a.Cyclomatic != b.Cyclomatic {
		return a.Cyclomatic > b.Cyclomatic
	}
	if a.Path != b.Path {
		return a.Path < b.Path
	}
	return a.Name < b.Name
}

func (r Runner) printGap(root string, maxCC int, fns []gapFn) {
	_, _ = fmt.Fprintf(r.Out, "gap  %s\nmax_cc  %d\nover  %d\n", root, maxCC, len(fns))
	for _, fn := range fns {
		_, _ = fmt.Fprintf(r.Out, "%d  %s:%s  %s\n", fn.Cyclomatic, fn.Path, fn.Name, fn.Language)
	}
}
