# Slack screen backup

## Context

tesseract keeps sessions in `$XDG_RUNTIME_DIR/tesseract/tesseract.sqlite` and publishes `s3://backups/tesseract.sqlite`. The upload waits 60 seconds. An empty live session list does not replace the saved snapshot. A user timer runs every 15 minutes. Crusade skips publish when the task table is empty, so a damaged local copy cannot replace the backup.

c3po uses the same shape for `$XDG_RUNTIME_DIR/c3po/c3po.sqlite` and `s3://backups/c3po.sqlite`, but the upload is a 3-second background task and a failed restore creates an empty file. The slack screen writes its place on every key, so it schedules two of those uploads per key.

## Goals

- After power loss, the next open loads the last published database before it reads a session.
- A failed load does not create or publish a database.
- An empty `record` table is not published.
- The slack place is durable within about 10 minutes, and on a clean leave, without an upload per key.

## Non-goals

- A separate slack database.
- Changing the Quickshell Slack desk.
- Storing the user token in the session.
- Retargeting the usage worker's object. That worker already puts `s3://backups/c3po.sqlite`. It uses the same empty-record guard and the same consistent copy.
- Opening slack on the restored place when the operator did not pass `--resume`.

## Decisions

- Restore and publish wait up to 60 seconds and run in the foreground. Output that contains `not found` is a missing object. Any other restore error removes a partial download, returns the error, and does not create a file.
- Publish copies the database with `VACUUM INTO`, then puts that copy. The live file is not the upload body. A `record` table with zero rows is not published. A database with no `record` table, which is the usage worker's test fixture, may still be published.
- `Put`, `Write`, `Delete`, and `Remove` publish after the write. Screen session writes use a local put and a local delete, so a key does not publish.
- The slack screen starts a 10-minute ticker when it has a session, and publishes once when the screen loop returns. `sqlxdb.Publish` does nothing when `SQLX_ROOT` is set or the runtime file is missing.
- `c3po __snapshot` is that publish. `install.sh` writes `c3po-snapshot.service` and `c3po-snapshot.timer` with `OnActiveSec=10min`, `OnUnitActiveSec=10min`, and `AccuracySec=5s`, then enables the timer. It does not start the service during install.
- Idealake and token writes still publish, because they use `Put` and `Write`.

## Risks

- A publish waits up to 60 seconds when the backup host is unreachable. A slack leave waits for that attempt. A key does not.
- The timer and the open screen can publish the same database. The copy is consistent, and a database with no records is still skipped.
- A restored backup can be up to about 10 minutes behind the last key.
