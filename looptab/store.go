package looptab

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
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

func schedulePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "looptab", "looptab"), nil
}

func loadCatalog(path string) (catalog, error) {
	body, err := sqlxdb.Read("looptab", "files", "files", path)
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

func saveCatalog(path string, cat catalog) error {
	if cat.Files == nil {
		cat.Files = []registered{}
	}
	body, err := json.MarshalIndent(cat, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	return sqlxdb.Write("looptab", "files", "files", path, body, 0o600)
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

func openVim(path string) error {
	bin, err := exec.LookPath("vim")
	if err != nil {
		return errors.New("vim is not on PATH")
	}
	cmd := exec.Command(bin, path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
