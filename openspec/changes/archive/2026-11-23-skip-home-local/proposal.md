## Why

`c3po dirt` on the home directory walks `~/.local`, so trash, editor plugins, and language-server checkouts show up as repositories. Those trees are not work the agent is doing.

## What Changes

- A walk whose scan root is outside `~/.local` does not enter that directory.
- `c3po dirt ~/.local` still scans it. A scan root with that path still scans it.
- A directory named `.local` anywhere else is still entered.
- Help, the README, and the dirt spec say so.

## Capabilities

### New Capabilities

### Modified Capabilities

- `dirt`: the walk skips the home directory's `.local`.

## Impact

- Specs: `openspec/specs/dirt/spec.md`
- Code: `dirt/scan.go`, `dirt/run.go`, `dirt/dirt_test.go`
- Docs: `README.md`, `openspec/config.yaml`
- Migration: none. The command only reads. The source version stays 0.5.0.
