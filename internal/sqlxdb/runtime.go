package sqlxdb

import (
	"os"
	"path/filepath"
	"strconv"
)

// RuntimeMode reports whether this process keeps the in-memory sqlite
// and publishes it with s3. SQLX_ROOT and SQLX_ENGINE=sqlite stay on the
// local tree used by tests.
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

// RuntimeDB is the sqlite file this app serves while the machine is up.
func RuntimeDB() string {
	if v := os.Getenv("SQLX_RUNTIME"); v != "" {
		return filepath.Join(v, DatabaseName+".sqlite")
	}
	return dbFile(DatabaseName)
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

// ExecRuntime applies sql to the app sqlite and does not publish the backup.
func ExecRuntime(sql string) error {
	return sqliteExec(RuntimeDB(), sql)
}

// ExecBackup publishes the runtime sqlite. The worker already wrote it.
// An empty record table is not published.
func ExecBackup() error {
	return publishFile(RuntimeDB(), DatabaseName)
}

// TouchRuntime opens the app sqlite and restores it from s3 when it is missing.
func TouchRuntime() error {
	return prepareAppDB(RuntimeDB(), DatabaseName, ormText())
}
