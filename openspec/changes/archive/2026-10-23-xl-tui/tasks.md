## 1. Spec

- [x] 1.1 Add the xl spec, the scout screen delta, and the o workbook delta
- [x] 1.2 Validate the change with `openspec validate xl-tui --strict --no-interactive`

## 2. Screen

- [x] 2.1 Move the spreadsheet packages into `xl/` and register `c3po tui xl`
- [x] 2.2 Select that screen when the program name is `xl`
- [x] 2.3 List the screen from `c3po tui` and from help

## 3. Navigator

- [x] 3.1 Open an `.xlsx` file in the xl screen from a direct path and from the navigator
- [x] 3.2 Leave text files on the editor and other files on xdg-open

## 4. Check

- [x] 4.1 Run `GOWORK=off go test ./...`
- [x] 4.2 Install from this checkout and confirm `c3po tui xl help` and `xl version`
