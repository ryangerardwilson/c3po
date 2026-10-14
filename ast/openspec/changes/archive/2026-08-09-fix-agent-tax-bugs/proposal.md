## Why

Field report `rgw-ast-bugs.md` (home workspace, 2026-08-09) documents high
agent tax: home-root LOC inflation via nested repos, no create path, shell and
hidden-path blind spots, argv-only multiline patches, serial hash cycles,
weak read/search UX, and stale cache fingerprints.

## What Changes

- Scope: stop at nested Git roots; honor `.gitignore` when root is a Git repo;
  include shell/config globs; do not skip hidden dirs that are tracked or
  explicitly targeted; `explain <path>`; status scope diagnostics.
- Create/append guarded file ops; patch via files/stdin/JSON ops batch.
- Read: `--number`, soft `--cap` behavior with continuation; path headers.
- Search: path/glob flags; subcommand help; no treating flags as query.
- Cache: `--refresh`, hit/age in status JSON; recheck before mutation.
- Map: Bash function/alias heuristics.
- Defaults and AGENTS generator lane note.

## Capabilities

### New Capabilities

- `file-create`: create/append contracts

### Modified Capabilities

- `measure-enforcement`, `file-boundary`, `code-intelligence`, `cli-surface`,
  `global-config`

## Impact

- Go packages under Apps/rgw-ast
- Global config defaults (tracked home `.config/rgw-ast/config.toml` separately)
- Non-goals this change: full trusted-generator exec transaction, QML/TOML
  deep maps, background indexer, version stamp automation
