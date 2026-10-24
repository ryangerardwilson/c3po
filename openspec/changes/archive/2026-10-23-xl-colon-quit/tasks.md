## 1. Spec

- [x] 1.1 Add the save and quit requirement
- [x] 1.2 Validate the change with `openspec validate xl-colon-quit --strict --no-interactive`

## 2. Screen

- [x] 2.1 Save with `:w`, save and quit with `:wq`, and discard with `:q!`
- [x] 2.2 Leave `q`, ctrl-c, and `:q` on the sheet
- [x] 2.3 Name `:w`, `:wq`, and `:q!` in the help

## 3. Check

- [x] 3.1 Run `GOWORK=off go test ./xl/...`
