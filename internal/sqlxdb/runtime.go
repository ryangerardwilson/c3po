package sqlxdb

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

// RuntimeMode reports whether this process is on the production path:
// the runtime sqlite is the source of truth and Postgres is only the backup.
// SQLX_ROOT and SQLX_ENGINE=sqlite stay on the local tree used by tests.
func RuntimeMode() bool {
	switch os.Getenv("SQLX_ENGINE") {
	case "sqlite":
		return false
	case "postgres":
		return true
	default:
		return os.Getenv("SQLX_ROOT") == ""
	}
}

// RuntimeDB is the sqlite file sqlx serves while the machine is up.
func RuntimeDB() string {
	name := DatabaseName + ".sqlite"
	if v := os.Getenv("SQLX_RUNTIME"); v != "" {
		return filepath.Join(v, name)
	}
	if v := os.Getenv("XDG_RUNTIME_DIR"); v != "" {
		return filepath.Join(v, "sqlx", name)
	}
	return filepath.Join("/dev/shm", fmt.Sprintf("sqlx-%d", os.Getuid()), name)
}

// RuntimeReady reports whether that file is present. A missing file must not
// make the command dial the backup.
func RuntimeReady() bool {
	info, err := os.Stat(RuntimeDB())
	return err == nil && info.Mode().IsRegular()
}

// UsageBody is one increment inside a transaction. n and ok are the amounts
// to add. last is the unix time of this run.
func UsageBody(feature, actor, day string, n, ok int, last int64) string {
	if n <= 0 {
		n = 1
	}
	if ok < 0 {
		ok = 0
	}
	lastText := strconv.FormatInt(last, 10)
	return "UPDATE usage SET n = n + " + strconv.Itoa(n) + ", ok = ok + " + strconv.Itoa(ok) +
		", last = CASE WHEN last > " + lastText + " THEN last ELSE " + lastText +
		" END WHERE feature = " + quote(feature) + " AND actor = " + quote(actor) +
		" AND day = " + quote(day) + ";\n" +
		"INSERT INTO usage(feature, actor, day, n, ok, last) SELECT " +
		quote(feature) + ", " + quote(actor) + ", " + quote(day) + ", " + strconv.Itoa(n) + ", " + strconv.Itoa(ok) + ", " + lastText +
		" WHERE changes() = 0;\n"
}

// WrapUsage makes one immediate transaction.
func WrapUsage(body string) string {
	return "BEGIN IMMEDIATE;\n" + body + "COMMIT;\n"
}

// ExecRuntime applies sql to the runtime sqlite and does not open the backup.
func ExecRuntime(sql string) error {
	return execSQLEnv([]string{"SQLX_COMMIT=local"}, sql)
}

// ExecBackup sends sql to the backup and does not write the runtime copy again.
func ExecBackup(sql string) error {
	return execSQLEnv([]string{"SQLX_COMMIT=remote"}, sql)
}

// TouchRuntime opens the runtime database so a missing copy can be restored.
func TouchRuntime() error {
	_, err := sqlxCall(DatabaseName, "query", "SELECT 1")
	return err
}
