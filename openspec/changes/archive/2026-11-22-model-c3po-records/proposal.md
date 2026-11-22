## Why

`c3po gates code architecture` §1.3 through §1.8 fail on the c3po stores. The generic `record` table keeps every app's data as `(app, kind, item_key, body)` JSON. The model has no foreign keys, no unique key, and no checks. A write deletes the row and inserts it again, so a row's id changes on every write. Its schema grows by ad hoc `ALTER TABLE ADD COLUMN`. The `tui` active body repeats the session id, screen, and workspace that its key and its session already hold. The slack event cache keeps `last_read` both as a column and inside `info_json`. Each message repeats its ts, user, and text beside the payload. A message's key is a string built from channel and ts. Messages and events reference conversations, but nothing enforces those references.

## What Changes

- The c3po database gets an ordered, versioned migration ledger, `schema_migration(version, name, applied_at)`. It replaces the unordered `app_once` ledger and the model applier that added missing columns. Each migration commits together with its ledger row, and each one is safe to apply twice.
- Screen sessions move to `tui_session` and `tui_active`. A `tui_active` row holds only `pid` and `pid_start`, and it is deleted with its session.
- Ideas and topics move to `idealake_idea` and `idealake_topic`. An idea references its topic by foreign key. The idea id sequence keeps the old `next` counter, so a removed id is never issued again.
- Looptab registrations move to `looptab_file`. Drive sync state moves to `gdrive_sync_entry`, one row per entry, unique on preset, registration, and relpath.
- OAuth tokens move to `google_token`, unique on app and email. The document stays the google-auth authorized-user JSON, so the token works with google-auth tools and unknown keys are kept.
- `usage` stores its day as whole UTC days since 1970-01-01 and gets checks on actor, `n >= 1`, and `ok <= n`. New tables are `STRICT`. Idea and topic length and one-line rules, the looptab one-word name, and enum columns are checked in the schema.
- A `record` row keeps its id across writes, and `(app, kind, item_key)` is unique. `record` and `meta` now hold only nebula's tunnel routes and import flag, and c3po neither reads nor writes them. The generic record functions are removed from `sqlxdb`, and apps read and write their own tables through `sqlxdb.Query` and `sqlxdb.Exec`. sqlite runs with foreign keys on.
- The one-time import of legacy token, Drive state, and tunnel route files is removed. It ran when those files moved into the database, and it wrote other modules' tables around their write paths. nebula imports its own routes.
- c3po is the one writer of screen sessions. `c3po __session` lets nebula's tunnel screen open, note, place, and close its session, and lets tesseract list running screens, without either touching the tables.
- Removing a Drive registration deletes its sync state first and stops on failure; registering clears state an earlier registration with the same id left. Registration ids are reused, so leftover state could otherwise pass to a different folder.
- The backup is skipped only when no table holds data. Before this change it was skipped whenever `record` was empty.
- The slack event cache goes to schema version 2. `last_read` lives only in its column. A message keeps its ts only in `message_json`, and `ts` and `sort_ts` become generated columns. The copied user, text, event id, unread flag, and string key are dropped; a message is unread when it is after `last_read` and not the operator's own. Marking read goes through the one conversation write path, and a conversation and its messages are stored in one transaction. Times are unix seconds in UTC, and the last sync is one typed `sync_state` row. Messages and events reference conversations with `ON DELETE CASCADE`. Surface, flags, and the unread mark are checked.
- The function scorer that `ast` and `cyclotree` share moves to `internal/score` and `internal/funcparse`, so the two modules no longer import each other. idealake's rules move to a core package that reaches storage and embeddings through interfaces it defines. The Drive syncer reaches its state through a `StateStore` interface.
- Command output does not change. The version does not change.

## Capabilities

### New Capabilities

- `slack-event-cache`: the slack event cache stores each fact once, is versioned, and enforces its references.

### Modified Capabilities

- `scout`: versioned migrations replace one-time named migrations; Drive sync state is a table.
- `idealake`: ideas and topics are tables related by a foreign key.
- `tui`: a session and its active process are tables; the backup guard counts every table.

## Impact

- Code: `internal/sqlxdb`, `internal/tuisession`, `idealake`, `looptab`, `gdrive/internal/syncer`, `gdrive/internal/auth`, `gmail`, `gcal`, `slack/internal/app`.
- Other repositories: tesseract lists running screens with `c3po __session active`. nebula's tunnel screen keeps its session through `c3po __session`. nebula's record writes keep the row id.
- Migration: the first c3po command after install runs migrations 1 through 11 on the runtime database. A copy of the live database moved 70 sessions, 9 active rows, 118 ideas, 2 topics, 112 Drive entries, and 8 tokens. The idea sequence stayed at 132 and the foreign-key and integrity checks passed. The slack cache migrates on its next open. A copy of the live caches kept all 243 conversations and 5485 messages.
- Backfill: every value is copied from the record body that held it. A topic is created for every topic an idea names. Rows the old code could not load are not copied: a session with a bad id or no screen, an active row whose session is missing, an idea without text or a numeric id, a Drive entry without a relpath or a file or dir kind, a token that is not a JSON object, a usage row that breaks the counter rules, and an idea, topic, or jobs file that breaks the add rules. A slack message or event whose conversation is missing gets a placeholder conversation.
- Rollback: back up `$XDG_RUNTIME_DIR/c3po/c3po.sqlite` before installing. To run an older binary on a migrated database, restore that backup. If none exists, copy the tables back into `record`:
  `INSERT INTO record(app, kind, item_key, body) SELECT 'tui', 'session', id, json_object('screen', screen, 'cwd', cwd, 'path', path, 'cursor', cursor, 'args', json(args)) FROM tui_session;`
  `INSERT INTO record(app, kind, item_key, body) SELECT 'idealake', 'idea', i.id, json_object('topic', t.name, 'text', i.text, 'created', strftime('%Y-%m-%dT%H:%M:%SZ', i.created_at, 'unixepoch'), 'model', i.model, 'embedding', json(i.embedding)) FROM idealake_idea i JOIN idealake_topic t ON t.id = i.topic_id;`
  `INSERT INTO record(app, kind, item_key, body) SELECT 'idealake', 'meta', 'next', seq FROM sqlite_sequence WHERE name = 'idealake_idea';`
  `INSERT INTO record(app, kind, item_key, body) SELECT 'looptab', 'files', 'files', json_object('files', json_group_array(json_object('name', name, 'path', path))) FROM looptab_file;`
  `INSERT INTO record(app, kind, item_key, body) SELECT 'gdrive', 'state', preset || '-' || registration, json_object('entries', json_group_array(json_object('relpath', relpath, 'kind', kind, 'drive_id', drive_id, 'size', size, 'mtime_ns', mtime_ns, 'sha1', sha1))) FROM gdrive_sync_entry GROUP BY preset, registration;`
  `INSERT INTO record(app, kind, item_key, body) SELECT app, 'token', email, document FROM google_token;`
  `CREATE TABLE usage_old AS SELECT id, feature, actor, date(day * 86400, 'unixepoch') AS day, n, ok, last FROM usage; DROP TABLE usage; ALTER TABLE usage_old RENAME TO usage; CREATE UNIQUE INDEX usage_feature_actor_day ON usage(feature, actor, day);`
  An older binary ignores the new tables and `schema_migration`, recreates `app_once`, and reruns its idempotent fixes. To roll back the slack cache, delete `events-<preset>.db`; it is a cache and the next sync refills it.
