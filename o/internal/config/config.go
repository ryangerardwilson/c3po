package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/ryangerardwilson/c3po/internal/xdg"
)

type Config struct {
	MatrixMode bool `json:"matrix_mode"`
}

func Path() string {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base != "" {
		base = expand(base)
	} else {
		home, err := os.UserHomeDir()
		if err != nil || home == "" {
			home = "."
		}
		base = filepath.Join(home, ".config")
	}
	dest := xdg.App(base, "o")
	_ = xdg.Adopt(dest, filepath.Join(base, "o"))
	return filepath.Join(dest, "config.json")
}

func Load() Config {
	body, err := os.ReadFile(Path())
	if err != nil {
		return Config{}
	}
	var cfg Config
	if err := json.Unmarshal(body, &cfg); err != nil {
		return Config{}
	}
	return cfg
}

func Ensure() error {
	path := Path()
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	body, err := json.MarshalIndent(Config{}, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	return os.WriteFile(path, body, 0o644)
}

func expand(path string) string {
	if path == "~" || len(path) > 2 && path[:2] == "~/" {
		home, err := os.UserHomeDir()
		if err == nil && home != "" {
			if path == "~" {
				return home
			}
			return filepath.Join(home, path[2:])
		}
	}
	return os.ExpandEnv(path)
}
