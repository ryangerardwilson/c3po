package config

import "os"

// Merge loads defaults, then the global file, then the nearest project file
// that is not above the git root. explicit replaces the project search.
func Merge(cwd, explicit string) (Config, error) {
	cfg := Default()
	path, err := GlobalPath()
	if err != nil {
		return cfg, err
	}
	if _, err := os.Stat(path); err == nil {
		cfg, err = Load(path, cfg)
		if err != nil {
			return cfg, err
		}
	}
	if explicit != "" {
		return Load(explicit, cfg)
	}
	stop := cwd
	if root, ok := GitRoot(cwd); ok {
		stop = root
	}
	project, err := FindProject(cwd, stop)
	if err != nil || project == "" {
		return cfg, err
	}
	return Load(project, cfg)
}

// Configured reports whether a global or project file is present.
func Configured(cwd string) bool {
	if path, err := GlobalPath(); err == nil {
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}
	stop := cwd
	if root, ok := GitRoot(cwd); ok {
		stop = root
	}
	project, err := FindProject(cwd, stop)
	return err == nil && project != ""
}
