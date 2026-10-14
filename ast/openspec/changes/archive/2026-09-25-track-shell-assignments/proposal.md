## Why

Agents often name a scratch directory once (`S=/tmp/...`) and reuse `$S`.
The hook judged every `$S` target as the current directory, so writes to
scratch space from an enforced repository were denied.

## What Changes

- Static `NAME=value` assignments in the command text define `$NAME` for the
  shell write analysis. Dynamic values stay unknown.

## Capabilities

### Modified Capabilities
- `agent-hook`: assignment-aware shell write targets

## Non-goals

- Tracking values from the parent environment, `read`, or loops
- Parameter expansion operators such as `${NAME:-default}`
