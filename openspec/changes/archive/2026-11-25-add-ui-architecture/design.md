## Context

`gates/architecture.go` holds the statute as one ordered slice of provisions. Sections are numbered without gaps, a parent prints its immediate subsections as title lines, and a body is at most 300 code points. Tests and the gates spec assert the provision count in several places.

## Goals / Non-Goals

**Goals:**
- Publish the UI rules as `§9` and its 25 leaves, independent of language, framework, and platform.
- Keep `§1` through `§8` and their receipts unchanged.

**Non-Goals:**
- No change to parsing, probing, tally, receipts, or the version.
- No new statute or type.

## Decisions

- **A new top section, not a new statute.** The UI rules extend the same architecture judgment (ownership, one source of truth, boundaries) to components, and `code` holds only `architecture`. A separate statute would add a type-level change for one subject.
- **Flat leaves, one rule per leaf.** UI-n is `§9.n`, so a rule's label and its section number stay aligned and a tally line names one rule.
- **MUST then SHOULD.** `§9.1`–`§9.13` are hard rules and `§9.14`–`§9.25` are defaults, the order `§1`–`§3` already use and `§5` defines.
- **`§10` as the unknown probe.** `§9` now exists, so the tests and spec move the unknown-section example to the next unpublished top section.

## Risks / Trade-offs

- Sheets written before this change fail tally until `§9` is judged → expected; tally already names the missing sections and the fetch command.
