package looptab

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/ryangerardwilson/c3po/internal/sqlxdb"
)

type registered struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type catalog struct {
	Files []registered `json:"files"`
}

func recordPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	base := filepath.Join(home, ".local", "share")
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" && os.Getenv("HOME") == home {
		base = xdg
	}
	return filepath.Join(base, "c3po", "looptab", "files.json"), nil
}

// configDir is <config>/c3po/looptab, where <config> is XDG_CONFIG_HOME when it
// is absolute and ~/.config otherwise. The scheduler file, the Looptab
// timezone file config.json, and the jobs files sit there.
func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	base := filepath.Join(home, ".config")
	if xdg := os.Getenv("XDG_CONFIG_HOME"); filepath.IsAbs(xdg) {
		base = xdg
	}
	return filepath.Join(base, "c3po", "looptab"), nil
}

// schedulePath is the scheduler file c3po installs. The looptab binary opens
// ~/.config/looptab/looptab, which the workstation links to this directory.
func schedulePath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "looptab"), nil
}

// loadCatalog reads the registered jobs files in the order they were added.
// A production record path reads looptab_file; another path is a JSON file.
func loadCatalog(path string) (catalog, error) {
	if sqlxdb.UsesDB(path) {
		var rows []registered
		if err := sqlxdb.Query("SELECT name, path FROM looptab_file ORDER BY id", &rows); err != nil {
			return catalog{}, err
		}
		if rows == nil {
			rows = []registered{}
		}
		return catalog{Files: rows}, nil
	}
	body, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return catalog{Files: []registered{}}, nil
	}
	if err != nil {
		return catalog{}, err
	}
	var cat catalog
	if err := json.Unmarshal(body, &cat); err != nil {
		return catalog{}, err
	}
	if cat.Files == nil {
		cat.Files = []registered{}
	}
	return cat, nil
}

// saveCatalog makes the stored jobs files match cat in one transaction. A
// file that stays keeps its row.
func saveCatalog(path string, cat catalog) error {
	if cat.Files == nil {
		cat.Files = []registered{}
	}
	if sqlxdb.UsesDB(path) {
		names := []string{"''"}
		var b strings.Builder
		for _, file := range cat.Files {
			names = append(names, sqlxdb.Quote(file.Name))
			b.WriteString(";\nINSERT INTO looptab_file(name, path) VALUES (" + sqlxdb.Quote(file.Name) + ", " +
				sqlxdb.Quote(file.Path) + ") ON CONFLICT(name) DO UPDATE SET path = excluded.path")
		}
		return sqlxdb.Exec("DELETE FROM looptab_file WHERE name NOT IN (" + strings.Join(names, ", ") + ")" + b.String())
	}
	body, err := json.MarshalIndent(cat, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, body, 0o600)
}

func installJobs(schedule string, files []registered) error {
	var b strings.Builder
	b.WriteString("# Installed by c3po looptab from registered jobs files.\n")
	if len(files) == 0 {
		b.WriteString("\n")
	}
	for _, file := range files {
		body, err := os.ReadFile(file.Path)
		if err != nil {
			return err
		}
		b.WriteString("\n")
		b.Write(body)
		if len(body) == 0 || body[len(body)-1] != '\n' {
			b.WriteByte('\n')
		}
	}
	if err := os.MkdirAll(filepath.Dir(schedule), 0o755); err != nil {
		return err
	}
	return os.WriteFile(schedule, []byte(b.String()), 0o644)
}
