## 1. Automatic sync

- [x] 1.1 An open slack screen asks the desk to refresh about every 10 minutes on the same path as `s`
- [x] 1.2 The screen does not refresh on open, does not refresh while `?` is open, refreshes after that list closes when the interval has elapsed, and starts the wait again after `s`
- [x] 1.3 Help, the README, and AGENTS.md say the screen syncs about every 10 minutes

## 2. Verification

- [x] 2.1 `openspec validate slack-auto-sync --strict --no-interactive`
- [x] 2.2 `GOWORK=off go test` for the slack screen and the screen package
