## Why

`c3po gates list code` printed one line for the whole architecture essay, and `c3po gates show code 1` printed the essay in one block. An agent judging work needs the statute, then one short rule at a time, each with an id it can report against.

## What Changes

- **BREAKING** `show` is gone. `c3po gates show ...` exits 2.
- `c3po gates list <type>` prints the statutes of that type. `code` has `architecture`. `essays` prints `no statutes`.
- `c3po gates list <type> <statute>` prints every provision, one `<id>  <heading>` line each, in section order, with no banner and no blank line.
- `c3po gates get <id>` prints only that provision's body. A body is at most 300 Unicode code points.
- An id is `<type letter>/<statute letter>/<section>`, for example `C/A/S.1`. A section is `S` and then dot-separated positive integers. Ids are case-sensitive.
- The embedded essay `gates/statute/code/1.md` is replaced by a table of 93 provisions cut from its text: the three disciplines, then how to apply, severity, checklist, reporting, and exceptions. The severity paragraph is split into two provisions.
- The sequential rule moves to the help only. The report line is `<id>  pass`, `<id>  fail`, or `<id>  fail  not judged`.
- `edit`, `add`, and `remove` exit 2 and say the text changes only when c3po is changed and reinstalled.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `gates`: statutes and provisions replace the single essay gate; `list` gains a statute level and `get` replaces `show`.
- `scout`: the gates usage scenario runs `get`, and the usage record does not store the type, statute, or id.

## Impact

- Specs: `openspec/specs/gates/spec.md`, `openspec/specs/scout/spec.md`
- Code: `gates/`, `internal/scout/help.go`, `internal/usage/feature_test.go`
- Docs: `AGENTS.md`, `README.md`
- Migration: an agent that ran `c3po gates show code 1` runs `c3po gates list code architecture` and then `c3po gates get <id>`. The source version does not change.
