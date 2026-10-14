## MODIFIED Requirements

### Requirement: Ensure defaults on first use
When the global config file is missing, `rgw-ast` SHALL create its parent
directory and write a default `config.toml` before proceeding.

Default values SHALL include at least:
- `threshold_loc = 5000`
- `enforcement.mode = "auto"`
- include globs covering common application languages, Markdown, shell
  (`.sh`, `.bash`, `.bashrc`, `.profile`), and common config extensions
  (`.toml`, `.yml`, `.yaml`, `.json`)
- exclude globs for `node_modules`, `.git`, `dist`, `build`, `.next`, `vendor`,
  `target`

#### Scenario: First run creates config
- **GIVEN** the global config file does not exist
- **WHEN** any command that needs config runs
- **THEN** the default config file SHALL be created at the global path
- **AND** subsequent loads SHALL read that file
