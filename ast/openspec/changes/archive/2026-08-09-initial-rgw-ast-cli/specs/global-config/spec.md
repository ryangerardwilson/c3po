## ADDED Requirements

### Requirement: Sole global policy file
`rgw-ast` SHALL load policy only from the global config file at
`$XDG_CONFIG_HOME/rgw-ast/config.toml` when `XDG_CONFIG_HOME` is set, otherwise
`$HOME/.config/rgw-ast/config.toml`. The tool SHALL NOT require or read a
project-local `.rgw-ast.toml` or equivalent policy file inside the workspace.

#### Scenario: No project config required
- **GIVEN** a git repository with no project-local rgw-ast policy file
- **WHEN** the user runs `rgw-ast status` from that repository
- **THEN** the tool SHALL use only the global config path
- **AND** the command SHALL succeed without creating a project-local config

### Requirement: Ensure defaults on first use
When the global config file is missing, `rgw-ast` SHALL create its parent
directory and write a default `config.toml` before proceeding.

Default values SHALL include at least:
- `threshold_loc = 5000`
- `enforcement.mode = "auto"`
- include globs covering common source extensions
- exclude globs for `node_modules`, `.git`, `dist`, `build`, `.next`, `vendor`,
  `target`

#### Scenario: First run creates config
- **GIVEN** the global config file does not exist
- **WHEN** any command that needs config runs
- **THEN** the default config file SHALL be created at the global path
- **AND** subsequent loads SHALL read that file

### Requirement: Config command
`rgw-ast config` SHALL print the resolved absolute path of the global config
file to stdout and exit 0. It SHALL ensure the file exists (writing defaults if
missing).

#### Scenario: Config path
- **WHEN** the user runs `rgw-ast config`
- **THEN** stdout SHALL contain the absolute path to `config.toml`
