## Why

The `code architecture` statute covers the data model, state ownership, boundaries, and user interfaces, but says nothing about tests. An agent writing or changing a test has no provision to judge it against, so a test can assert private state, depend on another test, or pass by weakening an assertion.

## What Changes

- Add `§10 Tests` to `code architecture` with 20 leaves, `§10.1` through `§10.20`, that are the test rules TS-1 through TS-20. `§10.1` through `§10.11` are MUST rules and `§10.12` through `§10.20` are SHOULD rules, matching `§5`.
- `§1` through `§9` keep their sections, headings, and bodies byte for byte.
- The statute grows from 119 to 140 provisions, so `all` prints 140 blocks and a complete tally sheet reports `140 of 140 judged`.
- `§11` and `11` replace `§10` and `10` as the example unknown sections, because `§10` is now published. The section-order example that uses a fictional `§10` stays, because it is a parser fixture and not a claim that `§10` is unpublished.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `gates`: the architecture statute publishes `§10`, and the counts in the probe and tally scenarios become 140.

## Impact

- `gates/architecture.go`, `gates/gates_test.go`, `gates/tally_test.go`, and `openspec/specs/gates/spec.md`.
- A tally sheet written for 119 provisions now reports the 21 `§10` sections as missing until they are judged. Receipts of `§1` through `§9` do not change, since a receipt hashes only the version, the section, and the body.
- The version stays 0.5.0.
