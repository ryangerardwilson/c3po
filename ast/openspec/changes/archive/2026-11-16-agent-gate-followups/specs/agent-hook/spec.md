## ADDED Requirements

### Requirement: MCP editor tools are denied in an enforced tree
The hook SHALL treat an MCP tool name as the editor tool it wraps. A name shaped `mcp__<server>__<tool>` SHALL be judged by `<tool>` after the server prefix is removed. `mcp__ast-mcp__edit_file`, `mcp__ast-mcp__file_write`, and `mcp__ast-mcp__file_patch` SHALL be denied on an existing file in an enforced tree, and the denial SHALL name `c3po ast patch`.

#### Scenario: An MCP edit is denied
- **WHEN** the hook receives tool name `mcp__ast-mcp__edit_file` for an existing file in an enforced repo
- **THEN** the decision SHALL be deny
- **AND** the reason SHALL name `c3po ast patch`

### Requirement: A new scored file is gated on host Write
When the hook receives a host Write of a path that does not exist, the path ends in `.go`, `.ts`, `.tsx`, or `.py`, and the tool input includes the file body, the hook SHALL score that body. In an enforced workspace, a function at max_cc or above SHALL be denied and the reason SHALL name the function and its score. Unparsed Go SHALL be denied and the reason SHALL say unparsed. The file SHALL NOT be written. A body under max_cc SHALL be allowed.

#### Scenario: A new Go file over the line is denied
- **WHEN** the hook receives a Write of a missing `.go` file in an enforced repo whose function is at max_cc or above
- **THEN** the decision SHALL be deny
- **AND** the reason SHALL name that function and its score

#### Scenario: A new Python file over the line is denied
- **WHEN** the hook receives a Write of a missing `.py` file in an enforced repo whose function is at max_cc or above
- **THEN** the decision SHALL be deny
- **AND** the file SHALL NOT be written

#### Scenario: A new TypeScript file over the line is denied
- **WHEN** the hook receives a Write of a missing `.ts` file in an enforced repo whose function is at max_cc or above
- **THEN** the decision SHALL be deny
