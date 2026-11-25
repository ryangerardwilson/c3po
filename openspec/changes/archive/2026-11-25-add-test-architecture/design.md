## Context

`gates/architecture.go` holds the statute as one ordered slice of provisions. Sections are numbered without gaps, a parent prints its immediate subsections as title lines, and a body is at most 300 code points. Tests and the gates spec assert the provision count in several places.

## Goals / Non-Goals

**Goals:**
- Publish the test rules as `§10` and its 20 leaves, independent of language and test framework.
- Keep `§1` through `§9` and their receipts unchanged.

**Non-Goals:**
- No change to parsing, probing, tally, receipts, or the version.
- No new statute or type.
- No rewrite of the checklist in `§6`.

## Decisions

- **A new top section, not a new statute.** The test rules extend the same architecture judgment, and `code` holds only `architecture`. A separate statute would add a type-level change for one subject.
- **Flat leaves, one rule per leaf.** TS-n is `§10.n`, so a rule's label and its section number stay aligned and a tally line names one rule.
- **MUST then SHOULD.** `§10.1`–`§10.11` are hard rules and `§10.12`–`§10.20` are defaults, the order `§1`–`§3` and `§9` already use and `§5` defines. Each of those bodies contains the word MUST or SHOULD.
- **`§11` as the unknown probe.** `§10` now exists, so the tests and spec move the unknown-section example to the next unpublished top section. The sort fixture that lists a fictional `§10` among `§1`, `§2.3`, `§3.5.1` is a parser example and stays.

## Risks / Trade-offs

- Sheets written before this change fail tally until `§10` is judged → expected; tally already names the missing sections and the fetch command.
