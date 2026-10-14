## ADDED Requirements

### Requirement: Pre-tool-use hook
`rgw-ast hook` SHALL read a single JSON object from stdin describing a
proposed agent tool call and write a JSON decision to stdout.

Recognized input fields SHALL include at least:
- `tool_name` (string)
- `tool_input` (object, optional)
- `command` (string, optional; shell command text)

When the workspace (cwd or `--root`) is enforced, the hook SHALL deny:
- direct editor/mutation tool names in a built-in deny list (for example
  `Write`, `Edit`, `apply_patch`, `str_replace`, `search_replace`)
- shell commands that mutate files outside `rgw-ast` (detected via simple
  argv patterns such as `sed -i`, `rm `, redirection write patterns)

The hook SHALL allow:
- tools whose name or command clearly invokes `rgw-ast`
- read-only shell inspection when not matching mutation patterns
- all tools when enforcement is inactive

Decision JSON SHALL include `permissionDecision` of `allow` or `deny` and a
`permissionDecisionReason` string when denied. Exit code SHALL be 0 even when
denying (hosts expect successful hook delivery).

#### Scenario: Deny Write when enforced
- **GIVEN** enforcement is active for the workspace
- **WHEN** stdin is `{"tool_name":"Write","tool_input":{"path":"x.go"}}`
- **THEN** stdout JSON SHALL have `permissionDecision` equal to `deny`
- **AND** the process SHALL exit 0

#### Scenario: Allow rgw-ast shell when enforced
- **GIVEN** enforcement is active
- **WHEN** stdin includes a shell command starting with `rgw-ast `
- **THEN** stdout JSON SHALL have `permissionDecision` equal to `allow`
