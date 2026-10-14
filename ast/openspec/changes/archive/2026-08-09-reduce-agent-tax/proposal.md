## Why

First-use evaluation showed high agent tax: full LOC remeasure on every
guarded command (~5s on large trees), whole-file read deny only for
include-matched paths (JSON dumps still allowed), fat search lines, and no
host-side hook surface to force agents onto `rgw-ast`.

## What Changes

- Cache LOC measurements under XDG cache with invalidation fingerprint.
- Reuse cached measure for status/read/patch enforcement decisions.
- When enforced, deny whole-file reads for any non-binary text under root.
- Truncate search hit snippets.
- Support global `--root` to scope workspace measurement.
- Add `rgw-ast hook` stdin JSON pre-tool-use guard for agent hosts.

## Capabilities

### New Capabilities

- `agent-hook`: pre-tool-use hook contract

### Modified Capabilities

- `measure-enforcement`: cache + --root
- `file-boundary`: deny all text whole-file reads when enforced
- `code-intelligence`: truncated search snippets
- `cli-surface`: hook command, --root flag

## Impact

- Go packages: measure, files, intel, cli, new hook package
- Cache files under ~/.cache/rgw-ast/
- Non-goals: full MCP server, multi-language AST rewrites
