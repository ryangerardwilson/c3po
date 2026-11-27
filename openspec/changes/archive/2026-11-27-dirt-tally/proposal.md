## Why

`c3po dirt` prints one line per dirty repository, but that line is an absolute path and a single count. The operator wants the path under home, and a short tally of what kind of dirt it is.

## What Changes

- Stdout stays one line per dirty repository, sorted by absolute path, with no blank lines and no file names. The line is `<path> : <tally>`. One space stands on each side of the colon. A newline follows every line.
- A path equal to the home directory is `~`. A path under that directory is `~/` plus the slash-separated relative path. Any other path stays absolute. An example is `~/src/app : M2 +3 -7 ??5`.
- The tally lists only the marks whose count is greater than zero, in the order `M`, `+`, `-`, `??`. Each mark is followed immediately by its decimal count. `M` is modified, `+` is added, `-` is deleted, and `??` is untracked. A rename counts as one modified. An untracked directory counts as one untracked. An ignored line counts as nothing. A repository whose tally is empty is omitted.
- Each porcelain line other than the `##` branch line adds one mark. `?` in either column is untracked, `D` in either column is deleted, `A` or `C` in either column is added, and every other line is modified.
- The scan, the walk, the status command, the inside-path filter, the failure handling, and the usage errors are unchanged.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `dirt`: print `<path> : <tally>` instead of `<absolute-path>  <count>`.

## Impact

- Specs: `openspec/specs/dirt/spec.md`
- Code: `dirt/`
- Docs: `README.md`, `AGENTS.md`, `openspec/config.yaml`
- Migration: an agent that split the old count line reads the marks after ` : `. The source version stays 0.5.0.
