## 1. Command

- [x] 1.1 Print `receipt <8 hex>` after every body, from SHA-256 over the version, the canonical section, and the body
- [x] 1.2 After a parent's subsection titles, print the count of subsection bodies not in this output and the command that fetches them, in brace form when they are `.1` through `.N`
- [x] 1.3 Add `all`: every provision in section order as section, body, and receipt, with no title index and no closing line
- [x] 1.4 Add `tally <sheet>`: count good lines, print missing with a fetch command, unfixed, unclaimed, and each bad line; read changed files from `git status --porcelain`
- [x] 1.5 Replace the sequential rule in the help; show `all`, `tally`, the receipt line, and a sheet example
- [x] 1.6 Update `README.md`, `AGENTS.md`, and the gates sentence in `/home/ryan/AGENTS.md`

## 2. Check

- [x] 2.1 Test the receipt of a leaf and of `§8`, and that title lines carry none
- [x] 2.2 Test the `§8` closing line and that shown subsections are not counted
- [x] 2.3 Test that `all` holds 93 blocks, the `§8.3` leaf body, and no closing line
- [x] 2.4 Test tally at 93 of 93, a missing section with its fetch command, a bad receipt, a missing path, a duplicate, and an unclaimed changed file, all in a temporary git repository
- [x] 2.5 Run `GOWORK=off go test ./gates/ ./internal/scout/ ./internal/usage/` and `c3po ast gap`
- [x] 2.6 Validate with `openspec validate gates-receipts-tally --strict --no-interactive` and archive it
