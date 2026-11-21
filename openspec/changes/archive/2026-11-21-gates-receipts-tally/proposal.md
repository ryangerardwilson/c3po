## Why

An agent can say it judged every provision without fetching a single body, and nothing shows the difference. The help also says a failed gate means later gates are not judged, so a report can stop at the first failure and leave most of the statute unread. A parent section lists its subsection headings but does not say their bodies were not printed or how to fetch them, so an agent can mistake a heading for the rule.

## What Changes

- Every printed body is followed by `receipt <8 hex>`: the first 8 hex characters of SHA-256 over `version.String()`, a newline, the canonical section, a newline, and the body. A title line has no receipt. No receipt is stored in source.
- A parent section lists its subsection titles and then `N bodies not shown: c3po gates <type> <statute> <args>` for the subsections whose bodies are not in the same output. The args are `8.{1..4}` when the missing subsections are `.1` through `.N` with no gap, otherwise the sections separated by spaces.
- `c3po gates <type> <statute> all` prints every provision in section order as its section, body, and receipt, with no title index and no closing line. `all` is not a section.
- `c3po gates <type> <statute> tally <sheet>` reads a sheet with one line per provision, `§1.2 <receipt> <pass|fail> <site> [files <path>...]`, and prints `<n> of <total> judged`, then the missing sections and the command that fetches them, the fails that name no files (`unfixed`), the changed files no line claims (`unclaimed`), and each bad line. It exits 0 only for a complete sheet whose receipts match this build, whose sites are `none` or a real `path:line`, and which claims every changed tracked file; an unfixed fail in a dirty tree exits 2. Otherwise it exits 2.
- The help replaces the sequential rule with: judge every provision; the first failure is the headline, and later fixes wait until that provision passes; every body is still judged. It shows `all`, `tally`, the receipt line, and one sheet example.
- The command stays stateless, only prints, never decides pass or fail, and never refuses a batch, an order, or a slice size.
- Provision headings, bodies, and section numbers do not change. The version does not change.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `gates`: bodies carry receipts, parents name the bodies not shown, `all` prints the whole statute, `tally` counts a sheet, and every provision is judged.

## Impact

- Specs: `openspec/specs/gates/spec.md`
- Code: `gates/run.go`, `gates/print.go`, `gates/tally.go`, `gates/gates_test.go`, `gates/tally_test.go`, `internal/scout/help.go`
- Docs: `README.md`, `AGENTS.md`, the gates sentence in `/home/ryan/AGENTS.md`
- Migration: a script that expected a bare body now also sees the receipt line. Receipts change with every build version.
