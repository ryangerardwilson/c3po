## Why

`c3po gates list code architecture` printed ids like `C/A/S.1`, and `c3po gates get C/A/S.1` took that id back. An agent had to learn two subcommands and an id grammar to walk one tree. Each argument can instead deepen the same probe.

## What Changes

- **BREAKING** `list` and `get` are gone. `c3po gates list ...` and `c3po gates get ...` exit 2 as unknown commands.
- `c3po gates` prints the types, `code` then `essays`.
- `c3po gates <type>` prints that type's statutes. A type with none prints `no statutes`.
- `c3po gates <type> <statute>` prints every provision in section order as `<section>  <heading>`, for example `S.1  Data model`. No banner and no blank line.
- `c3po gates <type> <statute> <section>` prints only that body and one newline.
- The `C/A/S.1` id is gone. `C/A/S.1`, `S.9`, `S.1.0`, `S.01`, and `1` are unknown sections. A fourth argument exits 2.
- The report line in the help is `S.1  pass`, `S.1  fail`, and after a fail `S.1.1  fail  not judged`.
- Provision bodies, headings, and section numbers do not change.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `gates`: one probe per added argument replaces `list` and `get`; the provision id becomes the bare section.
- `scout`: the gates usage scenario runs `c3po gates code architecture S.1`, and the record does not store the type, statute, or section.

## Impact

- Specs: `openspec/specs/gates/spec.md`, `openspec/specs/scout/spec.md`
- Code: `gates/run.go`, `gates/statute.go`, `gates/gates_test.go`, `internal/scout/help.go`, `internal/usage/feature_test.go`
- Docs: `AGENTS.md`, `README.md`, and the gates sentence in `/home/ryan/AGENTS.md`
- Migration: `c3po gates list code architecture` becomes `c3po gates code architecture`, and `c3po gates get C/A/S.1` becomes `c3po gates code architecture S.1`. The source version does not change.
