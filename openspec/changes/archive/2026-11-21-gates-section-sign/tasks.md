## 1. Command

- [x] 1.1 Parse a section as `§` and then dot-separated positive integers, with no space and no letter
- [x] 1.2 Renumber the `architecture` provisions from `S.` to `§` without changing order, headings, or bodies
- [x] 1.3 Show `§1.1` in the help example and `§1  pass`, `§1  fail`, and `§1.1  fail  not judged` in the report lines
- [x] 1.4 Update `README.md`, `AGENTS.md`, and the gates sentence in `/home/ryan/AGENTS.md`

## 2. Check

- [x] 2.1 Test that `§1`, `§2.3`, `§3.5.1`, and `§10` parse and sort as `§1`, `§2`, `§2.3`, `§3`, `§3.5.1`, `§10`
- [x] 2.2 Test that `S.1`, `s.1`, `§ 1`, `§.1`, `1`, `§0`, `§1.0`, and `§01` are refused and are unknown sections
- [x] 2.3 Test that the provision list starts `§1  Data model` then `§1.1  Domain truth` and that the help has no `S.1`
- [x] 2.4 Test that `c3po gates code architecture §1` is recorded as `gates`
- [x] 2.5 Run `GOWORK=off go test ./gates/ ./internal/scout/ ./internal/usage/` and `c3po ast gap`
- [x] 2.6 Validate with `openspec validate gates-section-sign --strict --no-interactive` and archive it
