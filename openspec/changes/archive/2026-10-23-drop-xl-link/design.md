## Context

`install.sh` links `xl` to the c3po binary, and the program name `xl` runs the spreadsheet. The screen is already `c3po tui xl`. There is no shell function or desktop entry for `xl`. `~/.local/bin/xl` is that symlink. `vixl` is a different command.

## Goals / Non-Goals

**Goals:**

- Remove the `xl` symlink and stop creating it.
- Stop selecting the spreadsheet from the program name `xl`.

**Non-Goals:**

- Removing `c3po tui xl`.
- Removing `vixl`.
- Changing how `o` opens an `.xlsx` file.

## Decisions

`remove_compat_link xl` uses the same rule as `cx` and `complexity`: delete the name only when it is a symlink to c3po. A real file with that name is left alone. The `rgw-ast` link stays.

`toolOf` keeps only `rgw-ast`. A process started as `xl` is dispatched as c3po, so `xl version` prints the c3po version.

## Risks / Trade-offs

- [A habit of typing `xl` stops working] → The screen is `c3po tui xl`. Help names that command.
