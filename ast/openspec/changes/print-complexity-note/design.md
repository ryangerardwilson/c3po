# Print a complexity note

## Behavior
On success, `create`, `patch`, and `apply` print their `ok` lines, then run `cx note --format text` on the source files that command wrote. The wait is five seconds. Empty output, a missing binary, a timeout, and a non-zero exit print nothing more. Deletes and moves are skipped. The command's error is unchanged.

## Evidence
A unit test with a stub `cx` on `PATH` shows the note after `ok`. A failing stub leaves stdout without a note and the command successful.

## Rollback
Remove the helper and the three calls. The `ok` lines stay.
