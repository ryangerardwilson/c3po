## Why

`c3po dirt` prints a space before the colon. The path should run straight into the colon.

## What Changes

- The line is `<path>: <tally>`. The colon follows the path with no space. One space follows the colon. An example is `~/src/app: M2 +3 -7 ??5`.
- The path form and the tally marks stay the same.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `dirt`: print `<path>: <tally>` instead of `<path> : <tally>`.

## Impact

- Specs: `openspec/specs/dirt/spec.md`
- Code: `dirt/`
- Docs: `README.md`, `AGENTS.md`, `openspec/config.yaml`
- Migration: an agent that split on ` : ` splits on `: ` instead. The source version stays 0.5.0.
