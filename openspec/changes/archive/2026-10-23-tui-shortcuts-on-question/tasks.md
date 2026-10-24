## 1. Spec

- [x] 1.1 Add the shortcut standard and update the navigator height requirement
- [x] 1.2 Validate the change with `openspec validate tui-shortcuts-on-question --strict --no-interactive`

## 2. Screens

- [x] 2.1 Drop the navigator key line and give that row to the listing
- [x] 2.2 Drop the key line on deport, Drive browse, the preset list, pass, and the tunnel route list, and show it from `?`

## 3. Check

- [x] 3.1 Run `GOWORK=off go test ./o/... ./deport/... ./gdrive/... ./pass/... ./tunnel/...`
