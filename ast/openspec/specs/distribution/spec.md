# distribution Specification

## Purpose
Install c3po from the public repository https://github.com/ryangerardwilson/c3po. Release tags are `vX.Y.Z`. `install.sh` writes `c3po` to `$C3PO_INSTALL_DIR` or `~/.local/bin` and links `rgw-ast`. The version, commit, and source hash are stamped into the binary. A dirty checkout is stamped `-dirty`.
## Requirements
### Requirement: Local install script
The repository SHALL provide `install.sh` that installs `c3po` from a local checkout to `$C3PO_INSTALL_DIR` or `$HOME/.local/bin`, and SHALL link `rgw-ast` to that binary. It SHALL NOT install `cx` or `complexity`. The version, commit, and source hash SHALL be stamped into the binary. A dirty checkout SHALL stamp `-dirty`.

#### Scenario: Install from checkout
- **GIVEN** a source checkout of c3po
- **WHEN** the operator runs `./install.sh`
- **THEN** a `c3po` binary SHALL be written to the install directory
- **AND** `rgw-ast` SHALL be a symlink to that binary
- **AND** `c3po version` SHALL run successfully

### Requirement: Public module
The Go module path SHALL be `github.com/ryangerardwilson/c3po`. The main package SHALL be `./cmd/c3po`.

#### Scenario: Tagged install
- **WHEN** the operator runs `go install github.com/ryangerardwilson/c3po/cmd/c3po@vX.Y.Z`
- **THEN** Go SHALL build the `c3po` command from that tag

### Requirement: Source verification without install
Agents and operators SHALL be able to verify the CLI without installing by running `GOWORK=off go test ./...`.

#### Scenario: Local verification
- **WHEN** `GOWORK=off go test ./...` is run in the repository root
- **THEN** all packages SHALL pass

