## ADDED Requirements

### Requirement: max_cc policy
The global config SHALL accept an integer `max_cc`, the cyclomatic number a
plain Go patch may not raise a function declaration to from below. When the
key is missing, zero, or negative, the effective value SHALL be 30. The
default SHALL be applied in memory; reading the config SHALL NOT rewrite the
operator's file.

#### Scenario: Missing max_cc defaults to 30
- **GIVEN** a config.toml without `max_cc`
- **WHEN** the config is read
- **THEN** the effective `max_cc` SHALL be 30
- **AND** the file SHALL be unchanged

#### Scenario: Non-positive max_cc defaults to 30
- **GIVEN** a config.toml with `max_cc = 0` or `max_cc = -4`
- **WHEN** the config is read
- **THEN** the effective `max_cc` SHALL be 30

#### Scenario: Configured max_cc is honored
- **GIVEN** a config.toml with `max_cc = 6`
- **WHEN** a plain Go patch raises a declaration from 1 to 6
- **THEN** the patch SHALL be refused with `crosses 6`
