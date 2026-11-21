## Why

`c3po gates code architecture 8` prints only `A rule may be broken deliberately, never silently.` The agent cannot see that `§8.1` through `§8.4` exist without stepping back to the full provision list. A section with subsections can name them under its body, so the agent can go one level deeper from where it stands.

## What Changes

- A section with subsections prints its body, a blank line, then one line per immediate subsection in section order, as the section, two spaces, and the heading. No banner and no subsection body.
- An immediate subsection is the next number only. `§8.1` is a subsection of `§8`. `§8.1.1` is a subsection of `§8.1`, not of `§8`.
- A section with no subsections still prints only its body and one newline.
- Several sections keep their order. Each block is the canonical section, its body, then its subsection lines when it has any. A blank line separates blocks.
- `8`, `§8`, and `S.8` still name the same section. One unknown section still prints nothing and exits 2.
- The help says a section with subsections lists those subsections under the body.
- Provision headings, bodies, and section numbers do not change.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `gates`: a section with subsections lists its immediate subsections under the body.

## Impact

- Specs: `openspec/specs/gates/spec.md`
- Code: `gates/run.go`, `gates/statute.go`, `gates/gates_test.go`
- Docs: `README.md`, `AGENTS.md`, the gates sentence in `/home/ryan/AGENTS.md`
