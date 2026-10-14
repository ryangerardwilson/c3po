## Context

See `proposal.md` for motivation. Current mutations resolve paths beneath one
workspace and protect content changes with SHA-256 preconditions. Deletion must
fit that model while avoiding the broad semantics and recovery risk of `rm -r`
or glob expansion.

## Goals / Non-Goals

**Goals:**

- Make intentional single-file removal possible inside enforced workspaces.
- Preserve the same root containment and stale-observation protection as patch.
- Let repeated single-file deletions clean up their now-empty directory tree.
- Keep command output deterministic and useful to agent hooks.

**Non-Goals:**

- A recursive delete API, manifest batch, trash service, glob language, or Git
  staging wrapper.
- Mutation of symlinks, devices, sockets, or nested repository boundaries.

## Decisions

### Delete one regular file per invocation

The file layer uses `Lstat`, rejects non-regular modes, compares `HashFile`
against the caller's expected digest, then calls `os.Remove` for that exact
path. This keeps every destructive target explicit and observed immediately
before deletion.

Alternative: a recursive `delete-tree` command. Rejected because one stale or
misresolved directory argument would multiply destructive scope and could not
be represented by a single file hash.

### Prune empty ancestors only after successful deletion

`--prune-empty` walks from the deleted file's parent toward the resolved root,
calling `os.Remove` only on empty directories. A non-empty-directory error ends
pruning successfully; other errors fail visibly. The root itself is never an
eligible removal target.

Alternative: accept a directory hash or generated manifest. Rejected for this
release because ordering, concurrent changes, symlink semantics, and manifest
authenticity require a larger batch-transaction contract.

### Keep the hash mandatory in every mode

Unlike a policy toggle, deletion always requires `--expect-hash`, even below
the enforcement threshold. Deletion has no safe unguarded convenience mode,
and consistent grammar is easier for hooks and agents to reason about.

### Keep hook allowance structural

The hook already permits one direct leading `rgw-ast` invocation and rejects
compound shell syntax. Adding a built-in verb therefore needs no special shell
exception; managed guidance is updated so agents choose it instead of `rm` or
`git rm`.

## Risks / Trade-offs

- **Concurrent change between hash and removal** → Re-hash immediately before
  `os.Remove`; the command fails on mismatch.
- **Concurrent recreation after removal** → The command guarantees its own
  exact removal, not a filesystem-wide transaction; callers verify final state
  when this distinction matters.
- **Pruning encounters permissions or filesystem errors** → Report the error;
  the file may already be deleted, and output/documentation make that ordering
  explicit.
- **Many-file cleanup requires many calls** → Accept the extra calls in return
  for explicit targets and independently guarded observations.

## Migration Plan

1. Ship the new verb in the next patch release with tests and managed guidance.
2. Install the stamped release through the repository's release script.
3. Use one hash/delete call per stale file and `--prune-empty` for cleanup.
4. Roll back by reinstalling the preceding release; no config or data migration
   is required.
