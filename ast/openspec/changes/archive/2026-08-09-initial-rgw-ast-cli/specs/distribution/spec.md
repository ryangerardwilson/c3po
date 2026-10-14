## ADDED Requirements

### Requirement: Local install script
The repository SHALL provide `install.sh` that can install the binary to
`$HOME/.local/bin/rgw-ast` (or `$RGW_AST_INSTALL_DIR`) from a local checkout
via `install.sh from <path>`.

#### Scenario: Install from checkout
- **GIVEN** a source checkout of rgw-ast
- **WHEN** the operator runs `./install.sh from .`
- **THEN** a `rgw-ast` binary SHALL be written to the install directory
- **AND** `rgw-ast version` SHALL run successfully from that path

### Requirement: Source verification without install
Agents and operators SHALL be able to verify the CLI without installing by
running `go test ./...` and `go run ./cmd/rgw-ast help`.

#### Scenario: Local verification
- **WHEN** `go test ./...` is run in the repository root
- **THEN** all packages SHALL pass
