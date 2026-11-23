package sqlxdb

import (
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestMigrateRunsEachVersionOnce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "c3po.sqlite")
	legacy := "CREATE TABLE record (id INTEGER PRIMARY KEY, app TEXT NOT NULL, kind TEXT NOT NULL, item_key TEXT NOT NULL, body TEXT NOT NULL);\n" +
		"INSERT INTO record(app, kind, item_key, body) VALUES ('a', 'k', '1', 'old'), ('a', 'k', '1', 'new');\n" +
		"INSERT INTO record(app, kind, item_key, body) VALUES " +
		"('tui', 'session', '01a17622-5fd9-7bc2-81ff-3e200a8c16b7', '{\"id\":\"x\",\"screen\":\"o\",\"cwd\":\"/w\",\"args\":[\"-a\"]}'), " +
		"('tui', 'session', 'bad', '{\"screen\":\"o\"}'), " +
		"('tui', 'active', '01a17622-5fd9-7bc2-81ff-3e200a8c16b7', '{\"session_id\":\"x\",\"pid\":7,\"pid_start\":9,\"screen\":\"o\"}'), " +
		"('tui', 'active', '01a17622-5fd9-7bc2-81ff-3e200a8c16b8', '{\"pid\":8}');\n" +
		"INSERT INTO record(app, kind, item_key, body) VALUES " +
		"('idealake', 'idea', '7', '{\"id\":\"7\",\"topic\":\"x\",\"text\":\"seven\",\"created\":\"2026-01-01T00:00:00Z\"}'), " +
		"('idealake', 'idea', '9', '{\"topic\":\"y\",\"text\":\"nine\",\"created\":\"2026-01-02T00:00:00Z\",\"model\":\"m\",\"embedding\":[0.5,1]}'), " +
		"('idealake', 'topic', 'x', '{\"topic\":\"x\",\"created\":\"2025-12-31T00:00:00Z\"}'), " +
		"('idealake', 'topic', 'empty', '{\"created\":\"2025-12-31T00:00:00Z\"}'), " +
		"('idealake', 'meta', 'next', '12');\n" +
		"INSERT INTO record(app, kind, item_key, body) VALUES ('gdrive', 'state', '1-1', '{\"entries\":[{\"relpath\":\"a/b\",\"kind\":\"file\",\"drive_id\":\"f\",\"parent_relpath\":\"a\",\"size\":3,\"mtime_ns\":5,\"sha1\":\"s\"},{\"relpath\":\"\",\"kind\":\"dir\"}]}');\n" +
		"INSERT INTO record(app, kind, item_key, body) VALUES ('looptab', 'files', 'files', '{\"files\":[{\"name\":\"b\",\"path\":\"/j/b\"},{\"name\":\"a\",\"path\":\"/j/a\"}]}');\n" +
		"INSERT INTO record(app, kind, item_key, body) VALUES ('gmail', 'token', 'a@b.c', '{\"refresh_token\":\"r\"}'), ('gcal', 'token', 'x@y.z', 'broken');\n" +
		"CREATE TABLE app_once (id TEXT PRIMARY KEY);\nINSERT INTO app_once VALUES ('gdrive-state-parent-1');\n"
	if err := sqliteExec(path, legacy); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := migrate(path); err != nil {
			t.Fatal(err)
		}
	}
	n, err := countRows(path, "schema_migration")
	if err != nil || n != len(migrations()) {
		t.Fatalf("versions %d %v", n, err)
	}
	if exists, err := tableExists(path, "app_once"); err != nil || exists {
		t.Fatalf("app_once %v %v", exists, err)
	}
	for _, check := range []struct{ sql, want string }{
		{"SELECT id, screen, cwd, args FROM tui_session", `[{"id":"01a17622-5fd9-7bc2-81ff-3e200a8c16b7","screen":"o","cwd":"/w","args":"[\"-a\"]"}]`},
		{"SELECT session_id, pid, pid_start FROM tui_active", `[{"session_id":"01a17622-5fd9-7bc2-81ff-3e200a8c16b7","pid":7,"pid_start":9}]`},
		{"SELECT name, created_at FROM idealake_topic ORDER BY name", `[{"name":"empty","created_at":1767139200},{"name":"x","created_at":1767139200},{"name":"y","created_at":1767312000}]`},
		{"SELECT i.id, t.name, i.text, i.model, i.embedding FROM idealake_idea i JOIN idealake_topic t ON t.id = i.topic_id ORDER BY i.id",
			`[{"id":7,"name":"x","text":"seven","model":null,"embedding":null},{"id":9,"name":"y","text":"nine","model":"m","embedding":"[0.5,1]"}]`},
		{"SELECT seq FROM sqlite_sequence WHERE name = 'idealake_idea'", `[{"seq":12}]`},
		{"SELECT body FROM record WHERE app = 'a'", `[{"body":"new"}]`},
		{"SELECT preset, registration, relpath, kind, drive_id, size, mtime_ns, sha1 FROM gdrive_sync_entry",
			`[{"preset":"1","registration":"1","relpath":"a/b","kind":"file","drive_id":"f","size":3,"mtime_ns":5,"sha1":"s"}]`},
		{"SELECT name, path FROM looptab_file ORDER BY id", `[{"name":"b","path":"/j/b"},{"name":"a","path":"/j/a"}]`},
		{"SELECT app, email, document FROM google_token", `[{"app":"gmail","email":"a@b.c","document":"{\"refresh_token\":\"r\"}"}]`},
		{"SELECT count(*) AS n FROM record WHERE app IN ('tui', 'idealake', 'looptab', 'gdrive', 'gmail', 'gcal')", `[{"n":0}]`},
	} {
		raw, err := sqliteRun(path, check.sql, true)
		if err != nil || compact(raw) != compact(check.want) {
			t.Fatalf("%s\n got %s\nwant %s %v", check.sql, compact(raw), check.want, err)
		}
	}
}

func compact(raw string) string {
	var v any
	if json.Unmarshal([]byte(raw), &v) != nil {
		return raw
	}
	out, _ := json.Marshal(v)
	return string(out)
}

func TestFailedMigrationCommitsNothing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "c3po.sqlite")
	if err := migrate(path); err != nil {
		t.Fatal(err)
	}
	bad := migration{version: 999, name: "bad", sql: "INSERT INTO record(app, kind, item_key, body) VALUES ('a', 'k', '1', 'x');\nINSERT INTO missing VALUES (1)"}
	if err := sqliteExec(path, migrationScript(bad)); err == nil {
		t.Fatal("a failing migration returned no error")
	}
	if n, _ := countRows(path, "record"); n != 0 {
		t.Fatal("the failed migration wrote a record")
	}
	n, err := countRows(path, "schema_migration")
	if err != nil || n != len(migrations()) {
		t.Fatalf("versions %d %v", n, err)
	}
}

func TestSchemaRejectsBrokenRows(t *testing.T) {
	path := filepath.Join(t.TempDir(), "c3po.sqlite")
	legacy := "CREATE TABLE usage (id INTEGER PRIMARY KEY, feature TEXT NOT NULL, actor TEXT NOT NULL, day TEXT NOT NULL, n INTEGER NOT NULL, ok INTEGER NOT NULL, last INTEGER NOT NULL);\n" +
		"INSERT INTO usage(feature, actor, day, n, ok, last) VALUES ('ls', 'human', '2026-11-22', 2, 1, 5), ('ls', 'robot', '2026-11-22', 1, 1, 5);\n"
	if err := sqliteExec(path, legacy); err != nil {
		t.Fatal(err)
	}
	if err := migrate(path); err != nil {
		t.Fatal(err)
	}
	if n, err := countRows(path, "usage"); err != nil || n != 1 {
		t.Fatalf("usage rows %d %v", n, err)
	}
	for _, bad := range []string{
		"INSERT INTO usage(feature, actor, day, n, ok, last) VALUES ('ls', 'human', '2026-11-23', 1, 2, 5)",
		"INSERT INTO usage(feature, actor, day, n, ok, last) VALUES ('ls', 'human', '22/11/2026', 1, 1, 5)",
		"INSERT INTO idealake_topic(name, created_at) VALUES ('two' || char(10) || 'lines', 0)",
		"INSERT INTO looptab_file(name, path) VALUES ('Jobs', '/j')",
		"INSERT INTO tui_active(session_id, pid) VALUES ('01a17622-5fd9-7bc2-81ff-3e200a8c16b7', 1)",
		"INSERT INTO google_token(app, email, document) VALUES ('slack', 'a@b.c', '{}')",
		"INSERT INTO gdrive_sync_entry(preset, registration, relpath, kind, drive_id) VALUES ('1', '1', 'a', 'link', 'x')",
	} {
		if err := sqliteExec(path, bad); err == nil {
			t.Fatalf("accepted %s", bad)
		}
	}
}

func TestMigrationsAreSafeToApplyTwice(t *testing.T) {
	path := filepath.Join(t.TempDir(), "c3po.sqlite")
	if err := sqliteExec(path, "CREATE TABLE usage (id INTEGER PRIMARY KEY, feature TEXT NOT NULL, actor TEXT NOT NULL, day TEXT NOT NULL, n INTEGER NOT NULL, ok INTEGER NOT NULL, last INTEGER NOT NULL);\n"+
		"INSERT INTO usage(feature, actor, day, n, ok, last) VALUES ('ls', 'human', '2026-11-22', 2, 1, 5);\n"); err != nil {
		t.Fatal(err)
	}
	if err := migrate(path); err != nil {
		t.Fatal(err)
	}
	for _, m := range migrations() {
		if err := sqliteExec(path, ".bail on\nBEGIN IMMEDIATE;\n"+m.sql+";\nCOMMIT;\n"); err != nil {
			t.Fatalf("version %d again: %v", m.version, err)
		}
		if err := sqliteExec(path, migrationScript(m)); err == nil {
			t.Fatalf("version %d ran a second time through the ledger", m.version)
		}
	}
	if err := migrate(path); err != nil {
		t.Fatal(err)
	}
	if n, err := countRows(path, "usage"); err != nil || n != 1 {
		t.Fatalf("usage rows %d %v", n, err)
	}
}

func countRows(path, table string) (int, error) {
	var rows []struct {
		N int `json:"n"`
	}
	if err := sqliteQueryJSON(path, "SELECT COUNT(*) AS n FROM "+quoteIdent(table), &rows); err != nil {
		return 0, err
	}
	if len(rows) == 0 {
		return 0, nil
	}
	return rows[0].N, nil
}
