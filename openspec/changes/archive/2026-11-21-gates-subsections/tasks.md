## 1. Command

- [x] 1.1 Find the immediate subsections of a section: one more number, in section order, never a grandchild
- [x] 1.2 Print a section's body, then a blank line and one `<section>  <heading>` line per immediate subsection when it has any
- [x] 1.3 Keep a leaf section to only its body and one newline
- [x] 1.4 Print several sections in the order written, each block as the canonical section, its body, and its subsection lines
- [x] 1.5 Say in the help that a section with subsections lists those subsections under the body
- [x] 1.6 Update `README.md`, `AGENTS.md`, and the gates sentence in `/home/ryan/AGENTS.md`

## 2. Check

- [x] 2.1 Test that `8`, `§8`, and `S.8` print the `§8` body and `§8.1` through `§8.4` without their bodies
- [x] 2.2 Test that `1` lists `§1.1  Domain truth` and not the `§1.1` body
- [x] 2.3 Test that a statute with `§8.1.2.3` lists it under neither `§8` nor `§8.1`, and lists `§8.1.1` through `§8.1.6` under `§8.1`
- [x] 2.4 Test that a leaf section prints only its body and that several sections keep their order and blocks
- [x] 2.5 Run `GOWORK=off go test ./gates/ ./internal/scout/ ./internal/usage/` and `c3po ast gap`
- [x] 2.6 Validate with `openspec validate gates-subsections --strict --no-interactive` and archive it
