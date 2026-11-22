// Package sqlxdb owns the c3po sqlite file: its versioned schema and the
// backup. Each app reads and writes its own
// tables through Query and Exec. nebula s3 backs the file up. nebula sql is
// not used. The record and meta tables hold nebula's tunnel routes and its
// import flag; nebula creates, reads, and writes them, and c3po does not.
// Operator config, credential files, signatures, and caches stay on disk.
package sqlxdb

import (
	"errors"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

const DatabaseName = "c3po"

// Key is the record key for a legacy JSON file path.
func Key(path string) string {
	return strings.TrimSuffix(filepath.Base(path), ".json")
}

func useDB(path string) bool {
	if os.Getenv("C3PO_SQLX") == "off" {
		return false
	}
	path = filepath.Clean(path)
	if os.Getenv("SQLX_ROOT") != "" && strings.Contains(path, string(os.PathSeparator)+"c3po"+string(os.PathSeparator)) {
		return true
	}
	home, err := loginHome()
	if err != nil {
		return false
	}
	data := filepath.Join(home, ".local", "share", "c3po")
	rel, err := filepath.Rel(data, path)
	return err == nil && rel != "." && !strings.HasPrefix(rel, "..")
}

func open() (string, error) {
	db, err := databaseFile()
	if err != nil {
		return "", err
	}
	if err := ensure(db); err != nil {
		return "", err
	}
	return db, nil
}

func databaseFile() (string, error) {
	return dbFile(DatabaseName), nil
}

func sqlxRoot() (string, error) {
	if root := os.Getenv("SQLX_ROOT"); root != "" {
		return root, nil
	}
	home, err := loginHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".sqlx"), nil
}

func loginHome() (string, error) {
	current, err := user.Current()
	if err != nil {
		return "", err
	}
	if current.HomeDir == "" {
		return "", errors.New("login home is empty")
	}
	return current.HomeDir, nil
}

func ensure(dbPath string) error {
	return prepareAppDB(dbPath, DatabaseName)
}

func quote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func queryJSON(dbPath, sql string, dest any) error {
	return sqliteQueryJSON(dbPath, sql, dest)
}

// publishAfterWrite uploads after a durable write. Tests replace it.
var publishAfterWrite = publishObject

func execSQL(dbPath, sql string) error {
	return sqliteExec(dbPath, sql)
}
