## MODIFIED Requirements

### Requirement: Search command
`rgw-ast search <query>` SHALL search workspace source (include/exclude
applied) for literal substring matches and print capped results
(`max_search_hits` from config). Each hit SHALL include path, line number, and
a content snippet truncated to at most 120 characters (with an ellipsis when
truncated).

#### Scenario: Search cap
- **GIVEN** more matches exist than `max_search_hits`
- **WHEN** search runs
- **THEN** at most `max_search_hits` hits SHALL be printed
- **AND** the tool MAY indicate that results were truncated

#### Scenario: Snippet truncation
- **GIVEN** a matching line longer than 120 characters
- **WHEN** search prints the hit
- **THEN** the content field SHALL be at most 120 characters plus an ellipsis
  indicator when truncated
