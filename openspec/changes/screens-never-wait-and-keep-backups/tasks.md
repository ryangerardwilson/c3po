## 1. Screens

- [x] 1.1 `internal/tuijob` serial worker; Slack `r`, `s`, enter, auto refresh off the loop with pending intents; `q` drains marks
- [x] 1.2 gdrive, pass, o, cyclotree, deport follow the screen rule
- [x] 1.3 `tuisession` latest-wins background writer

## 2. Backup

- [x] 2.1 `sqlxdb` marks through `nebula keep`, restores through `nebula keep restore`; c3po's own publisher and restore are deleted
- [x] 2.2 The schema ledger is checked once per process
- [x] 2.3 Installer retires `c3po-snapshot.timer`; `c3po __snapshot` marks

## 3. Verification

- [x] 3.1 `c3po ast exec -- openspec validate screens-never-wait-and-keep-backups --strict --no-interactive`
- [x] 3.2 `GOWORK=off go test ./...`
