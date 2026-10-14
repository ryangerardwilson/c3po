package config

import "path/filepath"

// Relative reports path relative to dir. Absolute and relative inputs are both rewritten.
func Relative(dir, path string) string {
	if path == "" {
		return ""
	}
	if dir == "" {
		return filepath.ToSlash(path)
	}
	target := path
	if !filepath.IsAbs(path) {
		target = filepath.Join(dir, path)
	}
	rel, err := filepath.Rel(dir, target)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}
