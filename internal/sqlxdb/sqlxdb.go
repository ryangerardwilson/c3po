// Package sqlxdb stores every durable c3po record in the sqlx database c3po.
// Operator config, credential files, signatures, and caches stay on disk.
package sqlxdb

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
)

const DatabaseName = "c3po"

const sqliteTimeoutMS = "5000"

const metaModel = "model meta\n" +
	"  id integer primary key\n" +
	"  key text not null\n" +
	"  value text\n"

const recordModel = "model record\n" +
	"  id integer primary key\n" +
	"  app text not null\n" +
	"  kind text not null\n" +
	"  item_key text not null\n" +
	"  body text not null\n"

const createORM = `CREATE TABLE IF NOT EXISTS sqlx_orm (
  model TEXT PRIMARY KEY,
  body TEXT NOT NULL,
  position INTEGER NOT NULL
)`

const createMeta = `CREATE TABLE IF NOT EXISTS meta (
  "id" INTEGER PRIMARY KEY,
  "key" TEXT NOT NULL,
  "value" TEXT
)`

const createRecord = `CREATE TABLE IF NOT EXISTS record (
  "id" INTEGER PRIMARY KEY,
  "app" TEXT NOT NULL,
  "kind" TEXT NOT NULL,
  "item_key" TEXT NOT NULL,
  "body" TEXT NOT NULL
)`

// Read returns the sqlx record for a production path, or the file otherwise.
func Read(app, kind, key, path string) ([]byte, error) {
	if !useDB(path) {
		return os.ReadFile(path)
	}
	db, err := open()
	if err != nil {
		return nil, err
	}
	body, ok, err := get(db, app, kind, key)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, os.ErrNotExist
	}
	return body, nil
}

// Write stores a production record in sqlx. Other paths stay files.
func Write(app, kind, key, path string, body []byte, mode os.FileMode) error {
	if !useDB(path) {
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return err
		}
		return os.WriteFile(path, body, mode)
	}
	db, err := open()
	if err != nil {
		return err
	}
	return put(db, app, kind, key, body)
}

// Remove deletes a production record. A file path is removed only when it is not in sqlx.
func Remove(app, kind, key, path string) error {
	if !useDB(path) {
		err := os.Remove(path)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	db, err := open()
	if err != nil {
		return err
	}
	return execSQL(db, "DELETE FROM record WHERE app = "+quote(app)+" AND kind = "+quote(kind)+" AND item_key = "+quote(key)+";\n")
}

// Key is the record key for a legacy JSON file path.
func Key(path string) string {
	return strings.TrimSuffix(filepath.Base(path), ".json")
}

// Exists reports whether a production record or a file path is present.
func Exists(app, kind, key, path string) (bool, error) {
	if !useDB(path) {
		_, err := os.Stat(path)
		if err == nil {
			return true, nil
		}
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	db, err := open()
	if err != nil {
		return false, err
	}
	_, ok, err := get(db, app, kind, key)
	return ok, err
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
	if rel, err := filepath.Rel(data, path); err == nil && rel != "." && !strings.HasPrefix(rel, "..") {
		return true
	}
	routes := filepath.Join(home, ".config", "c3po", "tunnel", "routes.json")
	return path == filepath.Clean(routes)
}

func open() (string, error) {
	db, err := databaseFile()
	if err != nil {
		return "", err
	}
	if err := ensure(db); err != nil {
		return "", err
	}
	if err := importOnce(db); err != nil {
		return "", err
	}
	return db, nil
}

func databaseFile() (string, error) {
	root, err := sqlxRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "data", DatabaseName+".sqlite"), nil
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
	if err := prepareDirs(dbPath); err != nil {
		return err
	}
	sql := "PRAGMA journal_mode=WAL;\n" + createORM + ";\n" + createMeta + ";\n" + createRecord + ";\n" + ormUpsertSQL()
	if err := execSQL(dbPath, sql); err != nil {
		return err
	}
	return writeORMFile(dbPath)
}

func prepareDirs(dbPath string) error {
	dataDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return err
	}
	if err := os.Chmod(dataDir, 0o700); err != nil {
		return err
	}
	orm, ok := ormFile(dbPath)
	if !ok {
		return nil
	}
	root := filepath.Dir(dataDir)
	if err := os.Chmod(root, 0o700); err != nil && !os.IsPermission(err) {
		return err
	}
	ormDir := filepath.Dir(orm)
	if err := os.MkdirAll(ormDir, 0o700); err != nil {
		return err
	}
	return os.Chmod(ormDir, 0o700)
}

func ormFile(dbPath string) (string, bool) {
	if filepath.Base(dbPath) != DatabaseName+".sqlite" {
		return "", false
	}
	dataDir := filepath.Dir(dbPath)
	if filepath.Base(dataDir) != "data" {
		return "", false
	}
	return filepath.Join(filepath.Dir(dataDir), "orm", DatabaseName), true
}

func ormText() string {
	return metaModel + "\n" + recordModel
}

func ormUpsertSQL() string {
	models := []struct {
		name string
		body string
		pos  int
	}{
		{"meta", metaModel, 0},
		{"record", recordModel, 1},
	}
	var b strings.Builder
	for _, model := range models {
		fmt.Fprintf(&b, "INSERT INTO sqlx_orm(model, body, position) VALUES (%s, %s, %d) ON CONFLICT(model) DO UPDATE SET body = excluded.body, position = excluded.position;\n", quote(model.name), quote(model.body), model.pos)
	}
	return b.String()
}

func writeORMFile(dbPath string) error {
	orm, ok := ormFile(dbPath)
	if !ok {
		return nil
	}
	tmp, err := os.CreateTemp(filepath.Dir(orm), ".orm-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	done := false
	defer func() {
		if !done {
			_ = os.Remove(tmpName)
		}
	}()
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.WriteString(ormText()); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, orm); err != nil {
		return err
	}
	done = true
	return os.Chmod(orm, 0o600)
}

func importOnce(dbPath string) error {
	if !allowImport(dbPath) {
		return nil
	}
	value, ok, err := meta(dbPath, "imported_records")
	if err != nil {
		return err
	}
	if ok && value == "1" {
		return nil
	}
	home, err := loginHome()
	if err != nil {
		return err
	}
	data := filepath.Join(home, ".local", "share", "c3po")
	imports := []struct {
		dir  string
		app  string
		kind string
	}{
		{filepath.Join(data, "gmail", "tokens"), "gmail", "token"},
		{filepath.Join(data, "gcal", "tokens"), "gcal", "token"},
		{filepath.Join(data, "gdrive", "tokens"), "gdrive", "token"},
		{filepath.Join(data, "gdrive", "state"), "gdrive", "state"},
	}
	for _, item := range imports {
		if item.kind == "state" {
			if err := renameLegacyDriveState(item.dir); err != nil {
				return err
			}
		}
		if err := importDir(dbPath, item.dir, item.app, item.kind); err != nil {
			return err
		}
	}
	routes := filepath.Join(home, ".config", "c3po", "tunnel", "routes.json")
	if err := importFile(dbPath, routes, "tunnel", "routes", "routes"); err != nil {
		return err
	}
	return execSQL(dbPath, "INSERT INTO meta(key, value) SELECT "+quote("imported_records")+", "+quote("1")+" WHERE NOT EXISTS (SELECT 1 FROM meta WHERE key = "+quote("imported_records")+");\n")
}

// renameLegacyDriveState matches StateFile: preset 1 used <id>.json.
func renameLegacyDriveState(dir string) error {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".json") {
			continue
		}
		if strings.Contains(strings.TrimSuffix(name, ".json"), "-") {
			continue
		}
		dest := filepath.Join(dir, "1-"+name)
		if _, err := os.Stat(dest); err == nil {
			continue
		}
		if err := os.Rename(filepath.Join(dir, name), dest); err != nil {
			return err
		}
	}
	return nil
}

func allowImport(dbPath string) bool {
	if os.Getenv("SQLX_ROOT") != "" {
		return false
	}
	home, err := loginHome()
	if err != nil {
		return false
	}
	want := filepath.Join(home, ".sqlx", "data", DatabaseName+".sqlite")
	return filepath.Clean(dbPath) == filepath.Clean(want)
}

func importDir(dbPath, dir, app, kind string) error {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		key := strings.TrimSuffix(entry.Name(), ".json")
		if err := importFile(dbPath, filepath.Join(dir, entry.Name()), app, kind, key); err != nil {
			return err
		}
	}
	return nil
}

func importFile(dbPath, path, app, kind, key string) error {
	body, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	found, ok, err := get(dbPath, app, kind, key)
	if err != nil {
		return err
	}
	if ok && len(found) > 0 {
		return nil
	}
	return put(dbPath, app, kind, key, body)
}

func get(dbPath, app, kind, key string) ([]byte, bool, error) {
	var rows []struct {
		Body string `json:"body"`
	}
	sql := "SELECT body FROM record WHERE app = " + quote(app) + " AND kind = " + quote(kind) + " AND item_key = " + quote(key) + " ORDER BY id LIMIT 1"
	if err := queryJSON(dbPath, sql, &rows); err != nil {
		return nil, false, err
	}
	if len(rows) == 0 {
		return nil, false, nil
	}
	return []byte(rows[0].Body), true, nil
}

func put(dbPath, app, kind, key string, body []byte) error {
	sql := "BEGIN;\nDELETE FROM record WHERE app = " + quote(app) + " AND kind = " + quote(kind) + " AND item_key = " + quote(key) + ";\n" +
		"INSERT INTO record(app, kind, item_key, body) VALUES (" + quote(app) + ", " + quote(kind) + ", " + quote(key) + ", " + quote(string(body)) + ");\nCOMMIT;\n"
	return execSQL(dbPath, sql)
}

func meta(dbPath, key string) (string, bool, error) {
	var rows []struct {
		Value *string `json:"value"`
	}
	if err := queryJSON(dbPath, "SELECT value FROM meta WHERE key = "+quote(key)+" LIMIT 1", &rows); err != nil {
		return "", false, err
	}
	if len(rows) == 0 || rows[0].Value == nil {
		if len(rows) == 0 {
			return "", false, nil
		}
		return "", true, nil
	}
	return *rows[0].Value, true, nil
}

func quote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func queryJSON(dbPath, sql string, dest any) error {
	raw, err := run(dbPath, sql, "-json")
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

func execSQL(dbPath, sql string) error {
	cmd := exec.Command(sqliteBin(), append(sqliteArgs(), dbPath)...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		return err
	}
	_, writeErr := io.WriteString(stdin, sql)
	closeErr := stdin.Close()
	waitErr := cmd.Wait()
	if waitErr != nil {
		_ = chmodDB(dbPath)
		return sqliteErr(stderr.String(), stdout.String(), waitErr)
	}
	if writeErr != nil {
		_ = chmodDB(dbPath)
		return writeErr
	}
	if closeErr != nil {
		_ = chmodDB(dbPath)
		return closeErr
	}
	return chmodDB(dbPath)
}

func run(dbPath, sql string, flags ...string) (string, error) {
	args := append(sqliteArgs(flags...), dbPath, sql)
	cmd := exec.Command(sqliteBin(), args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		_ = chmodDB(dbPath)
		return stdout.String(), sqliteErr(stderr.String(), stdout.String(), err)
	}
	if err := chmodDB(dbPath); err != nil {
		return stdout.String(), err
	}
	return stdout.String(), nil
}

func sqliteArgs(flags ...string) []string {
	args := []string{"-batch", "-cmd", ".timeout " + sqliteTimeoutMS, "-cmd", ".bail on"}
	return append(args, flags...)
}

func sqliteBin() string {
	if bin := os.Getenv("SQLX_SQLITE"); bin != "" {
		return bin
	}
	return "/usr/bin/sqlite3"
}

func sqliteErr(stderr, stdout string, err error) error {
	msg := strings.TrimSpace(stderr)
	if msg == "" {
		msg = strings.TrimSpace(stdout)
	}
	if msg == "" {
		msg = err.Error()
	}
	return errors.New(msg)
}

func chmodDB(dbPath string) error {
	if _, err := os.Stat(dbPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	return os.Chmod(dbPath, 0o600)
}
