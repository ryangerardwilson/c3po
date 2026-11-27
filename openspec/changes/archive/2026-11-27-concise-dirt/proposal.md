## Why

`c3po dirt` prints each dirty repository's path and then its whole `git status --porcelain=v1 -b`, with a blank line between repositories. Across the home directory that list runs long. The operator only needs to know which path is dirty and how much dirt is in it.

## What Changes

- Stdout is one line per dirty repository, sorted by absolute path, with no blank lines and no file names: `<absolute-path>  <count>`. Two spaces separate the path and the count, and a newline follows every line.
- The count is the number of `git status --porcelain=v1 -b` lines that are not the `##` branch line, after the inside-path filter. A rename is one line. An untracked-directory line is one line. A repository whose count is 0 is omitted.
- A scan path inside a repository counts only the porcelain lines under the scan path. The repository is omitted when that count is 0.
- The scan, the walk, the status command, the failure handling, the usage errors, and version are unchanged. The help text describes the one-line list.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `dirt`: print one `<path>  <count>` line per dirty repository instead of the porcelain dump.

## Impact

- Specs: `openspec/specs/dirt/spec.md`
- Code: `dirt/`
- Docs: `README.md`, `AGENTS.md`, `openspec/config.yaml`
- Migration: an agent that read the porcelain lines from `c3po dirt` runs `git status` in the named repository instead. The source version stays 0.5.0.
