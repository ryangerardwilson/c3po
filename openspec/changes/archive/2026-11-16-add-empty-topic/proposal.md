## Why

A topic cannot exist until it has an idea. Naming the topic first, before any idea is ready, exits 2.

## What Changes

- **BREAKING** `c3po idealake add <topic>` with no ideas stores that topic and exits 0. A terminal with no idea arguments used to exit 2. Empty stdin with no idea arguments used to exit 2.
- `c3po idealake topics` lists that topic with a count of 0.
- A later `add` of ideas under the same topic stores those ideas. Adding the same empty topic again does not duplicate it.
- A rejected idea still stores nothing, including the topic when that call created none.
- `--json add` of a topic with no ideas prints a `topics` array. `--json add` of ideas still prints an `ideas` array and no embedding vector.

## Capabilities

### New Capabilities

### Modified Capabilities

- `idealake`: A topic can be stored with no ideas, and topics lists it with a count of 0.

## Impact

- Specs: `openspec/specs/idealake/spec.md`
- Code: `idealake/`
