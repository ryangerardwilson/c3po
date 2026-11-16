## Why

An agent can land a function at or above max_cc, and the docs tell some agents to write through a path the hook does not recognize. The cyclomatic gate covers Go patch, create, and apply only. TypeScript and Python are not refused.

## What Changes

- Mark these as open. The 2026-11-16 probe has run. This document does not implement them.
- Recognize MCP tool names in the PreToolUse hook. `normalizeTool` strips `_` before it looks for `__`, so `mcp__ast-mcp__edit_file` does not match a denied tool.
- Run the max_cc gate on `c3po ast append` and on a new Go, TypeScript, and Python file. The probe landed all three through a host Write of a path that did not exist yet. That write skipped the gate. The agent deleted only the Go file after `c3po ast gap`.
- Do not accept a Go write that does not parse when that skips the gate.
- Enforce max_cc on Go, TypeScript, and Python. A function that reaches max_cc or above is refused. A function already at or above the line may still be edited, matching the Go rule.
- Remove `push_release_upgrade.sh` from `generators.allow`.
- Leave the American Dream Enterprises writer conflict, the stale `c3po tui` lines, and the copied agents blocks as doc follow-ups. They are not part of the gate.

## Capabilities

### New Capabilities

### Modified Capabilities

- `complexity-gap`: The write gate refuses a function at or above max_cc in Go, TypeScript, and Python.
- `agent-hook`: MCP tool names are judged as editor writes.

## Impact

- Specs: `openspec/specs/ast` if that spec owns the gate, otherwise the ast spec that names max_cc
- Code: `ast/internal/hook`, `ast/internal/cli`, the TypeScript and Python scorers
- Not in this change: editing product code, installing, or changing the live config
