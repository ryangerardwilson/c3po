package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/ryangerardwilson/c3po/complexity/internal/metric"
)

const (
	DefaultCognitive  = 15
	DefaultNesting    = 4
	DefaultLines      = 80
	DefaultParams     = 6
	DefaultCyclomatic = 10
	DefaultDegree     = 10
	DefaultDepth      = 5
	DefaultM2M        = 4
	DefaultTop        = 20
	DefaultDupLines   = 6
	DefaultMode       = "auto"
	DefaultBase       = "main"
	DefaultDataColumn = "data"
	DefaultLogSuffix  = "logs"
)

// Config is the merged complexity policy.
type Config struct {
	Version     int         `json:"version" toml:"version"`
	Thresholds  Thresholds  `json:"thresholds" toml:"thresholds"`
	Exclude     Exclude     `json:"exclude" toml:"exclude"`
	Languages   Languages   `json:"languages" toml:"languages"`
	Output      Output      `json:"output" toml:"output"`
	Git         Git         `json:"git" toml:"git"`
	Conventions Conventions `json:"conventions" toml:"conventions"`
	Enforcement Enforcement `json:"enforcement" toml:"enforcement"`
	Duplication Duplication `json:"duplication" toml:"duplication"`
}

// Thresholds holds function and model limits.
type Thresholds struct {
	Cognitive  int             `json:"cognitive" toml:"cognitive"`
	Nesting    int             `json:"nesting" toml:"nesting"`
	Lines      int             `json:"lines" toml:"lines"`
	Params     int             `json:"params" toml:"params"`
	Cyclomatic int             `json:"cyclomatic" toml:"cyclomatic"`
	Models     ModelThresholds `json:"models" toml:"models"`
}

// ModelThresholds holds relationship limits.
type ModelThresholds struct {
	Degree int `json:"degree" toml:"degree"`
	Depth  int `json:"depth" toml:"depth"`
	M2M    int `json:"m2m" toml:"m2m"`
}

// Exclude lists globs. Tests is dropped when include_tests is set.
type Exclude struct {
	Patterns     []string `json:"patterns" toml:"patterns"`
	Tests        []string `json:"tests" toml:"tests"`
	IncludeTests bool     `json:"include_tests" toml:"include_tests"`
}

// Languages selects which languages to analyze. Empty means all.
type Languages struct {
	Include []string `json:"include" toml:"include"`
}

// Output controls the default rendering.
type Output struct {
	Format       string `json:"format" toml:"format"`
	Top          int    `json:"top" toml:"top"`
	HotspotsOnly bool   `json:"hotspots_only" toml:"hotspots_only"`
}

// Git holds the default comparison ref.
type Git struct {
	DefaultBase string `json:"default_base" toml:"default_base"`
}

// Conventions controls naming and column order.
type Conventions struct {
	Naming                bool   `json:"naming" toml:"naming"`
	ColumnOrder           bool   `json:"column_order" toml:"column_order"`
	DataColumnName        string `json:"data_column_name" toml:"data_column_name"`
	LogSuffix             string `json:"log_suffix" toml:"log_suffix"`
	RequireFeatureMapping bool   `json:"require_feature_mapping" toml:"require_feature_mapping"`
	Strict                bool   `json:"strict" toml:"strict"`
}

// Enforcement controls the agent hook.
type Enforcement struct {
	Mode   string `json:"mode" toml:"mode"`
	OnStop bool   `json:"on_stop" toml:"on_stop"`
}

// Duplication controls clone detection.
type Duplication struct {
	Enabled  bool `json:"enabled" toml:"enabled"`
	MinLines int  `json:"min_lines" toml:"min_lines"`
}

// FuncThresholds converts the function limits.
func (c Config) FuncThresholds() metric.Thresholds {
	return metric.Thresholds{
		Cognitive:  c.Thresholds.Cognitive,
		Nesting:    c.Thresholds.Nesting,
		Lines:      c.Thresholds.Lines,
		Params:     c.Thresholds.Params,
		Cyclomatic: c.Thresholds.Cyclomatic,
	}
}

// Default is the built-in policy.
func Default() Config {
	return Config{
		Version: 1,
		Thresholds: Thresholds{
			Cognitive:  DefaultCognitive,
			Nesting:    DefaultNesting,
			Lines:      DefaultLines,
			Params:     DefaultParams,
			Cyclomatic: DefaultCyclomatic,
			Models: ModelThresholds{
				Degree: DefaultDegree,
				Depth:  DefaultDepth,
				M2M:    DefaultM2M,
			},
		},
		Exclude:     Exclude{Patterns: append([]string{}, defaultPatterns...), Tests: append([]string{}, defaultTests...)},
		Languages:   Languages{Include: []string{"python", "go", "rust", "typescript"}},
		Output:      Output{Format: "text", Top: DefaultTop},
		Git:         Git{DefaultBase: DefaultBase},
		Conventions: defaultConventions(),
		Enforcement: Enforcement{Mode: DefaultMode, OnStop: true},
		Duplication: Duplication{Enabled: true, MinLines: DefaultDupLines},
	}
}

func defaultConventions() Conventions {
	return Conventions{
		Naming:                true,
		ColumnOrder:           true,
		DataColumnName:        DefaultDataColumn,
		LogSuffix:             DefaultLogSuffix,
		RequireFeatureMapping: true,
		Strict:                false,
	}
}

// GlobalPath is $XDG_CONFIG_HOME/complexity/settings.json, or
// ~/.config/complexity/settings.json when XDG_CONFIG_HOME is unset.
func GlobalPath() (string, error) {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "complexity", "settings.json"), nil
}

// Load reads a TOML or JSON file over base. Missing keys keep base values.
func Load(path string, base Config) (Config, error) {
	cfg := base
	if err := decode(path, &cfg); err != nil {
		return base, configErr(path, err)
	}
	return fillDefaults(cfg), nil
}

func shownConfig(path string) string {
	cwd, err := os.Getwd()
	if err != nil {
		return path
	}
	return Relative(cwd, path)
}

func configErr(path string, err error) error {
	shown := shownConfig(path)
	msg := err.Error()
	if path != "" && shown != path {
		msg = strings.ReplaceAll(msg, path, shown)
	}
	return fmt.Errorf("config %s: %s", shown, msg)
}

func decode(path string, cfg *Config) error {
	if strings.HasSuffix(strings.ToLower(path), ".json") {
		return overlayJSON(cfg, path)
	}
	_, err := toml.DecodeFile(path, cfg)
	return err
}

func fillDefaults(cfg Config) Config {
	if cfg.Output.Top <= 0 {
		cfg.Output.Top = DefaultTop
	}
	if cfg.Duplication.MinLines <= 0 {
		cfg.Duplication.MinLines = DefaultDupLines
	}
	if cfg.Git.DefaultBase == "" {
		cfg.Git.DefaultBase = DefaultBase
	}
	if cfg.Conventions.DataColumnName == "" {
		cfg.Conventions.DataColumnName = DefaultDataColumn
	}
	if cfg.Conventions.LogSuffix == "" {
		cfg.Conventions.LogSuffix = DefaultLogSuffix
	}
	if cfg.Enforcement.Mode == "" {
		cfg.Enforcement.Mode = DefaultMode
	}
	return cfg
}

// Ensure writes the default global settings file when it is missing.
func Ensure(path string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return writeJSON(path, Default())
}
