## 1. Statute

- [x] 1.1 Append `§10 Tests` and `§10.1` through `§10.20` to `architecture` in `gates/architecture.go`, leaving `§1` through `§9` byte for byte
- [x] 1.2 Name tests in the comment above `var architecture`

## 2. Check

- [x] 2.1 Add `§10` with its heading and body to the top-section test, and expect 140 blocks and 140 provisions for `all`
- [x] 2.2 Recompute the tally counts from 140: `140 of 140`, `139 of 140`, `137 of 140`, and line 141 for the extra line
- [x] 2.3 Probe `§11` and `11` as unknown sections in place of `§10` and `10`, and leave the sort fixture that uses a fictional `§10`
- [x] 2.4 Run `GOWORK=off go test ./gates/ -count=1`
- [x] 2.5 Validate with `openspec validate add-test-architecture --strict --no-interactive` and archive it
