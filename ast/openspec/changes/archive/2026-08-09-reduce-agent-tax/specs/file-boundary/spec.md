## MODIFIED Requirements

### Requirement: Bounded read
`rgw-ast read <file> --lines <start>-<end>` SHALL print only the inclusive
line range requested. Line numbers are 1-based. The range length SHALL NOT
exceed `max_read_lines` from global config.

When enforcement is active and `deny_whole_file_read` is true, `rgw-ast read`
without `--lines` SHALL fail for any non-binary regular file under the
workspace root (not only include-glob matched source files).

#### Scenario: Bounded slice
- **WHEN** the user runs `rgw-ast read f.go --lines 1-10`
- **THEN** stdout SHALL contain at most 10 lines of file content

#### Scenario: Whole-file read denied when enforced
- **GIVEN** enforcement is active and `deny_whole_file_read` is true
- **WHEN** the user runs `rgw-ast read src/main.go` without `--lines`
- **THEN** the process SHALL exit non-zero
- **AND** stderr SHALL direct the user to `--lines` or intelligence commands

#### Scenario: Whole-file read denied for json when enforced
- **GIVEN** enforcement is active and `deny_whole_file_read` is true
- **WHEN** the user runs `rgw-ast read package.json` without `--lines`
- **THEN** the process SHALL exit non-zero
- **AND** stderr SHALL direct the user to `--lines` or intelligence commands
