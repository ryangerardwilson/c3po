# nlp Specification

## Purpose

The neuro-linguistic programming trainer is the crusade mode `crusade -m nlp`. c3po does not open it, and its courses stay in the sqlx database `nlp`.

## Requirements

### Requirement: c3po does not open the trainer
`c3po tui nlp` SHALL exit 2. `c3po nlp` SHALL exit 2. `c3po help -f tui nlp` SHALL exit 2. c3po SHALL NOT list `nlp` as a terminal screen.

#### Scenario: The screen is rejected
- **WHEN** the operator runs `c3po tui nlp`
- **THEN** the command SHALL exit 2

#### Scenario: The bare name is rejected
- **WHEN** the operator runs `c3po nlp`
- **THEN** the command SHALL exit 2

#### Scenario: Help does not describe the trainer
- **WHEN** the operator runs `c3po help -f tui nlp`
- **THEN** the command SHALL exit 2
