## ADDED Requirements

### Requirement: Cyclomatic complexity is enforced for Go, TypeScript, and Python
A create, patch, apply, or append that adds a Go, TypeScript, or Python function at max_cc or above, or raises one from below max_cc to max_cc or above, SHALL be refused and SHALL NOT be written. A function already at or above max_cc MAY still be edited. The languages are `.go`, `.ts`, `.tsx`, and `.py`. A refusal SHALL name the function and its score. A Go file that does not parse, or a scored file whose functions share a name, SHALL be refused and SHALL NOT be written, and that refusal SHALL say unparsed.

#### Scenario: A new Python function over the line is refused
- **WHEN** an agent creates a Python function whose cyclomatic score is max_cc or above
- **THEN** the write SHALL be refused
- **AND** the file SHALL NOT contain that function

#### Scenario: A new TypeScript function over the line is refused
- **WHEN** an agent creates a TypeScript function whose cyclomatic score is max_cc or above
- **THEN** the write SHALL be refused
- **AND** the file SHALL NOT contain that function

#### Scenario: A Go function already over the line stays editable
- **WHEN** a Go function is already at or above max_cc
- **THEN** a later edit of that function SHALL still be allowed

#### Scenario: Unparsed Go is refused
- **WHEN** an agent writes a Go file that go/parser rejects
- **THEN** the write SHALL be refused
- **AND** the refusal SHALL say unparsed
- **AND** the file SHALL NOT contain the rejected bytes
