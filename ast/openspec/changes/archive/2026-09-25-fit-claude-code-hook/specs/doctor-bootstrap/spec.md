## MODIFIED Requirements

### Requirement: Doctor and agents-block
`rgw-ast doctor` SHALL report version, config path, root measure, enforcement,
and evidence-based readiness flags. An AGENTS block SHALL be current only when
it matches the canonical managed block. Hook readiness SHALL require a parsed
command field whose argv begins with `rgw-ast hook`; prose mentions are not
readiness evidence. Doctor SHALL look for that command in the workspace host
settings (`.codex/hooks.json`, `.claude/settings.json`,
`.claude/settings.local.json`) and in the user-level host settings:
`settings.json` under `$CLAUDE_CONFIG_DIR` (default `~/.claude`) and
`hooks.json` under `$CODEX_HOME` (default `~/.codex`). `rgw-ast agents-block`
SHALL print the canonical managed AGENTS boundary without writing files.

#### Scenario: Agents block is pure output
- **WHEN** agents-block runs
- **THEN** stdout SHALL contain `<!-- rgw-ast:begin -->`
- **AND** no project-local config file SHALL be created

#### Scenario: Superficial agents block is stale
- **GIVEN** AGENTS.md has managed markers but omits canonical instructions
- **WHEN** doctor inspects the workspace
- **THEN** `agents_block_ready` SHALL be false
- **AND** `agents_block_status` SHALL be `stale`

#### Scenario: Prose hook mention is not configured
- **GIVEN** a supported host settings file mentions `rgw-ast hook` only in prose
- **WHEN** doctor inspects the workspace
- **THEN** `hook_ready` SHALL be false
- **AND** `hook_status` SHALL be `not_checked`

#### Scenario: User-level hook is configured
- **GIVEN** the user-level Claude Code `settings.json` has a PreToolUse command `rgw-ast hook`
- **AND** the workspace has no host settings file
- **WHEN** doctor inspects the workspace
- **THEN** `hook_ready` SHALL be true
- **AND** `hook_status` SHALL be `configured`

#### Scenario: Enforced checkout with a missing hook is degraded
- **GIVEN** enforcement is on
- **AND** the agents block is missing or the host hook command is not configured
- **WHEN** doctor inspects the workspace
- **THEN** `enforcement_status` SHALL be `degraded`
- **AND** `ready` SHALL be false
- **AND** `ok` SHALL be false
- **AND** `hook_ready` and `agents_block_ready` SHALL still be reported separately

#### Scenario: Unenforced checkout is not degraded
- **GIVEN** enforcement is off
- **AND** diagnostics complete without a hard error
- **WHEN** doctor inspects the workspace
- **THEN** `enforcement_status` SHALL be `inactive`
- **AND** `ok` SHALL be true
