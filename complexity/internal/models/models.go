package models

// Column is one stored field in declaration order.
type Column struct {
	Name string `json:"name"`
	Role string `json:"role"`
	Line int    `json:"line"`
}

// Relationship is one edge declared on a model.
type Relationship struct {
	Name   string `json:"name"`
	Target string `json:"target"`
	Type   string `json:"type"`
}

// Model is one ORM model or SQL table after graph scoring.
type Model struct {
	Name          string         `json:"name"`
	File          string         `json:"file"`
	Line          int            `json:"line"`
	Degree        int            `json:"degree"`
	Depth         int            `json:"depth"`
	M2M           int            `json:"m2m"`
	Cycles        bool           `json:"cycles"`
	FanIn         int            `json:"fan_in"`
	FanOut        int            `json:"fan_out"`
	Relationships []Relationship `json:"relationships,omitempty"`
	Columns       []Column       `json:"columns,omitempty"`
	Violations    []string       `json:"violations,omitempty"`
	Suggestions   []string       `json:"suggestions,omitempty"`
	Warnings      []string       `json:"warnings,omitempty"`
}

// Summary is the schema-level rollup.
type Summary struct {
	ModelsAnalyzed int `json:"models_analyzed"`
	HighDegree     int `json:"high_degree"`
	MaxDepth       int `json:"max_depth"`
	Cycles         int `json:"cycles"`
	TotalM2M       int `json:"total_m2m"`
}

// Section is the models payload embedded in a report.
type Section struct {
	Summary  Summary `json:"summary"`
	Hotspots []Model `json:"hotspots"`
	Models   []Model `json:"models,omitempty"`
}

// Thresholds are the structural limits. Zero disables a numeric limit.
// A cycle is always a violation.
type Thresholds struct {
	Degree int
	Depth  int
	M2M    int
}

// Conventions controls naming and column order.
type Conventions struct {
	Naming                bool
	ColumnOrder           bool
	DataColumnName        string
	LogSuffix             string
	RequireFeatureMapping bool
	Strict                bool
}

type draft struct {
	name    string
	table   string
	file    string
	line    int
	columns []Column
	rels    []Relationship
}
