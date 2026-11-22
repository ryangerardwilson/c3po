package sqlxdb

import (
	"strconv"
	"strings"
	"time"
)

// migration is one versioned change to the c3po database. Migrations run in
// version order. Each runs in one immediate transaction together with its
// schema_migration row, so a migration and its version commit together or not
// at all. Each sql is also safe to apply twice, so an older ledger that lost a
// row cannot damage data.
type migration struct {
	version int
	name    string
	sql     string
}

const migrationTable = "CREATE TABLE IF NOT EXISTS schema_migration (\n" +
	"  version INTEGER PRIMARY KEY,\n" +
	"  name TEXT NOT NULL,\n" +
	"  applied_at INTEGER NOT NULL\n" +
	") STRICT"

// migrations is the c3po schema history. Append only; never edit an applied entry.
func migrations() []migration {
	return []migration{
		{1, "baseline", "" +
			"CREATE TABLE IF NOT EXISTS meta (id INTEGER PRIMARY KEY, key TEXT NOT NULL, value TEXT);\n" +
			"CREATE TABLE IF NOT EXISTS record (id INTEGER PRIMARY KEY, app TEXT NOT NULL, kind TEXT NOT NULL, item_key TEXT NOT NULL, body TEXT NOT NULL);\n" +
			"CREATE TABLE IF NOT EXISTS usage (id INTEGER PRIMARY KEY, feature TEXT NOT NULL, actor TEXT NOT NULL, day TEXT NOT NULL, n INTEGER NOT NULL, ok INTEGER NOT NULL, last INTEGER NOT NULL);\n" +
			"CREATE UNIQUE INDEX IF NOT EXISTS usage_feature_actor_day ON usage(feature, actor, day)"},
		// A record keeps its id across writes. Duplicate keys keep the newest row.
		{2, "record-stable-key", "" +
			"DELETE FROM record WHERE id NOT IN (SELECT MAX(id) FROM record GROUP BY app, kind, item_key);\n" +
			"CREATE UNIQUE INDEX IF NOT EXISTS record_app_kind_key ON record(app, kind, item_key)"},
		// The record key is the one place for an idea id and a topic name.
		{3, "idealake-body-key", "" +
			"UPDATE record SET body = json_remove(body, '$.id') WHERE app = 'idealake' AND kind = 'idea' AND json_valid(body);\n" +
			"UPDATE record SET body = json_remove(body, '$.topic') WHERE app = 'idealake' AND kind = 'topic' AND json_valid(body)"},
		// A Drive sync entry's parent is derived from its relpath.
		{4, "gdrive-state-parent", "" +
			"UPDATE record SET body = json_set(body, '$.entries', " +
			"(SELECT json_group_array(json_remove(value, '$.parent_relpath')) FROM json_each(record.body, '$.entries'))) " +
			"WHERE app = 'gdrive' AND kind = 'state' AND json_valid(body) AND json_type(body, '$.entries') = 'array'"},
		// The record key is the session id; the session body holds screen and workspace.
		{5, "tui-active-copies", "" +
			"UPDATE record SET body = json_remove(body, '$.session_id', '$.screen', '$.cwd') WHERE app = 'tui' AND kind = 'active' AND json_valid(body);\n" +
			"UPDATE record SET body = json_remove(body, '$.id') WHERE app = 'tui' AND kind = 'session' AND json_valid(body)"},
		// schema_migration replaces the app_once ledger. Every app_once entry is
		// one of the versions above.
		{6, "drop-app-once", "DROP TABLE IF EXISTS app_once"},
		// A screen session and its running process become tables. The active
		// row belongs to its session and goes with it. A session record with
		// a bad id or no screen cannot be opened and is not copied. An active
		// record whose session is missing is not copied. args is NULL when the
		// screen noted no args; [] is an explicit empty list.
		{7, "tui-tables", "" +
			"CREATE TABLE IF NOT EXISTS tui_session (\n" +
			"  id TEXT PRIMARY KEY CHECK (length(id) = 36),\n" +
			"  screen TEXT NOT NULL CHECK (screen <> ''),\n" +
			"  cwd TEXT NOT NULL DEFAULT '',\n" +
			"  path TEXT NOT NULL DEFAULT '',\n" +
			"  cursor TEXT NOT NULL DEFAULT '',\n" +
			"  args TEXT CHECK (args IS NULL OR json_type(args) = 'array')\n" +
			") STRICT;\n" +
			"CREATE TABLE IF NOT EXISTS tui_active (\n" +
			"  session_id TEXT PRIMARY KEY REFERENCES tui_session(id) ON DELETE CASCADE,\n" +
			"  pid INTEGER NOT NULL CHECK (pid > 0),\n" +
			"  pid_start INTEGER NOT NULL DEFAULT 0\n" +
			") STRICT;\n" +
			"INSERT OR IGNORE INTO tui_session(id, screen, cwd, path, cursor, args) " +
			"SELECT item_key, json_extract(body, '$.screen'), coalesce(json_extract(body, '$.cwd'), ''), coalesce(json_extract(body, '$.path'), ''), " +
			"coalesce(json_extract(body, '$.cursor'), ''), CASE WHEN json_type(body, '$.args') = 'array' THEN json_extract(body, '$.args') END " +
			"FROM record WHERE app = 'tui' AND kind = 'session' AND json_valid(body) AND length(item_key) = 36 AND coalesce(json_extract(body, '$.screen'), '') <> '';\n" +
			"INSERT OR IGNORE INTO tui_active(session_id, pid, pid_start) " +
			"SELECT item_key, json_extract(body, '$.pid'), coalesce(json_extract(body, '$.pid_start'), 0) " +
			"FROM record WHERE app = 'tui' AND kind = 'active' AND json_valid(body) AND json_extract(body, '$.pid') > 0 AND item_key IN (SELECT id FROM tui_session);\n" +
			"DELETE FROM record WHERE app = 'tui'"},
		// Topics and ideas become tables. Every topic an idea names becomes a
		// topic. The idea id sequence starts past the old next counter, so a
		// removed id is never issued again. An idea record without text or a
		// numeric id cannot be shown and is not copied, and neither can a topic or
		// idea that breaks the length and one-line rules idealake add enforces.
		// model and embedding are NULL together when an idea was stored
		// without an embedding.
		{8, "idealake-tables", "" +
			"CREATE TABLE IF NOT EXISTS idealake_topic (\n" +
			"  id INTEGER PRIMARY KEY,\n" +
			"  name TEXT NOT NULL UNIQUE CHECK (name <> '' AND length(name) <= 80 AND name NOT GLOB '*[' || char(10, 13) || ']*'),\n" +
			"  created_at INTEGER NOT NULL\n" +
			") STRICT;\n" +
			"CREATE TABLE IF NOT EXISTS idealake_idea (\n" +
			"  id INTEGER PRIMARY KEY AUTOINCREMENT,\n" +
			"  topic_id INTEGER NOT NULL REFERENCES idealake_topic(id) ON DELETE RESTRICT,\n" +
			"  text TEXT NOT NULL CHECK (text <> '' AND length(text) <= 300 AND text NOT GLOB '*[' || char(10, 13) || ']*'),\n" +
			"  created_at INTEGER NOT NULL,\n" +
			"  model TEXT,\n" +
			"  embedding TEXT CHECK (embedding IS NULL OR json_type(embedding) = 'array'),\n" +
			"  CHECK ((model IS NULL) = (embedding IS NULL))\n" +
			") STRICT;\n" +
			"CREATE INDEX IF NOT EXISTS idealake_idea_topic ON idealake_idea(topic_id);\n" +
			"INSERT OR IGNORE INTO idealake_topic(name, created_at) " +
			"SELECT item_key, coalesce(unixepoch(json_extract(body, '$.created')), unixepoch()) FROM record " +
			"WHERE app = 'idealake' AND kind = 'topic' AND item_key <> '' AND json_valid(body);\n" +
			"INSERT OR IGNORE INTO idealake_topic(name, created_at) " +
			"SELECT json_extract(body, '$.topic'), min(coalesce(unixepoch(json_extract(body, '$.created')), unixepoch())) FROM record " +
			"WHERE app = 'idealake' AND kind = 'idea' AND json_valid(body) AND coalesce(json_extract(body, '$.topic'), '') <> '' GROUP BY 1;\n" +
			"INSERT OR IGNORE INTO idealake_idea(id, topic_id, text, created_at, model, embedding) " +
			"SELECT CAST(r.item_key AS INTEGER), t.id, json_extract(r.body, '$.text'), coalesce(unixepoch(json_extract(r.body, '$.created')), unixepoch()), " +
			"CASE WHEN json_type(r.body, '$.embedding') = 'array' THEN coalesce(json_extract(r.body, '$.model'), '') END, " +
			"CASE WHEN json_type(r.body, '$.embedding') = 'array' THEN json_extract(r.body, '$.embedding') END " +
			"FROM record r JOIN idealake_topic t ON t.name = json_extract(r.body, '$.topic') " +
			"WHERE r.app = 'idealake' AND r.kind = 'idea' AND json_valid(r.body) AND r.item_key <> '' AND r.item_key NOT GLOB '*[^0-9]*' " +
			"AND coalesce(json_extract(r.body, '$.text'), '') <> '';\n" +
			"INSERT INTO sqlite_sequence(name, seq) SELECT 'idealake_idea', 0 " +
			"WHERE NOT EXISTS (SELECT 1 FROM sqlite_sequence WHERE name = 'idealake_idea');\n" +
			"UPDATE sqlite_sequence SET seq = max(seq, coalesce((SELECT max(id) FROM idealake_idea), 0), " +
			"coalesce((SELECT CAST(body AS INTEGER) FROM record WHERE app = 'idealake' AND kind = 'meta' AND item_key = 'next'), 0)) " +
			"WHERE name = 'idealake_idea';\n" +
			"DELETE FROM record WHERE app = 'idealake'"},
		// A looptab jobs file and a Drive sync entry become rows. A sync entry
		// belongs to one preset and registration from the operator's Drive
		// config. A state record key is <preset>-<registration>. An entry
		// without a relpath or a file or dir kind is not copied, and neither is a
		// jobs file whose name is not the one lowercase word looptab add derives.
		{9, "looptab-gdrive-tables", "" +
			"CREATE TABLE IF NOT EXISTS looptab_file (\n" +
			"  id INTEGER PRIMARY KEY,\n" +
			"  name TEXT NOT NULL UNIQUE CHECK (name <> '' AND name <> '.' AND name = lower(name) AND name NOT GLOB '*[ /' || char(9) || ']*'),\n" +
			"  path TEXT NOT NULL CHECK (path <> '')\n" +
			") STRICT;\n" +
			"CREATE TABLE IF NOT EXISTS gdrive_sync_entry (\n" +
			"  id INTEGER PRIMARY KEY,\n" +
			"  preset TEXT NOT NULL CHECK (preset <> ''),\n" +
			"  registration TEXT NOT NULL CHECK (registration <> ''),\n" +
			"  relpath TEXT NOT NULL CHECK (relpath <> ''),\n" +
			"  kind TEXT NOT NULL CHECK (kind IN ('file', 'dir')),\n" +
			"  drive_id TEXT NOT NULL,\n" +
			"  size INTEGER NOT NULL DEFAULT 0 CHECK (size >= 0),\n" +
			"  mtime_ns INTEGER NOT NULL DEFAULT 0,\n" +
			"  sha1 TEXT NOT NULL DEFAULT '',\n" +
			"  UNIQUE (preset, registration, relpath)\n" +
			") STRICT;\n" +
			"INSERT OR IGNORE INTO looptab_file(name, path) " +
			"SELECT json_extract(f.value, '$.name'), json_extract(f.value, '$.path') FROM record r, json_each(r.body, '$.files') f " +
			"WHERE r.app = 'looptab' AND r.kind = 'files' AND json_valid(r.body) AND coalesce(json_extract(f.value, '$.name'), '') <> '' " +
			"AND coalesce(json_extract(f.value, '$.path'), '') <> '' ORDER BY f.key;\n" +
			gdriveStateMove + ";\n" +
			"DELETE FROM record WHERE app = 'looptab'"},
		// An OAuth token becomes a row keyed by app and account email. Its
		// document stays the google-auth authorized-user JSON, so the same
		// token works with google-auth tools and unknown keys are kept.
		{10, "google-token-table", "" +
			"CREATE TABLE IF NOT EXISTS google_token (\n" +
			"  id INTEGER PRIMARY KEY,\n" +
			"  app TEXT NOT NULL CHECK (app IN ('gmail', 'gcal', 'gdrive')),\n" +
			"  email TEXT NOT NULL CHECK (email <> ''),\n" +
			"  document TEXT NOT NULL CHECK (json_valid(document) AND json_type(document) = 'object'),\n" +
			"  UNIQUE (app, email)\n" +
			") STRICT;\n" + tokenMove},
		// A usage day becomes the number of whole UTC days since 1970-01-01.
		// A usage row keeps its counter rules: a known actor, at least one
		// run, and no more successes than runs. A row that breaks them or
		// whose day is not a YYYY-MM-DD date is not copied.
		{11, "usage-checks", "" +
			"CREATE TABLE usage_v2 (\n" +
			"  id INTEGER PRIMARY KEY,\n" +
			"  feature TEXT NOT NULL CHECK (feature <> ''),\n" +
			"  actor TEXT NOT NULL CHECK (actor IN ('human', 'agent', 'service')),\n" +
			"  day INTEGER NOT NULL CHECK (day >= 0),\n" +
			"  n INTEGER NOT NULL CHECK (n >= 1),\n" +
			"  ok INTEGER NOT NULL CHECK (ok BETWEEN 0 AND n),\n" +
			"  last INTEGER NOT NULL,\n" +
			"  UNIQUE (feature, actor, day)\n" +
			") STRICT;\n" +
			"INSERT OR IGNORE INTO usage_v2(id, feature, actor, day, n, ok, last) SELECT id, feature, actor, " +
			"CASE WHEN typeof(day) = 'integer' THEN day ELSE unixepoch(day) / 86400 END, n, ok, last FROM usage " +
			"WHERE typeof(day) = 'integer' OR (typeof(day) = 'text' AND date(day) = day);\n" +
			"DROP TABLE usage;\n" +
			"ALTER TABLE usage_v2 RENAME TO usage"},
	}
}

// gdriveStateMove copies Drive sync state records into gdrive_sync_entry and
// removes them. The legacy file import runs it again after importing.
const gdriveStateMove = "" +
	"INSERT OR IGNORE INTO gdrive_sync_entry(preset, registration, relpath, kind, drive_id, size, mtime_ns, sha1) " +
	"SELECT substr(r.item_key, 1, instr(r.item_key, '-') - 1), substr(r.item_key, instr(r.item_key, '-') + 1), " +
	"json_extract(e.value, '$.relpath'), json_extract(e.value, '$.kind'), coalesce(json_extract(e.value, '$.drive_id'), ''), " +
	"coalesce(json_extract(e.value, '$.size'), 0), coalesce(json_extract(e.value, '$.mtime_ns'), 0), coalesce(json_extract(e.value, '$.sha1'), '') " +
	"FROM record r, json_each(r.body, '$.entries') e " +
	"WHERE r.app = 'gdrive' AND r.kind = 'state' AND json_valid(r.body) AND instr(r.item_key, '-') > 1 " +
	"AND json_extract(e.value, '$.kind') IN ('file', 'dir') AND coalesce(json_extract(e.value, '$.relpath'), '') <> '';\n" +
	"DELETE FROM record WHERE app = 'gdrive' AND kind = 'state'"

// tokenMove copies OAuth token records into google_token and removes them.
// A token whose JSON is not an object cannot be read and is not copied.
const tokenMove = "" +
	"INSERT OR IGNORE INTO google_token(app, email, document) SELECT app, item_key, body FROM record " +
	"WHERE app IN ('gmail', 'gcal', 'gdrive') AND kind = 'token' AND item_key <> '' AND json_valid(body) AND json_type(body) = 'object';\n" +
	"DELETE FROM record WHERE app IN ('gmail', 'gcal', 'gdrive') AND kind = 'token'"

// schemaText is every migration in order, for the SQLX_ROOT schema file.
func schemaText() string {
	var b strings.Builder
	b.WriteString(migrationTable + ";\n")
	for _, m := range migrations() {
		b.WriteString("-- " + strconv.Itoa(m.version) + " " + m.name + "\n" + m.sql + ";\n")
	}
	return b.String()
}

// migrate applies every migration newer than the database's latest version.
// A process that waited for another to apply the same version finds its
// ledger row taken, rolls back, and goes on.
func migrate(path string) error {
	if err := sqliteExec(path, migrationTable); err != nil {
		return err
	}
	applied, err := appliedVersion(path)
	if err != nil {
		return err
	}
	for _, m := range migrations() {
		if m.version <= applied {
			continue
		}
		if err := sqliteExec(path, migrationScript(m)); err != nil {
			if now, verr := appliedVersion(path); verr == nil && now >= m.version {
				continue
			}
			return err
		}
	}
	return nil
}

func appliedVersion(path string) (int, error) {
	var rows []struct {
		V int `json:"v"`
	}
	if err := sqliteQueryJSON(path, "SELECT COALESCE(MAX(version), 0) AS v FROM schema_migration", &rows); err != nil {
		return 0, err
	}
	if len(rows) == 0 {
		return 0, nil
	}
	return rows[0].V, nil
}

// migrationScript claims the version's ledger row first, so the sql runs at
// most once even when two processes race.
func migrationScript(m migration) string {
	return ".bail on\nBEGIN IMMEDIATE;\n" +
		"INSERT INTO schema_migration(version, name, applied_at) VALUES (" +
		strconv.Itoa(m.version) + ", " + sqlQuote(m.name) + ", " + strconv.FormatInt(time.Now().UTC().Unix(), 10) + ");\n" +
		m.sql + ";\nCOMMIT;\n"
}
