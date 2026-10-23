## 1. Spec

- [x] 1.1 Remove the requirement that the program name `xl` runs the spreadsheet
- [x] 1.2 Validate with `openspec validate drop-xl-link --strict --no-interactive`

## 2. Install

- [x] 2.1 Stop linking `xl` and remove a symlink that points at c3po
- [x] 2.2 Dispatch the program name `xl` as c3po, not the spreadsheet

## 3. Check

- [x] 3.1 Run `GOWORK=off go test ./internal/scout/ ./xl/...`
- [x] 3.2 Install and confirm `xl` is not on `PATH` and `c3po tui xl help` still works
