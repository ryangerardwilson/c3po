## 1. Statute

- [x] 1.1 Append `§9 Components and UX` and `§9.1` through `§9.25` to `architecture` in `gates/architecture.go`, leaving `§1` through `§8` byte for byte
- [x] 1.2 Name UI in the comment above `var architecture`

## 2. Check

- [x] 2.1 Add `§9` with its heading and body to the top-section test, and expect 119 blocks and 119 provisions for `all`
- [x] 2.2 Recompute the tally counts from 119: `119 of 119`, `118 of 119`, `116 of 119`, and line 120 for the extra line
- [x] 2.3 Probe `§10` and `10` as unknown sections in place of `§9` and `9`
- [x] 2.4 Run `GOWORK=off go test ./gates/ -count=1`
- [x] 2.5 Validate with `openspec validate add-ui-architecture --strict --no-interactive` and archive it
