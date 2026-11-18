## 1. Backup

- [x] 1.1 Restore waits, and a failed restore does not create or publish a database
- [x] 1.2 Publish waits, uploads a consistent copy, and skips a database with no records
- [x] 1.3 Screen places write locally, and the slack screen publishes every 10 minutes and on leave
- [x] 1.4 `c3po __snapshot` and `c3po-snapshot.timer` run that publish every 10 minutes

## 2. Verification

- [x] 2.1 `openspec validate slack-snapshot --strict --no-interactive`
- [x] 2.2 `GOWORK=off go test` for sqlxdb, tuisession, slack, scout, and usage
