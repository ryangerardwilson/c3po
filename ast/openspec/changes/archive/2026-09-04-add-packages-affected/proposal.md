## Why

Agents need a generated package index and an affected-verify set. `map` dumps
symbols, not Apps/Tests packages.

## What Changes

- Add `rgw-ast packages` and `rgw-ast affected` (optional `--json`)
- Discover Apps/*, Tests/*, `.config/hypr`, `.config/quickshell`
- Map Git dirty/untracked or explicit paths onto package verify commands

## Capabilities

### Modified Capabilities
- `cli-surface`: packages and affected verbs
