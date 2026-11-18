## 1. Sync

- [x] 1.1 Remove the watching-after line and say `no items` when a workspace has no rows
- [x] 1.2 `s` refreshes the desk from Slack, and the screen shows `last synced at`

## 2. Verification

- [x] 2.1 `openspec validate slack-sync --strict --no-interactive`
- [x] 2.2 `GOWORK=off go test` for the slack screen and the screen package
