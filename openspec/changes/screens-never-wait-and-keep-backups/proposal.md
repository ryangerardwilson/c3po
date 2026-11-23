## Why

Screens hosted in c3po froze on the network. `r` on the Slack screen waited about 3.5 s, up to 9 s, on the desk helper's synchronous upload; `s` froze every key for the whole refresh; leaving the screen waited about 6 s on a publish of the c3po database; every key ran about six `sqlite3` processes to store the place. gdrive, pass, o, cyclotree, and deport ran Drive HTTP, `nebula crypt`, commands, and process kills on the key loop. c3po also carried its own copy of the backup: a snapshot, an empty-file guard, an `s3 make bucket` before every publish, and a restore.

## What Changes

- A screen never waits before it paints: a key changes only UI state and pending intents, paints, and queues any write or remote call on one serial worker with a timeout (`internal/tuijob`). The owner's answer replaces the screen's copy whole; a failure reverts the view.
- The Slack screen: `r` drops the row at once and restores it if the desk refuses; `s` and the 10-minute refresh run off the loop; `q` drains pending marks and does not publish.
- The session place is written by a background latest-wins writer only when it changes; `Close` stores the last place.
- `sqlxdb` commits and runs `nebula keep mark c3po <path>`; a missing file is restored with `nebula keep restore`; c3po's own upload, snapshot, empty-file guard, and restore are deleted. The schema ledger is checked once per process. `c3po __snapshot` stays one release as a mark. The installer retires `c3po-snapshot.timer`.

## Capabilities

### Modified Capabilities

- `slack`: the screen's place and leaving no longer publish; `r`, `s`, and `q` change as above.
- `tui`: the backup requirement moves to nebula keep; a new requirement says a screen never waits before it paints.
- `scout`: the usage worker marks the database instead of publishing it.

## Deliberate exceptions (statute 8)

- `internal/sqlxdb` caches, per process, which runtime files are already migrated (statute 2.11). It is dropped when a file is restored, and can go once sqlxdb holds one handle per process.
- The `record` and `meta` tables in `c3po.sqlite` belong to nebula's tunnel, not to c3po (statute 2.2, 2.4, 3.5). This is older than this change; they go in a c3po migration once nebula keeps its routes in its own database, which needs `tunnel.service` restarted on that nebula.
- `c3po __snapshot` stays one release as an alias that marks the database, because timers and agents may still name it.

## Impact

Install nebula with `keep` first. `$XDG_RUNTIME_DIR/c3po/c3po.sqlite`, `s3://backups/c3po.sqlite`, and the schema ledger are unchanged; nothing moves. r2d2 gets these screens when it bumps its c3po pin. Rollback is the previous c3po binary.
