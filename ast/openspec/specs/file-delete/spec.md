# file-delete Specification

## Purpose
Provides a deliberate deletion boundary for removing one verified workspace
file without enabling recursive, ambiguous, or outside-root destruction.
## Requirements
### Requirement: Hash-guarded file deletion

`rgw-ast delete <file> --expect-hash <sha256>` SHALL remove exactly one existing
regular file only when its current SHA-256 equals the expected hash. Success
SHALL identify the deleted workspace-relative path without printing its
contents.

#### Scenario: Matching file is deleted

- **GIVEN** a regular file under the workspace root with current hash H
- **WHEN** the user runs `rgw-ast delete <file> --expect-hash H`
- **THEN** that file SHALL no longer exist
- **AND** the command SHALL exit 0 and report its workspace-relative path

#### Scenario: Stale hash is rejected

- **GIVEN** a regular file whose current hash differs from H
- **WHEN** the user runs `rgw-ast delete <file> --expect-hash H`
- **THEN** the command SHALL fail
- **AND** the file MUST remain unchanged

### Requirement: Deletion target containment

The delete command SHALL reject a workspace root, directory, symlink, special
file, missing path, and any path that resolves outside the selected workspace.
It MUST NOT follow a symlink to delete its target.

#### Scenario: Directory deletion is attempted

- **GIVEN** the selected path is a directory
- **WHEN** the user invokes `rgw-ast delete` for that path
- **THEN** the command SHALL fail without removing the directory or its contents

#### Scenario: Symlink deletion is attempted

- **GIVEN** the selected path is a symlink inside the workspace
- **WHEN** the user invokes `rgw-ast delete` for that path
- **THEN** the command SHALL fail
- **AND** the symlink target MUST remain unchanged

### Requirement: Bounded empty-directory pruning

When `--prune-empty` is present and guarded file deletion succeeds, rgw-ast
SHALL attempt to remove empty ancestor directories. Pruning SHALL stop at the
first non-empty ancestor and MUST stop before the workspace root.

#### Scenario: Deleted file was the last nested entry

- **GIVEN** a verified file is the only remaining entry beneath one or more
  nested directories
- **WHEN** deletion succeeds with `--prune-empty`
- **THEN** those empty nested directories SHALL be removed
- **AND** the workspace root MUST remain

#### Scenario: Ancestor contains another entry

- **GIVEN** an ancestor of the deleted file contains another entry
- **WHEN** pruning reaches that ancestor
- **THEN** pruning SHALL stop
- **AND** that ancestor and its remaining entries MUST remain unchanged
