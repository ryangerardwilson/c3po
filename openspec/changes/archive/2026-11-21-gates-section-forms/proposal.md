## Why

A gates section must be typed with the section sign, `§6`. The sign is awkward to type, so `6` and the old `S.6` both fail as unknown sections. An agent that judges a checklist also needs several provisions and has to run one command per section.

## What Changes

- A section argument may be written three ways: `§6`, the bare `6`, or the old `S.6`. `8.3` and `S.8.3` name the same provision as `§8.3`. The published section and the provision list stay `§`, with no space: `§6  Checklist`.
- `6.0`, `01`, `0`, `S.0`, `s.1`, `§ 1`, and `§.1` are still unknown sections.
- Several sections may follow the statute. `c3po gates code architecture 6 7 8.3` prints `§6`, its body, a blank line, `§7`, its body, a blank line, `§8.3`, and its body, in the order written. The last body ends with one newline and no blank line.
- One section still prints only the body and one newline, with no label, whichever form it is written in.
- If any requested section is unknown, stdout is empty, the process exits 2, and stderr names that section as an unknown section. The valid sections are not printed.
- The help shows that a section may be `§1.1`, `1.1`, or `S.1.1` and that several sections may follow the statute. The report lines stay `§1  pass` and `§1.1  fail  not judged`.
- Headings, bodies, section order, and the version do not change.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `gates`: a section argument may be `§1.1`, `1.1`, or `S.1.1`, and several sections may follow the statute in one pass.

## Impact

- Specs: `openspec/specs/gates/spec.md`
- Code: `gates/statute.go`, `gates/run.go`, `gates/gates_test.go`, `internal/scout/help.go`
- Docs: `AGENTS.md`, `README.md`, and the gates sentence in `/home/ryan/AGENTS.md`
- Migration: none. `§6` keeps working, `6` and `S.6` now resolve, and a fourth argument is now another section instead of an error.
