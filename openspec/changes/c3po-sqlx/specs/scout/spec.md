## MODIFIED Requirements

### Requirement: Config lives under c3po
Each app's config SHALL be `$XDG_CONFIG_HOME/c3po/<app>` when `XDG_CONFIG_HOME` is set, and `~/.config/c3po/<app>` otherwise. A data directory MAY remain at `$XDG_DATA_HOME/c3po/<app>` or `~/.local/share/c3po/<app>`. Cache SHALL be `$XDG_CACHE_HOME/c3po/<app>` or `~/.cache/c3po/<app>`. Durable records SHALL live in the sqlx database `c3po`: OAuth tokens for gmail, gcal, and gdrive, gdrive sync state, and tunnel routes. Operator-edited config, credential files, signatures, the pass service-account token, generated tunnel credentials, and caches SHALL stay files. The pass token SHALL be the file `service-account-token` in the pass config directory. The directory `~/.config/op` SHALL stay in place. `GMAIL_CLI_CONFIG`, `GDRIVE_CONFIG`, `OP_SERVICE_ACCOUNT_TOKEN`, and `OP_SERVICE_ACCOUNT_TOKEN_FILE` SHALL still override the file they name.

#### Scenario: A legacy directory is adopted
- **GIVEN** config exists at the legacy path and the c3po path is absent
- **WHEN** c3po runs
- **THEN** the legacy path SHALL be renamed under `c3po/<app>`
- **AND** an existing c3po path SHALL be left as it is

#### Scenario: Durable records use one database
- **WHEN** c3po saves an OAuth token, Drive sync state, or a tunnel route
- **THEN** the record SHALL be stored in the sqlx database `c3po`
- **AND** a config file, credential file, signature, or cache SHALL stay a file
