package config

import (
	"os"
	"path/filepath"
)

// FindProject walks from start up to stop, inclusive, and returns the nearest
// complexity.toml or .complexity.toml. stop is usually the git root.
func FindProject(start, stop string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	if stop == "" {
		stop = string(filepath.Separator)
	}
	stop, err = filepath.Abs(stop)
	if err != nil {
		return "", err
	}
	for {
		if found := namedConfig(dir); found != "" {
			return found, nil
		}
		if dir == stop || dir == string(filepath.Separator) {
			return "", nil
		}
		next := filepath.Dir(dir)
		if next == dir {
			return "", nil
		}
		dir = next
	}
}

func namedConfig(dir string) string {
	for _, name := range []string{"complexity.toml", ".complexity.toml"} {
		p := filepath.Join(dir, name)
		st, err := os.Stat(p)
		if err == nil && !st.IsDir() {
			return p
		}
	}
	return ""
}

// GitRoot returns the nearest directory at or above start that contains .git.
func GitRoot(start string) (string, bool) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", false
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, true
		}
		next := filepath.Dir(dir)
		if next == dir {
			return "", false
		}
		dir = next
	}
}
