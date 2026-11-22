package sqlxdb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// The application owns this sqlite file. Nebula sql is the Postgres
// interface and is not called. Nebula s3 backs the file up.

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
	if err := migrate(path); err != nil {
		return err
	}
	if err := writeSchemaFile(name, schemaText()); err != nil {
		return err
	}
	return os.Chmod(path, 0o600)
}

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

func copyLegacyRuntime(path, name string) error {
	if os.Getenv("SQLX_ROOT") != "" {
		return nil
	}
	base := os.Getenv("XDG_RUNTIME_DIR")
	if base == "" {
		return nil
	}
	src := filepath.Join(base, "sqlx", name+".sqlite")
	if src == path {
		return nil
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return nil
	}
	return os.WriteFile(path, data, 0o600)
}

var errNotFound = errors.New("not found")

var runS3 = defaultRunS3

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

func loadMissing(path, name string) error {
	gone, err := fileMissing(path)
	if err != nil || !gone {
		return err
	}
	_ = copyLegacyRuntime(path, name)
	gone, err = fileMissing(path)
	if err != nil || !gone {
		return err
	}
	rerr := restoreObject(path, name)
	if rerr != nil && !errors.Is(rerr, errNotFound) {
		_ = os.Remove(path)
		return rerr
	}
	if errors.Is(rerr, errNotFound) {
		_ = os.Remove(path)
	}
	gone, err = fileMissing(path)
	if err != nil || !gone {
		return err
	}
	return os.WriteFile(path, nil, 0o600)
}

func restoreObject(path, name string) error {
	if os.Getenv("SQLX_ROOT") != "" {
		return nil
	}
	bin, args, err := s3Args("get", "s3://backups/"+name+".sqlite", path)
	if err != nil {
		return err
	}
	msg, err := runS3(bin, args)
	if err == nil {
		return nil
	}
	if strings.Contains(msg, "not found") || strings.Contains(err.Error(), "not found") {
		return errNotFound
	}
	return fmt.Errorf("s3 get s3://backups/%s.sqlite: %s", name, s3Text(msg, err))
}

func publishObject(path, name string) {
	if os.Getenv("SQLX_ROOT") != "" || path == "" {
		return
	}
	_ = publishFile(path, name)
}

func defaultRunS3(bin string, args []string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func s3Text(msg string, err error) string {
	text := strings.TrimSpace(msg)
	if text == "" && err != nil {
		return strings.TrimSpace(err.Error())
	}
	return text
}

func s3Args(action string, rest ...string) (string, []string, error) {
	if bin := os.Getenv("NEBULA_BIN"); bin != "" {
		return bin, append([]string{"s3", action}, rest...), nil
	}
	if bin, err := exec.LookPath("s3"); err == nil {
		return bin, append([]string{action}, rest...), nil
	}
	bin, err := exec.LookPath("nebula")
	if err != nil {
		return "", nil, err
	}
	return bin, append([]string{"s3", action}, rest...), nil
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
