## Why

`c3po gates code architecture §1.3` requires every fact to be stored in exactly one place, with derived values computed, not stored. Three c3po records break it. An idea body repeats its record key as `id`. A topic body repeats its record key as `topic`. Each Drive sync state entry stores `parent_relpath`, which is always the directory part of its `relpath`. A reader that trusted the copy instead of the key or the relpath could disagree with the rest of the store, and nothing keeps the two in step.

## What Changes

- An idea record body SHALL NOT hold `id`. The record key is the idea id, and loading an idea SHALL take the id from the key.
- A topic record body SHALL NOT hold `topic`. The record key is the topic name, and listing topics SHALL take the name from the key.
- A Drive sync state entry SHALL NOT hold `parent_relpath`. Loading state SHALL derive the parent from `relpath`.
- `sqlxdb.Once(id, sql)` runs one data migration one time in the c3po database. The sql and the `app_once` row for its id commit in one transaction, an error commits neither, and the database is published only when the migration ran. `sqlxdb.OnceFor(path, id, sql)` runs it only for a production record path.
- Idealake runs `idealake-body-key-1` before it lists ideas or topics. It removes `$.id` from idea bodies and `$.topic` from topic bodies.
- gdrive runs `gdrive-state-parent-1` before it loads state. It removes `parent_relpath` from every entry of every stored state.
- Command output does not change. `--json` still names each idea id. The version does not change.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `idealake`: an idea body and a topic body do not repeat the record key.
- `scout`: Drive sync state stores no parent path, and a stored-data migration runs once in one transaction.

## Impact

- Specs: `openspec/specs/idealake/spec.md`, `openspec/specs/scout/spec.md`
- Code: `internal/sqlxdb/sqlxdb.go`, `internal/sqlxdb/localdb.go`, `idealake/store.go`, `gdrive/internal/syncer/state.go`
- Tests: `internal/sqlxdb/once_test.go`, `idealake/key_test.go`, `gdrive/internal/syncer/state_test.go`
- Migration: the first idealake or gdrive command after install rewrites the affected bodies in place. Backfill: none is needed, because the key and the relpath already hold the facts. A body whose JSON is invalid is left unchanged. Rollback: an older binary reads migrated ideas, because it already fell back to the key when `id` was empty, and an older gdrive treats a missing parent as the root. To restore the old shape before running an older binary, run in the c3po sqlite:
  `UPDATE record SET body = json_set(body, '$.topic', item_key) WHERE app = 'idealake' AND kind = 'topic';`
  `UPDATE record SET body = json_set(body, '$.entries', (SELECT json_group_array(json_set(value, '$.parent_relpath', rtrim(rtrim(json_extract(value, '$.relpath'), replace(json_extract(value, '$.relpath'), '/', '')), '/'))) FROM json_each(record.body, '$.entries'))) WHERE app = 'gdrive' AND kind = 'state' AND json_type(body, '$.entries') = 'array';`
  and delete the two `app_once` rows.
- Not changed here: the `tui` `active` record repeats the session id, screen, and workspace of its session record, but tesseract reads those fields from the active body and nebula writes the same records, so that copy needs a change in those repositories first. The slack event cache keeps `last_read` both as a column and inside `info_json`.
