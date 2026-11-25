## Why

The `code architecture` statute covers the data model, state ownership, and boundaries, but says nothing about user interfaces. An agent building a screen has no provision to judge components, view states, feedback, accessibility, or layout against, so UI work goes unjudged.

## What Changes

- Add `§9 Components and UX` to `code architecture` with 25 leaves, `§9.1` through `§9.25`, that are the UI rules UI-1 through UI-25. `§9.1` through `§9.13` are MUST rules and `§9.14` through `§9.25` are SHOULD rules, matching `§5`.
- `§1` through `§8` keep their sections, headings, and bodies byte for byte.
- The statute grows from 93 to 119 provisions, so `all` prints 119 blocks and a complete tally sheet reports `119 of 119 judged`.
- `§10` and `10` replace `§9` and `9` as the example unknown sections, because `§9` is now published.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `gates`: the architecture statute publishes `§9`, and the counts in the probe and tally scenarios become 119.

## Impact

- `gates/architecture.go`, `gates/gates_test.go`, `gates/tally_test.go`, and `openspec/specs/gates/spec.md`.
- A tally sheet written for 93 provisions now reports the 26 `§9` sections as missing until they are judged. Receipts of `§1` through `§8` do not change, since a receipt hashes only the version, the section, and the body.
- The version stays 0.5.0.
