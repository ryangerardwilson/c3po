package sqlxdb

import (
	"os"
	"path/filepath"
	"strconv"
)

// RuntimeMode reports whether this process keeps the in-memory sqlite
// and backs it up through nebula keep. SQLX_ROOT and SQLX_ENGINE=sqlite stay on the
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

// UsageBody is one increment inside a transaction. day is YYYY-MM-DD in UTC
// and is stored as whole days since 1970-01-01. n and ok are the amounts to
// add. last is the unix time of this run.
func UsageBody(feature, actor, day string, n, ok int, last int64) string {
	if n <= 0 {
		n = 1
	}
	if ok < 0 {
		ok = 0
	}
	lastText := strconv.FormatInt(last, 10)
	dayNumber := "(unixepoch(" + quote(day) + ") / 86400)"
	return "UPDATE usage SET n = n + " + strconv.Itoa(n) + ", ok = ok + " + strconv.Itoa(ok) +
		", last = CASE WHEN last > " + lastText + " THEN last ELSE " + lastText +
		" END WHERE feature = " + quote(feature) + " AND actor = " + quote(actor) +
		" AND day = " + dayNumber + ";\n" +
		"INSERT INTO usage(feature, actor, day, n, ok, last) SELECT " +
		quote(feature) + ", " + quote(actor) + ", " + dayNumber + ", " + strconv.Itoa(n) + ", " + strconv.Itoa(ok) + ", " + lastText +
		" WHERE changes() = 0;\n"
}

// WrapUsage makes one immediate transaction.
func WrapUsage(body string) string {
	return "BEGIN IMMEDIATE;\n" + body + "COMMIT;\n"
}

// UsageBatchApplied reports whether the runtime sqlite already holds the
// usage batch id.
func UsageBatchApplied(id string) (bool, error) {
	if err := migrate(RuntimeDB()); err != nil {
		return false, err
	}
	var rows []struct {
		N int `json:"n"`
	}
	if err := sqliteQueryJSON(RuntimeDB(), "SELECT COUNT(*) AS n FROM usage_batch WHERE id = "+quote(id), &rows); err != nil {
		return false, err
	}
	return len(rows) == 1 && rows[0].N > 0, nil
}

// WrapUsageBatch is one immediate transaction that applies body and records
// batch id with it, so a batch is applied once. It stops at the first
// failing statement, so nothing commits half. now is a unix time.
func WrapUsageBatch(id, body string, now int64) string {
	return ".bail on\nBEGIN IMMEDIATE;\n" + body +
		"INSERT INTO usage_batch(id, applied) VALUES (" + quote(id) + ", " + strconv.FormatInt(now, 10) + ");\n" +
		"DELETE FROM usage_batch WHERE applied < " + strconv.FormatInt(now-30*86400, 10) + ";\n" +
		"COMMIT;\n"
}

// ExecRuntime brings the app sqlite to the current schema, applies sql, and
// does not publish the backup.
func ExecRuntime(sql string) error {
	if err := migrate(RuntimeDB()); err != nil {
		return err
	}
	return sqliteExec(RuntimeDB(), sql)
}

// ExecBackup marks the runtime sqlite changed for nebula keep. The worker
// already wrote it.
func ExecBackup() error {
	return keepMark(RuntimeDB(), DatabaseName)
}

// TouchRuntime opens the app sqlite and restores it through nebula keep when
// it is missing.
func TouchRuntime() error {
	return prepareAppDB(RuntimeDB(), DatabaseName)
}
