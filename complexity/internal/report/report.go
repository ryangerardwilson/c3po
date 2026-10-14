package report

import (
	"io"
	"time"

	"github.com/ryangerardwilson/c3po/complexity/internal/change"
	"github.com/ryangerardwilson/c3po/complexity/internal/dup"
	"github.com/ryangerardwilson/c3po/complexity/internal/metric"
	"github.com/ryangerardwilson/c3po/complexity/internal/models"
)

const JSONVersion = "0.1"

// Summary is the function rollup in JSON reports.
type Summary struct {
	FunctionsAnalyzed     int `json:"functions_analyzed"`
	Hotspots              int `json:"hotspots"`
	P90Cognitive          int `json:"p90_cognitive"`
	P95Cognitive          int `json:"p95_cognitive"`
	MaxCognitive          int `json:"max_cognitive"`
	MaxNesting            int `json:"max_nesting"`
	LongFunctions         int `json:"long_functions"`
	DuplicationClusters   int `json:"duplication_clusters"`
	DuplicatedLinesApprox int `json:"duplicated_lines_approx"`
}

// Document is the machine-readable report.
type Document struct {
	Version         string            `json:"version"`
	Timestamp       string            `json:"timestamp"`
	Summary         Summary           `json:"summary"`
	Hotspots        []metric.Function `json:"hotspots"`
	Ranked          []metric.Function `json:"ranked,omitempty"`
	Duplication     []dup.Cluster     `json:"duplication,omitempty"`
	Thresholds      metric.Thresholds `json:"thresholds"`
	ViolationsFound bool              `json:"violations_found"`
	Diagnostics     []string          `json:"diagnostics,omitempty"`
	Models          *models.Section   `json:"models,omitempty"`
	// Ratchet holds the change-first facts when the run compares a baseline.
	Ratchet *change.Report `json:"ratchet,omitempty"`
}

// View controls rendering.
type View struct {
	Format       string
	Top          int
	HotspotsOnly bool
	Quiet        bool
	ShowExit     bool
	ExitCode     int
	RankBy       string
}

// Input is the scored data a report is built from.
type Input struct {
	Functions   []metric.Function
	Hotspots    []metric.Function
	Clusters    []dup.Cluster
	Thresholds  metric.Thresholds
	Diagnostics []string
	Models      *models.Section
	Failed      bool
	Ratchet     *change.Report
}

// Build fills summary fields from the analyzed functions.
func Build(in Input) Document {
	doc := Document{
		Version: JSONVersion, Timestamp: time.Now().UTC().Format(time.RFC3339),
		Hotspots: in.Hotspots, Duplication: in.Clusters, Thresholds: in.Thresholds,
		Diagnostics: in.Diagnostics, ViolationsFound: in.Failed, Ratchet: in.Ratchet,
	}
	if doc.Ratchet != nil && doc.Ratchet.Changes == nil {
		doc.Ratchet.Changes = []change.Fact{}
	}
	if doc.Hotspots == nil {
		doc.Hotspots = []metric.Function{}
	}
	doc.Summary = summarize(in.Functions, in.Hotspots, in.Clusters, in.Thresholds)
	if in.Models != nil {
		cp := *in.Models
		cp.Models = nil
		doc.Models = &cp
	}
	return doc
}

func summarize(fns, hotspots []metric.Function, clusters []dup.Cluster, th metric.Thresholds) Summary {
	s := Summary{FunctionsAnalyzed: len(fns), Hotspots: len(hotspots), DuplicationClusters: len(clusters)}
	cogs := make([]int, 0, len(fns))
	limit := th.Lines
	if limit <= 0 {
		limit = 80
	}
	for _, fn := range fns {
		cogs = append(cogs, fn.Cognitive)
		if fn.Cognitive > s.MaxCognitive {
			s.MaxCognitive = fn.Cognitive
		}
		if fn.Nesting > s.MaxNesting {
			s.MaxNesting = fn.Nesting
		}
		if fn.Lines > limit {
			s.LongFunctions++
		}
	}
	s.P90Cognitive = metric.Percentile(cogs, 90)
	s.P95Cognitive = metric.Percentile(cogs, 95)
	for _, c := range clusters {
		s.DuplicatedLinesApprox += c.Lines
	}
	return s
}

// Write renders doc in format to w.
func Write(w io.Writer, doc Document, fns []metric.Function, view View) error {
	switch view.Format {
	case "json":
		return writeJSON(w, doc)
	case "sarif":
		return writeSARIF(w, doc)
	case "csv":
		return writeCSV(w, fns, view.HotspotsOnly)
	case "html":
		return writeHTML(w, doc, fns)
	default:
		return writeText(w, doc, fns, view)
	}
}
