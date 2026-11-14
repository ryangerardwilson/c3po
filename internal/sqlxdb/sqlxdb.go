// Package sqlxdb stores every durable c3po record in the sqlx database c3po.
// Operator config, credential files, signatures, and caches stay on disk.
package sqlxdb

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
)

const DatabaseName = "c3po"

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

// usageModel is the daily feature counter. One row is one feature, one actor,
// and one UTC day. n is how many times it ran. ok is how many of those exited 0.
const usageModel = "model usage\n" +
	"  id integer primary key\n" +
	"  feature text not null\n" +
	"  actor text not null\n" +
	"  day text not null\n" +
	"  n integer not null\n" +
	"  ok integer not null\n" +
	"  last integer not null\n" +
	"  index usage_feature_actor_day unique feature actor day\n"

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
	_ = dbPath
	if _, err := sqlxCall("create", "database", DatabaseName); err != nil {
		return err
	}
	return sqlxMigrate(ormText())
}

func ormText() string {
	return metaModel + "\n" + recordModel + "\n" + usageModel
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

// Record is one sqlx row addressed by app, kind, and key.
type Record struct {
	Key  string
	Body []byte
}

// Put writes a record. It always uses the sqlx database, including for
// callers that do not have a legacy file path.
func Put(app, kind, key string, body []byte) error {
	db, err := open()
	if err != nil {
		return err
	}
	return put(db, app, kind, key, body)
}

// Get reads one record. ok is false when the key is absent.
func Get(app, kind, key string) ([]byte, bool, error) {
	db, err := open()
	if err != nil {
		return nil, false, err
	}
	return get(db, app, kind, key)
}

// Delete removes one record. A missing key is not an error.
func Delete(app, kind, key string) error {
	db, err := open()
	if err != nil {
		return err
	}
	return execSQL(db, "DELETE FROM record WHERE app = "+quote(app)+" AND kind = "+quote(kind)+" AND item_key = "+quote(key))
}

// List returns every record of one app and kind.
func List(app, kind string) ([]Record, error) {
	db, err := open()
	if err != nil {
		return nil, err
	}
	var rows []struct {
		ItemKey string `json:"item_key"`
		Body    string `json:"body"`
	}
	sql := "SELECT item_key, body FROM record WHERE app = " + quote(app) + " AND kind = " + quote(kind) + " ORDER BY item_key"
	if err := queryJSON(db, sql, &rows); err != nil {
		return nil, err
	}
	out := make([]Record, 0, len(rows))
	for _, row := range rows {
		out = append(out, Record{Key: row.ItemKey, Body: []byte(row.Body)})
	}
	return out, nil
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
	_ = dbPath
	raw, err := sqlxCall(DatabaseName, "query", "--json", sql)
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
	_ = dbPath
	return execSQLEnv(nil, sql)
}

func execSQLEnv(env []string, sql string) error {
	file, err := os.CreateTemp("", DatabaseName+"-sql-*")
	if err != nil {
		return err
	}
	sqlPath := file.Name()
	defer os.Remove(sqlPath)
	if _, err := file.WriteString(sql); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	_, err = sqlxCallEnv(env, DatabaseName, "exec", "--file", sqlPath)
	return err
}

func sqlxBin() string {
	if bin := os.Getenv("SQLX_BIN"); bin != "" {
		return bin
	}
	path, err := exec.LookPath("sqlx")
	if err != nil {
		return "sqlx"
	}
	return path
}

func sqlxCall(args ...string) (string, error) {
	return sqlxCallEnv(nil, args...)
}

func sqlxCallEnv(extra []string, args ...string) (string, error) {
	cmd := exec.Command(sqlxBin(), args...)
	if len(extra) > 0 {
		cmd.Env = append(os.Environ(), extra...)
	}
	var stdout, stderr bytes.Buffer
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

func sqlxMigrate(body string) error {
	file, err := os.CreateTemp("", DatabaseName+"-orm-*")
	if err != nil {
		return err
	}
	name := file.Name()
	defer os.Remove(name)
	if _, err := file.WriteString(body); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	_, err = sqlxCall(DatabaseName, "migrate", name)
	return err
}
