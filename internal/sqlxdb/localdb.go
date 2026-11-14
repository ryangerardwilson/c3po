package sqlxdb

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// The application owns this sqlite file. Nebula sql is the Postgres
// interface and is not called. Nebula s3 backs the file up.

type appModel struct {
	name    string
	cols    []appColumn
	indexes []string
}

type appColumn struct {
	name    string
	typ     string
	pk      bool
	notNull bool
	def     string
}

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

func prepareAppDB(path, name, models string) error {
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
	if _, err := os.Stat(path); os.IsNotExist(err) {
		_ = copyLegacyRuntime(path, name)
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		_ = restoreObject(path, name)
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.WriteFile(path, nil, 0o600); err != nil {
			return err
		}
	}
	if err := applyModels(path, models); err != nil {
		return err
	}
	if err := writeSchemaFile(name, models); err != nil {
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

func restoreObject(path, name string) error {
	if os.Getenv("SQLX_ROOT") != "" {
		return nil
	}
	bin, args, err := s3Args("get", "s3://backups/"+name+".sqlite", path)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return exec.CommandContext(ctx, bin, args...).Run()
}

func publishObject(path, name string) {
	if os.Getenv("SQLX_ROOT") != "" || path == "" {
		return
	}
	bin, args, err := s3Args("put", path, "s3://backups/"+name+".sqlite")
	if err != nil {
		return
	}
	go runS3(bin, args)
}

func runS3(bin string, args []string) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = exec.CommandContext(ctx, bin, args...).Run()
}

func s3Args(action string, rest ...string) (string, []string, error) {
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
	args := []string{"-batch", "-cmd", ".timeout 5000"}
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

func applyModels(path, text string) error {
	for _, model := range parseModels(text) {
		if err := applyModel(path, model); err != nil {
			return err
		}
	}
	return nil
}

func applyModel(path string, model appModel) error {
	exists, err := tableExists(path, model.name)
	if err != nil {
		return err
	}
	if !exists {
		if err := sqliteExec(path, createSQL(model)); err != nil {
			return err
		}
	} else if err := addMissing(path, model); err != nil {
		return err
	}
	for _, index := range model.indexes {
		if index == "" {
			continue
		}
		if err := sqliteExec(path, index); err != nil {
			return err
		}
	}
	return nil
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

func addMissing(path string, model appModel) error {
	have, err := tableColumns(path, model.name)
	if err != nil {
		return err
	}
	for _, column := range model.cols {
		if have[column.name] || column.pk {
			continue
		}
		if err := sqliteExec(path, "ALTER TABLE "+quoteIdent(model.name)+" ADD COLUMN "+columnSQL(column)); err != nil {
			return err
		}
	}
	return nil
}

func tableColumns(path, table string) (map[string]bool, error) {
	var rows []struct {
		Name string `json:"name"`
	}
	if err := sqliteQueryJSON(path, "PRAGMA table_info("+quoteIdent(table)+")", &rows); err != nil {
		return nil, err
	}
	out := map[string]bool{}
	for _, row := range rows {
		out[row.Name] = true
	}
	return out, nil
}

func parseModels(text string) []appModel {
	var models []appModel
	cur := -1
	for _, line := range strings.Split(text, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		switch fields[0] {
		case "model":
			if len(fields) < 2 {
				continue
			}
			models = append(models, appModel{name: fields[1]})
			cur = len(models) - 1
		case "index":
			if cur < 0 {
				continue
			}
			models[cur].indexes = append(models[cur].indexes, indexSQL(models[cur].name, fields))
		default:
			if cur < 0 || len(fields) < 2 {
				continue
			}
			models[cur].cols = append(models[cur].cols, parseColumn(fields))
		}
	}
	return models
}

func parseColumn(fields []string) appColumn {
	column := appColumn{name: fields[0], typ: fields[1]}
	for i := 2; i < len(fields); {
		switch fields[i] {
		case "primary":
			if i+1 < len(fields) && fields[i+1] == "key" {
				column.pk = true
				i += 2
				continue
			}
		case "not":
			if i+1 < len(fields) && fields[i+1] == "null" {
				column.notNull = true
				i += 2
				continue
			}
		case "default":
			column.def = strings.Join(fields[i+1:], " ")
			return column
		}
		i++
	}
	return column
}

func indexSQL(table string, fields []string) string {
	if len(fields) < 3 {
		return ""
	}
	name := fields[1]
	rest := fields[2:]
	unique := ""
	if rest[0] == "unique" {
		unique = "UNIQUE "
		rest = rest[1:]
	}
	if len(rest) == 0 {
		return ""
	}
	cols := make([]string, len(rest))
	for i, column := range rest {
		cols[i] = quoteIdent(column)
	}
	return "CREATE " + unique + "INDEX IF NOT EXISTS " + quoteIdent(name) + " ON " + quoteIdent(table) + "(" + strings.Join(cols, ", ") + ")"
}

func createSQL(model appModel) string {
	var b strings.Builder
	b.WriteString("CREATE TABLE IF NOT EXISTS " + quoteIdent(model.name) + " (\n")
	for i, column := range model.cols {
		b.WriteString("  " + columnSQL(column))
		if i+1 < len(model.cols) {
			b.WriteString(",")
		}
		b.WriteString("\n")
	}
	b.WriteString(")")
	return b.String()
}

func columnSQL(column appColumn) string {
	line := quoteIdent(column.name) + " " + strings.ToUpper(column.typ)
	if column.pk {
		line += " PRIMARY KEY"
	}
	if column.notNull {
		line += " NOT NULL"
	}
	if column.def != "" {
		line += " DEFAULT " + column.def
	}
	return line
}

func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func sqlQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func runOnce(path, id, sql string) error {
	if err := sqliteExec(path, "CREATE TABLE IF NOT EXISTS app_once (id TEXT PRIMARY KEY)"); err != nil {
		return err
	}
	var rows []struct {
		N int `json:"n"`
	}
	if err := sqliteQueryJSON(path, "SELECT COUNT(*) AS n FROM app_once WHERE id = "+sqlQuote(id), &rows); err != nil {
		return err
	}
	if len(rows) > 0 && rows[0].N > 0 {
		return nil
	}
	if err := sqliteExec(path, sql); err != nil {
		return err
	}
	return sqliteExec(path, "INSERT INTO app_once(id) VALUES ("+sqlQuote(id)+")")
}
