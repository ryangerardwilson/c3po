## 1. Store

- [x] 1.1 Store gmail, gcal, and gdrive tokens, gdrive sync state, and tunnel routes in the sqlx database `c3po`
- [x] 1.2 Import the legacy files once and leave them on disk

## 2. Spec

- [x] 2.1 Record the database in the scout spec
- [x] 2.2 Validate with `openspec validate c3po-sqlx --strict --no-interactive`

## 3. Check

- [x] 3.1 Run `GOWORK=off go test ./internal/sqlxdb ./gmail/... ./gcal/... ./gdrive/... ./tunnel/...`
- [x] 3.2 Install c3po from this checkout and confirm the imported row counts
