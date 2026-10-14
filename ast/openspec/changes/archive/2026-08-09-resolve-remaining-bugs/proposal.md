## Why

Re-evaluation of `rgw-ast-bugs.md` on v0.1.0 left: trusted-generator gap,
cache staleness for nested working-tree edits, incomplete config-language
maps, missing doctor/agents-block, and README drift.

## What Changes

- Git dirty-tree fingerprint for measure cache; force refresh on mutation/hook
- `rgw-ast exec` trusted-generator transaction with before/after file report
- Hook allowlist for generators only when invoked via `rgw-ast exec` (or
  allow documented generator prefixes when wrapped)
- `doctor` and `agents-block`
- Map Markdown headings, JSON/TOML/YAML keys, light QML
- README command surface update

## Capabilities

### New Capabilities
- `exec-generator`, `doctor-bootstrap`

### Modified Capabilities
- `measure-enforcement`, `code-intelligence`, `cli-surface`, `agent-hook`

## Impact
Go packages; global config optional generators.allow; release 0.1.1
