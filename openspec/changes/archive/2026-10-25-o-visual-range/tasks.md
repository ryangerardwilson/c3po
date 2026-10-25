## 1. Spec

- [x] 1.1 Add the visual range requirement and extend cut, copy, paste, and dd
- [x] 1.2 Validate the change with `openspec validate o-visual-range --strict --no-interactive`

## 2. Screen

- [x] 2.1 Toggle a consecutive range with `v` and highlight it
- [x] 2.2 Delete the range with the Delete key and with `dd`
- [x] 2.3 Cut and copy the range, and paste every staged item
- [x] 2.4 Name the keys in the shortcut list and the help

## 3. Check

- [x] 3.1 Run `GOWORK=off go test ./o/...`
