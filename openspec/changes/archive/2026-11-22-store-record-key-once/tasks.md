## 1. Store each fact once

- [x] 1.1 Make `runOnce` apply the sql and record its id in one transaction that stops on the first error, and report whether it ran
- [x] 1.2 Add `sqlxdb.Once` and `sqlxdb.OnceFor`; publish only when the migration ran
- [x] 1.3 Stop storing `id` in an idea body and `topic` in a topic body; take both from the record key; run `idealake-body-key-1` before listing
- [x] 1.4 Stop storing `parent_relpath` in Drive sync state; derive it on load; run `gdrive-state-parent-1` before loading from sqlx

## 2. Check

- [x] 2.1 Test that a migration runs once, that a failed migration commits nothing and runs again, and that a file path runs nothing
- [x] 2.2 Test that new idea and topic bodies hold no key copy, and that old bodies lose it and still load
- [x] 2.3 Test that saved Drive state has no parent path, and that old state loses it and loads the derived parent
- [x] 2.4 Run `GOWORK=off go test ./...` and `c3po ast gap`
- [x] 2.5 Validate with `openspec validate store-record-key-once --strict --no-interactive` and archive it
