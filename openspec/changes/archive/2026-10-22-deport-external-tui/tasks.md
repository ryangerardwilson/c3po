## 1. Spec

- [x] 1.1 Add the deport spec and the scout screen delta
- [x] 1.2 Validate the change with `openspec validate deport-external-tui --strict --no-interactive`

## 2. Command

- [x] 2.1 Refuse a port outside 8000 through 8999 before any signal
- [x] 2.2 Add `c3po deport list` for occupied ports in that band
- [x] 2.3 Cover an internal port, an empty list, and the edges 8000 and 8999

## 3. Screen

- [x] 3.1 Add `c3po tui deport` with `j` and `k` and armed `dd`
- [x] 3.2 List the screen from `c3po tui` and from help
- [x] 3.3 Point the listener test at a free port inside the band

## 4. Check

- [x] 4.1 Run `GOWORK=off go test ./...`
- [x] 4.2 Install from this checkout and confirm `c3po tui deport help`
