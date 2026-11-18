# Slack screen backup

## Why

The slack screen stores its place in the c3po sqlite file on the runtime tmpfs. Each key publishes that whole file in a 3-second background upload. An abrupt power loss kills the upload. The next start treats a failed restore as an empty database and the next write publishes that empty file over `s3://backups/c3po.sqlite`. That object also holds tokens, Drive state, tunnel routes, and ideas.

## What Changes

- A missing runtime database is loaded from `s3://backups/c3po.sqlite` and the load waits up to 60 seconds.
- A failed load does not create a database and does not publish. A missing object may create an empty database, and that database is not published.
- A database with no `record` rows is not published. A publish waits up to 60 seconds and uploads a consistent copy.
- Screen places are written locally. They are not published on each key.
- The slack screen publishes about every 10 minutes while it is open, and once when it leaves.
- A user timer runs `c3po __snapshot` every 10 minutes. That command publishes the runtime database when it has a record, and does nothing when the file is missing.
- `SQLX_ROOT` still skips publish and restore.

## Capabilities

### New Capabilities

### Modified Capabilities

- `tui`: Screen places stay in the runtime database and the backup is a 10-minute snapshot that refuses an empty database.
- `slack`: The slack screen snapshots its place on that interval and on close, and a later open loads the last good backup.

## Impact

- Specs: `openspec/specs/tui/spec.md`, `openspec/specs/slack/spec.md`
- Code: `internal/sqlxdb`, `internal/tuisession`, `slack/internal/tui`, `internal/scout`, `install.sh`
- The installer enables `c3po-snapshot.timer`. It does not run a snapshot during install.
- `c3po slack` and the Quickshell desk are unchanged. The session still does not store the user token.
