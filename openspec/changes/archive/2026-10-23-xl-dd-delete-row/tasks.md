## 1. Spec

- [x] 1.1 Add the dd row requirement
- [x] 1.2 Validate the change with `openspec validate xl-dd-delete-row --strict --no-interactive`

## 2. Screen

- [x] 2.1 Delete the current row with an armed `dd` and shift formulas
- [x] 2.2 Cancel the arm with Esc and with any other key
- [x] 2.3 Name `dd` in the help

## 3. Check

- [x] 3.1 Run `GOWORK=off go test ./xl/...`
