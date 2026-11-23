package sqlxdb

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// The application owns this sqlite file. Nebula sql is the Postgres
// interface and is not called. nebula keep backs the file up (keep.go).

func dbFile(name string) string {
	if root := os.Getenv("SQLX_ROOT"); root != "" {
		return filepath.Join(root, "data", name+".sqlite")
	}
	if v := os.Getenv("SQLX_RUNTIME"); v != "" {
		return filepath.Join(v, name+".sqlite")
	}
	base := os.Getenv("XDG_RUNTIME_DIR")
	if base == "" {
		base = filepath.Join("/dev/shm", fmt.Sprintf("%s-%d", name, os.Getuid()))
	} else {
		base = filepath.Join(base, name)
	}
	return filepath.Join(base, name+".sqlite")
}

func prepareAppDB(path, name string) error {
	if root := os.Getenv("SQLX_ROOT"); root != "" {
		if err := os.MkdirAll(root, 0o700); err != nil {
			return err
		}
		if err := os.Chmod(root, 0o700); err != nil {
			return err
		}
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return err
	}
	if err := loadMissing(path, name); err != nil {
		return err
	}
	if _, ok := migrated.Load(path); ok {
		return nil
	}
	if err := migrate(path); err != nil {
		return err
	}
	if err := writeSchemaFile(name, schemaText()); err != nil {
		return err
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return err
	}
	migrated.Store(path, true)
	return nil
}

// migrated remembers, for this process, the files already at the current
// schema. Migrations only run forward, so the answer stays true while the
// file stays; loadMissing drops it when it brings a file back. It is package
// state on purpose (statute 2.11, 8.1): a cache of one fact about one file
// for one short process. It can go when sqlxdb holds an open handle per
// process, which would carry the same fact.
var migrated sync.Map

func writeSchemaFile(name, models string) error {
	root := os.Getenv("SQLX_ROOT")
	if root == "" {
		return nil
	}
	dir := filepath.Join(root, "orm")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, name), []byte(models), 0o600)
}

func fileMissing(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return false, nil
	}
	if os.IsNotExist(err) {
		return true, nil
	}
	return false, err
}

// loadMissing fills a missing runtime file through nebula keep, the one
// restore path. A restore that fails leaves no file, so an empty database is
// never made while the backup is out of reach. SQLX_ROOT is the local test
// tree and starts empty. A file it makes is migrated again even if this
// process migrated an older file at that path.
func loadMissing(path, name string) error {
	gone, err := fileMissing(path)
	if err != nil || !gone {
		return err
	}
	migrated.Delete(path)
	if os.Getenv("SQLX_ROOT") != "" {
		return os.WriteFile(path, nil, 0o600)
	}
	if err := keepRestore(path, name); err != nil {
		_ = os.Remove(path)
		return err
	}
	if gone, err = fileMissing(path); err != nil || gone {
		return fmt.Errorf("nebula keep restore %s: no file", name)
	}
	return nil
}

func adoptSQLite(dir, legacy, destPath string) error {
	oldPath := filepath.Join(dir, legacy+".sqlite")
	if _, err := os.Stat(oldPath); err != nil {
		return nil
	}
	if _, err := os.Stat(destPath); err == nil {
		return nil
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	return os.Rename(oldPath, destPath)
}

func sqliteQueryJSON(path, sql string, dest any) error {
	raw, err := sqliteRun(path, sql, true)
	if err != nil {
		return err
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	if err := json.Unmarshal([]byte(raw), dest); err != nil {
		return fmt.Errorf("sqlite json: %w", err)
	}
	return nil
}

func sqliteExec(path, sql string) error {
	_, err := sqliteRun(path, sql, false)
	return err
}

func sqliteRun(path, sql string, asJSON bool) (string, error) {
	bin := os.Getenv("SQLX_SQLITE")
	if bin == "" {
		bin = "/usr/bin/sqlite3"
	}
	args := []string{"-batch", "-cmd", ".timeout 5000", "-cmd", "PRAGMA foreign_keys = ON"}
	if asJSON {
		args = append(args, "-json")
	}
	args = append(args, path)
	cmd := exec.Command(bin, args...)
	cmd.Stdin = strings.NewReader(sql)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = strings.TrimSpace(stdout.String())
		}
		if msg == "" {
			msg = err.Error()
		}
		return stdout.String(), fmt.Errorf("%s", msg)
	}
	return stdout.String(), nil
}

func tableExists(path, name string) (bool, error) {
	var rows []struct {
		Name string `json:"name"`
	}
	err := sqliteQueryJSON(path, "SELECT name FROM sqlite_master WHERE type = 'table' AND name = "+sqlQuote(name), &rows)
	if err != nil {
		return false, err
	}
	return len(rows) > 0, nil
}

func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func sqlQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}
