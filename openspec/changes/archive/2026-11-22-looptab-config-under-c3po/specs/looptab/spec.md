## MODIFIED Requirements

### Requirement: Jobs files are registered and installed from the CLI
`c3po looptab` SHALL be a command line and SHALL NOT open vim or any other terminal screen. With no arguments it SHALL print the help, SHALL exit 0, and SHALL NOT create or register a file. `edit` SHALL NOT be a command; `c3po looptab edit` SHALL print the help and SHALL exit 2. `c3po looptab register <path>` SHALL record an existing regular file and install it. The registered name SHALL be the lowercase basename without its extension. A duplicate name SHALL exit 2. The looptab config directory SHALL be `<config>/c3po/looptab`, where `<config>` is `$XDG_CONFIG_HOME` when it is an absolute path and `~/.config` otherwise. The scheduler file SHALL be `<config>/c3po/looptab/looptab`, the Looptab timezone file SHALL be `<config>/c3po/looptab/config.json`, and jobs files SHALL sit beside them. The scheduler file SHALL be installed output, and registering that path, or a path that opens the same file, SHALL exit 2. `c3po looptab remove <name>` SHALL drop that registration and install the rest. An unknown name SHALL exit 2. `c3po looptab list` SHALL print each name and path. An empty set SHALL print `no jobs files`. `c3po looptab show <name>` SHALL print the name, the path, and the file. `c3po looptab help` SHALL exit 0, SHALL say the jobs file is edited with `r2d2 looptab`, and SHALL NOT name vim. `c3po looptab version` SHALL print the c3po version. Extra arguments SHALL print the help and SHALL exit 2. Registrations SHALL be stored in the sqlx database `c3po`. An empty registration set SHALL write a comment-only scheduler file and SHALL keep that file. c3po SHALL be the only writer of the scheduler file, and register, remove, and the install after an edit SHALL write it through one install function. c3po SHALL export `JobsFile`, which returns the first registered jobs file, and with none registered creates `<config>/c3po/looptab/jobs` from a comment template, registers it as `jobs`, and installs it. c3po SHALL export `Install`, which installs every registered file and returns how many it installed.

#### Scenario: No arguments print the help
- **WHEN** the operator runs `c3po looptab` with nothing registered
- **THEN** the command SHALL exit 0 and SHALL print the help
- **AND** no jobs file SHALL be created and no scheduler file SHALL be written

#### Scenario: Edit is not a command
- **WHEN** the operator runs `c3po looptab edit`
- **THEN** the command SHALL exit 2
- **AND** the text SHALL include the help

#### Scenario: Help names the screen
- **WHEN** the operator runs `c3po looptab help`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL say the jobs file is edited with `r2d2 looptab`
- **AND** the text SHALL NOT name vim

#### Scenario: A jobs file is registered and removed
- **WHEN** the operator registers a regular file and then removes its name
- **THEN** list shows the name
- **AND** the scheduler file contains the file body
- **AND** remove leaves `no jobs files`
- **AND** a second remove exits 2

#### Scenario: An empty set gets the template
- **WHEN** nothing is registered and the screen asks c3po for the jobs file
- **THEN** `~/.config/c3po/looptab/jobs` SHALL hold the comment template
- **AND** it SHALL be registered as `jobs` and installed

#### Scenario: The scheduler file is under the c3po config directory
- **WHEN** `XDG_CONFIG_HOME` is an absolute path and c3po installs the registered files
- **THEN** c3po SHALL write `$XDG_CONFIG_HOME/c3po/looptab/looptab`
- **AND** with `XDG_CONFIG_HOME` empty or relative it SHALL write `~/.config/c3po/looptab/looptab`

#### Scenario: A link to the scheduler file is not a source
- **WHEN** the operator registers a symlink that opens the scheduler file
- **THEN** the command SHALL exit 2
- **AND** nothing SHALL be registered
