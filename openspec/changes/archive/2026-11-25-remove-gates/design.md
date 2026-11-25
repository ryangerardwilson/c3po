## Context

`gates` is a top-level c3po app. It prints a compiled statute and can tally a sheet. It has no screen and no other caller in this module. r2d2 does not call it. The operator now keeps the same kind of rule in r2d2 skills.

## Goals / Non-Goals

**Goals:**
- Remove the command, the package, and the usage feature. The living gates spec states only that the command is unknown.
- Make `c3po gates` an unknown command.

**Non-Goals:**
- No replacement statute inside c3po.
- No new r2d2 skill in this change.
- No edit to archived OpenSpec changes.
- No version bump.

## Decisions

- **Delete the package.** The statute exists only to be printed by this command. Leaving the Go types with no command would keep the text in the binary.
- **Unknown, not a hidden alias.** `c3po gates` joins the retired names and exits 2. Help and version of that name are not the c3po help or version commands.
- **Keep the archives.** Past changes under `openspec/changes/archive/` record what the command was. They are not instructions for the current binary.

## Risks / Trade-offs

- A script that still runs `c3po gates` exits 2 → expected. The replacement is an r2d2 skill, not a flag on this command.
