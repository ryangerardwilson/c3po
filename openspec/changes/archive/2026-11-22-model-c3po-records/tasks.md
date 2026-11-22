## 1. Store each fact once

- [x] 1.1 Keep only the process in the `tui` active row and stop repeating the session id in its body
- [x] 1.2 Keep slack `last_read` only in its column; drop the copied message ts, user, text, event id, and string key

## 2. Version and model the c3po database

- [x] 2.1 Replace `app_once` and the column applier with the ordered `schema_migration` ledger
- [x] 2.2 Keep a record's id across writes and make `(app, kind, item_key)` unique
- [x] 2.3 Move tui sessions, idealake ideas and topics, looptab files, Drive sync state, and OAuth tokens into their own tables with foreign keys and checks
- [x] 2.4 Check `usage` in the schema and store its day as a number
- [x] 2.5 Remove the generic record functions; add `Query`, `Exec`, `ExecLocal`, `Quote`, and the token functions; run sqlite with foreign keys on
- [x] 2.6 Publish when any table holds data

## 3. Version the slack event cache

- [x] 3.1 Move a version 1 cache to version 2 in one transaction with references and checks

## 4. Own each fact once

- [x] 4.1 Add `c3po __session` and move nebula's tunnel screen and tesseract onto it
- [x] 4.2 Derive a cached slack message's unread state from `last_read`; route mark-read through the conversation write path; store a conversation and its messages in one transaction
- [x] 4.3 Delete Drive sync state before removing a registration, and clear it when registering
- [x] 4.4 Remove the legacy file import

## 5. Point dependencies inward

- [x] 5.1 Move the shared function scorer to `internal/score` and `internal/funcparse` so `ast` and `cyclotree` do not import each other
- [x] 5.2 Split idealake into the `lake` core with `Store` and `Embedder` interfaces, a sqlite store, and an xAI embedder
- [x] 5.3 Give the Drive syncer a `StateStore` interface and move its sqlite and file state to `syncstate`

## 6. Check

- [x] 6.1 Test migrations on legacy fixtures, a failed migration, stable ids, and schema rejections
- [x] 6.2 Test the slack v1 to v2 move and `last_read` stored once
- [x] 6.3 Dry-run every migration on copies of the live c3po database and slack caches
- [x] 6.4 Run `GOWORK=off go test ./...` with the runtime directory pointed at a temporary directory
- [x] 6.5 Validate with `openspec validate model-c3po-records --strict --no-interactive`
