## 1. Spec

- [x] 1.1 Validate `revive-slack` with `openspec validate revive-slack --strict --no-interactive`

## 2. Command

- [x] 2.1 Register `c3po slack` and keep `c3po sluser` unknown
- [x] 2.2 `GOWORK=off go test ./slack/... ./internal/scout/... ./internal/xdg/...` passes

## 3. Accounts

- [x] 3.1 Adopt `~/.config/sluser` and `~/.local/state/sluser` under `c3po/slack`, and leave `~/.config/slack` in place
