## 1. Spec

- [x] 1.1 Add the looptab spec and the scout and tui deltas
- [x] 1.2 Validate the change with `openspec validate looptab-tui --strict --no-interactive`

## 2. Screen

- [x] 2.1 List jobs from `looptab now` and run the selected index with `looptab now <index>`
- [x] 2.2 Keep shortcut text behind `?`, and let `q` quit while that list is open
- [x] 2.3 Register `c3po tui looptab` and reject `c3po looptab`

## 3. Check

- [x] 3.1 Run `GOWORK=off go test` for the looptab screen and scout
- [x] 3.2 Install from this checkout and confirm `c3po tui looptab help`
