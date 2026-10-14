## 1. Guarded file layer

- [x] 1.1 Implement regular-file validation, immediate hash comparison, and exact file removal.
- [x] 1.2 Implement optional empty-ancestor pruning that stops before the workspace root.
- [x] 1.3 Add file-layer tests for success, stale hashes, directories, symlinks, pruning, and containment.

## 2. CLI and agent boundary

- [x] 2.1 Add `delete <file> --expect-hash <sha> [--prune-empty]` parsing, dispatch, and deterministic output.
- [x] 2.2 Add CLI tests for success, required arguments, failure preservation, root override, and pruning.
- [x] 2.3 Update compact help, README, managed AGENTS guidance, and hook-facing messages/tests.

## 3. Verification and release

- [x] 3.1 Run formatting, `go test ./...`, focused command smoke tests, and strict OpenSpec validation.
- [x] 3.2 Confirm the completed OpenSpec change is ready to archive and re-run strict validation after archive.
- [x] 3.3 Prepare generated release notes and verify the release/installation script before archiving.
