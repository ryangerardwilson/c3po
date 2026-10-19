package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/ryangerardwilson/c3po/internal/xdg"
)

const (
	DefaultThresholdLOC = 5000
	DefaultMode         = "auto"
	DefaultMaxReadLines = 200
	DefaultMaxSearch    = 50
	DefaultMaxMap       = 200
)

// Config is the sole global policy for rgw-ast.
type Config struct {
	Version      int      `toml:"version"`
	ThresholdLOC int      `toml:"threshold_loc"`
	Include      []string `toml:"include"`
	Exclude      []string `toml:"exclude"`
	// Prose files are measured as prose_loc and do not reach threshold_loc
	// unless enforcement.count_prose is set.
	Prose         []string    `toml:"prose"`
	Enforcement   Enforcement `toml:"enforcement"`
	Cache         Cache       `toml:"cache"`
	Generators    Generators  `toml:"generators"`
	MaxMapEntries int         `toml:"max_map_entries"`
}

// Generators configures trusted external scaffolding tools.
type Generators struct {
	// Allow is a list of prefix or glob-like command patterns that may run
	// under `rgw-ast exec` (e.g. "npm exec -- openspec", "openspec ").
	Allow []string `toml:"allow"`
}

type Enforcement struct {
	Mode                   string `toml:"mode"`
	DenyWholeFileRead      bool   `toml:"deny_whole_file_read"`
	RequireHashBeforePatch bool   `toml:"require_hash_before_patch"`
	MaxReadLines           int    `toml:"max_read_lines"`
	MaxSearchHits          int    `toml:"max_search_hits"`
	CountProse             bool   `toml:"count_prose"`
}

type Cache struct {
	Dir string `toml:"dir"`
}

// Default returns the built-in global defaults.
func Default() Config {
	return Config{
		Version:      1,
		ThresholdLOC: DefaultThresholdLOC,
		Include: []string{
			"**/*.{ts,tsx,js,jsx,mjs,cjs}",
			"**/*.{py,go,rs,java,kt,swift}",
			"**/*.{c,cc,cpp,h,hpp}",
			"**/*.{rb,php,cs}",
			"**/*.{sh,bash}",
			"**/.bashrc",
			"**/.bash_profile",
			"**/.profile",
			"**/*.{toml,yml,yaml,json}",
			"**/*.qml",
			"**/.gitignore",
			"**/*.md",
		},
		Exclude: []string{
			"**/node_modules/**",
			"**/dist/**",
			"**/build/**",
			"**/.git/**",
			"**/vendor/**",
			"**/.next/**",
			"**/target/**",
			"**/.cache/**",
			"**/coverage/**",
			"**/.venv/**",
			"**/venv/**",
			"**/__pycache__/**",
			"**/.turbo/**",
			"**/.bun/**",
		},
		Prose: []string{
			"**/*.md",
			"**/*.markdown",
			"**/*.mdx",
			"**/*.rst",
			"**/*.txt",
		},
		Enforcement: Enforcement{
			Mode:                   DefaultMode,
			DenyWholeFileRead:      true,
			RequireHashBeforePatch: true,
			MaxReadLines:           DefaultMaxReadLines,
			MaxSearchHits:          DefaultMaxSearch,
		},
		Cache: Cache{
			Dir: "",
		},
		Generators: Generators{
			// Token-prefix rules matched from argv[0] (not substrings of the full line).
			Allow: []string{
				"npm exec -- openspec",
				"npm exec openspec",
				"npx openspec",
				"openspec",
				"go generate",
			},
		},
		MaxMapEntries: DefaultMaxMap,
	}
}

// Dir returns the global config directory.
func Dir() (string, error) {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".config")
	}
	dest := xdg.App(base, "ast")
	if err := xdg.Adopt(dest, filepath.Join(base, "rgw-ast")); err != nil {
		return "", err
	}
	return dest, nil
}

// Path returns the absolute path to config.toml.
func Path() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.toml"), nil
}

// CacheDir resolves the cache directory (expanded ~).
func (c Config) CacheDir() (string, error) {
	dir := strings.TrimSpace(c.Cache.Dir)
	if dir == "" {
		if testing.Testing() {
			return filepath.Join(os.TempDir(), "c3po-ast-cache-test"), nil
		}
		base := os.Getenv("XDG_CACHE_HOME")
		if base == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			base = filepath.Join(home, ".cache")
		}
		dest := xdg.App(base, "ast")
		if err := xdg.Adopt(dest, filepath.Join(base, "rgw-ast")); err != nil {
			return "", err
		}
		return dest, nil
	}
	if strings.HasPrefix(dir, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, dir[2:]), nil
	}
	return dir, nil
}

// Load ensures the global config exists and returns it with the path used.
func Load() (Config, string, error) {
	path, err := Path()
	if err != nil {
		return Config{}, "", err
	}
	if err := Ensure(path); err != nil {
		return Config{}, "", err
	}
	cfg, err := Read(path)
	if err != nil {
		return Config{}, "", err
	}
	return cfg, path, nil
}

// Ensure creates default config.toml if missing.
func Ensure(path string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return Write(path, Default())
}

// Read parses a config file.
func Read(path string) (Config, error) {
	cfg := Default()
	meta, err := toml.DecodeFile(path, &cfg)
	if err != nil {
		return Config{}, err
	}
	_ = meta
	normalize(&cfg)
	if err := validate(cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Write writes config as TOML.
func Write(path string, cfg Config) error {
	normalize(&cfg)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := toml.NewEncoder(f)
	return enc.Encode(cfg)
}

func normalize(cfg *Config) {
	if cfg.Version == 0 {
		cfg.Version = 1
	}
	if cfg.ThresholdLOC <= 0 {
		cfg.ThresholdLOC = DefaultThresholdLOC
	}
	if cfg.MaxMapEntries <= 0 {
		cfg.MaxMapEntries = DefaultMaxMap
	}
	if cfg.Enforcement.Mode == "" {
		cfg.Enforcement.Mode = DefaultMode
	}
	cfg.Enforcement.Mode = strings.ToLower(strings.TrimSpace(cfg.Enforcement.Mode))
	if cfg.Enforcement.MaxReadLines <= 0 {
		cfg.Enforcement.MaxReadLines = DefaultMaxReadLines
	}
	if cfg.Enforcement.MaxSearchHits <= 0 {
		cfg.Enforcement.MaxSearchHits = DefaultMaxSearch
	}
	if len(cfg.Include) == 0 {
		d := Default()
		cfg.Include = d.Include
	}
	if len(cfg.Exclude) == 0 {
		d := Default()
		cfg.Exclude = d.Exclude
	}
	if len(cfg.Generators.Allow) == 0 {
		d := Default()
		cfg.Generators.Allow = d.Generators.Allow
	}
}

func validate(cfg Config) error {
	switch cfg.Enforcement.Mode {
	case "auto", "always", "never":
		return nil
	default:
		return fmt.Errorf("invalid enforcement.mode %q (want auto|always|never)", cfg.Enforcement.Mode)
	}
}
