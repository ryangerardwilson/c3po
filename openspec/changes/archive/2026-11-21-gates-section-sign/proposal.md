## Why

A gates section was written `S.1`, a letter and a dot standing in for the section sign. Statutes number their provisions with `§`. The letter form also left case to decide whether `s.1` was a section.

## What Changes

- **BREAKING** A section is the section sign `§` (U+00A7) immediately followed by one or more positive integers separated by dots, with no space and no letter. `§1` replaces `S.1`, `§1.1` replaces `S.1.1`, `§3.5.1` replaces `S.3.5.1`, and `§10` replaces `S.10`.
- The parser accepts only that form. `S.1`, `s.1`, `§ 1`, `§.1`, `1`, `§0`, `§1.0`, and `§01` are not sections and are unknown sections of `code architecture`.
- `c3po gates code architecture` prints `§1  Data model` then `§1.1  Domain truth`. `c3po gates code architecture §1.1` prints that body and one newline.
- The help and the report lines use `§1  pass`, `§1  fail`, and after a fail `§1.1  fail  not judged`.
- Section order, headings, and bodies do not change.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `gates`: a section is `§` and then dot-separated positive integers.
- `scout`: the gates usage scenario runs `c3po gates code architecture §1`.

## Impact

- Specs: `openspec/specs/gates/spec.md`, `openspec/specs/scout/spec.md`
- Code: `gates/statute.go`, `gates/architecture.go`, `gates/run.go`, `gates/gates_test.go`, `internal/usage/feature_test.go`
- Docs: `AGENTS.md`, `README.md`, and the gates sentence in `/home/ryan/AGENTS.md`
- Migration: `c3po gates code architecture S.1.1` becomes `c3po gates code architecture §1.1`. `S.1.1` now exits 2 as an unknown section. The source version does not change.
