# notify Specification

## Purpose
TBD - created by archiving change add-notify. Update Purpose after archive.
## Requirements
### Requirement: Notify sends an omarchy-shell toast
`c3po notify heading "<title>" body "<message>"` SHALL show one toast with that title and body. `c3po notify --urgency <level> heading "<title>" body "<message>"` SHALL show it at that urgency, and `-u` SHALL be the short flag. The urgency SHALL be `low`, `normal`, or `critical`, matched without case. Any other urgency SHALL become `normal`, and with no flag the urgency SHALL be `normal`. The legacy positional form `c3po notify <title> [body...]` SHALL show the first word as the title and the remaining words joined by one space as the body.

Delivery SHALL first run `omarchy-shell notify show <title> <body> <urgency>`. When `OMARCHY_PATH` is unset or blank, that process SHALL receive `OMARCHY_PATH=/usr/share/omarchy`. When omarchy-shell is missing or fails, delivery SHALL run `quickshell ipc -p <bar> call bar notify <title> <body> <urgency>`. The bar SHALL be `$NOTIFY_QUICKSHELL_BAR` when it is set and not blank, and otherwise `$HOME/.config/quickshell/omarchy-bar`. On success the command SHALL print nothing and SHALL exit 0. When both fail, the command SHALL exit 1 and stderr SHALL name `omarchy-shell toast unavailable` and the omarchy-shell error.

#### Scenario: A labeled toast is silent
- **WHEN** the operator runs `c3po notify heading "gdrive" body "backup finished"` and omarchy-shell succeeds
- **THEN** omarchy-shell SHALL receive `notify show gdrive "backup finished" normal`
- **AND** stdout and stderr SHALL be empty
- **AND** the process SHALL exit 0

#### Scenario: Critical urgency
- **WHEN** the operator runs `c3po notify -u CRITICAL heading "gdrive" body "backup failed"`
- **THEN** the urgency SHALL be `critical`

#### Scenario: An unknown urgency is normal
- **WHEN** the operator runs `c3po notify --urgency loud heading "x"`
- **THEN** the urgency SHALL be `normal`

#### Scenario: The positional form
- **WHEN** the operator runs `c3po notify gdrive backup finished`
- **THEN** the title SHALL be `gdrive`
- **AND** the body SHALL be `backup finished`

#### Scenario: omarchy-shell gets a default OMARCHY_PATH
- **GIVEN** `OMARCHY_PATH` is unset
- **WHEN** the toast is sent through omarchy-shell
- **THEN** omarchy-shell SHALL see `OMARCHY_PATH=/usr/share/omarchy`

#### Scenario: The quickshell bar is the fallback
- **GIVEN** omarchy-shell exits non-zero and `NOTIFY_QUICKSHELL_BAR` is `/tmp/bar`
- **WHEN** the operator runs `c3po notify heading "gmail" body "done"`
- **THEN** quickshell SHALL receive `ipc -p /tmp/bar call bar notify gmail done normal`
- **AND** the process SHALL exit 0

#### Scenario: Both deliveries fail
- **GIVEN** omarchy-shell and quickshell both fail
- **WHEN** the operator runs `c3po notify heading "gmail" body "done"`
- **THEN** the process SHALL exit 1
- **AND** stderr SHALL contain `omarchy-shell toast unavailable` and the omarchy-shell output

### Requirement: Notify refuses a bad message
An empty or blank heading SHALL be refused. An empty or blank positional title SHALL be refused. A labeled form that has `body` and no `heading` SHALL be refused. A word after a complete labeled form SHALL be refused. `heading` or `body` with no value SHALL be refused. No arguments, `--urgency` with no value, and an unknown flag SHALL be refused. A refusal SHALL NOT send a toast, SHALL print the reason and the usage on stderr, and SHALL exit 2.

#### Scenario: Body without heading
- **WHEN** the operator runs `c3po notify body "backup finished"`
- **THEN** the process SHALL exit 2
- **AND** no toast SHALL be sent
- **AND** stderr SHALL name `usage: c3po notify`

#### Scenario: Empty heading
- **WHEN** the operator runs `c3po notify heading ""`
- **THEN** the process SHALL exit 2
- **AND** no toast SHALL be sent

#### Scenario: A word after the labeled form
- **WHEN** the operator runs `c3po notify heading "x" body "y" z`
- **THEN** the process SHALL exit 2
- **AND** no toast SHALL be sent

#### Scenario: No message
- **WHEN** the operator runs `c3po notify` or `c3po notify --urgency`
- **THEN** the process SHALL exit 2

### Requirement: Notify help, version, and plugin
`c3po notify help`, `c3po notify --help`, and `c3po notify -h` SHALL print help that names `c3po notify heading` and SHALL exit 0 without sending a toast. `c3po notify version` SHALL print `c3po <version>` and SHALL exit 0. There SHALL be no notify screen. The overlay plugin source SHALL be `notify/plugin/` with id `ryan.notify`. `install.sh` SHALL copy `manifest.json` and `Overlay.qml` to `${XDG_CONFIG_HOME:-$HOME/.config}/omarchy/plugins/ryan.notify/`. `install.sh` SHALL remove `~/.local/bin/notify` when it is a regular file or a symlink, and SHALL NOT install a `notify` link. The gmail and gdrive timer units SHALL toast through `c3po notify heading '<title>' body '<message>'` from the same c3po binary, with `--urgency critical` for the failure toast, and SHALL NOT define a shell `notify` function or call `notify-send`.

#### Scenario: Help
- **WHEN** the operator runs `c3po notify help`
- **THEN** the process SHALL exit 0
- **AND** the text SHALL name `c3po notify heading`
- **AND** no toast SHALL be sent

#### Scenario: Version
- **WHEN** the operator runs `c3po notify version`
- **THEN** stdout SHALL start with `c3po `

#### Scenario: Install replaces the standalone binary
- **GIVEN** `~/.local/bin/notify` is a regular file or a symlink
- **WHEN** the operator runs `install.sh`
- **THEN** `~/.local/bin/notify` SHALL be gone
- **AND** `~/.config/omarchy/plugins/ryan.notify/manifest.json` SHALL have id `ryan.notify`

#### Scenario: The gdrive timer calls c3po notify
- **WHEN** the gdrive timer unit is written
- **THEN** its script SHALL contain `notify --urgency critical heading 'gdrive' body 'Hourly backup failed'` after the c3po binary
- **AND** it SHALL NOT contain `notify-send`

