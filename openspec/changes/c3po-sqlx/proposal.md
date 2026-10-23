## Why

c3po kept OAuth tokens, Drive sync state, and tunnel routes in JSON files. Those files are outside the sqlx backup that already covers the other laptop apps.

## What Changes

- OAuth tokens for gmail, gcal, and gdrive, gdrive sync state, and tunnel routes are stored in one sqlx database named `c3po`.
- Operator-edited config, credential files, signatures, the pass service-account token, generated cloudflared files, and caches stay files.
- Existing JSON files are imported once and left on disk. Later reads and writes use the database.

## Capabilities

### New Capabilities

### Modified Capabilities

- `scout`: Durable c3po records live in the sqlx database `c3po`.

## Impact

gmail, gcal, gdrive, and tunnel record IO. Config layout is unchanged. Rollback is the previous binary. The legacy files remain on disk.
