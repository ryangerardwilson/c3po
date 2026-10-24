## Context

`looptab now` prints the registered jobs and, with an index or job id, runs that job once. `looptab now` with no argument then asks for a line of input. c3po screens are names under `c3po tui`. A listing screen hides its shortcut text until `?`.

## Goals / Non-Goals

- List every configured job and run the selected one immediately.
- Keep the job's own output on the terminal while it runs, then return to the list.
- Leave the Looptab scheduler, file format, and `looptab` command unchanged.
- Do not add a top-level `c3po looptab` command.

## Decisions

- The screen shells out to the `looptab` binary. c3po does not import the Looptab module.
- The list is the text `looptab now` prints before it asks for a selection. Fields are separated by two spaces, so a schedule such as `hourly at 15` stays one field.
- Enter runs `looptab now` with the selected row's index. That index is the position `looptab now` printed, and `looptab now <index>` runs any job, including one whose schedule is not `now`.
- The screen leaves the alternate screen while that command runs, then restores the list. The status is `ran <id>` or `<id> failed`.
- A missing `looptab` binary or an empty file stays on the list and does not start a command.
- `j`, `k`, and enter do nothing while the shortcut list is open. `q` still quits. Esc closes the list.

## Risks / Trade-offs

- The list parser follows the current `looptab now` text. A change to that text needs a matching parser update.
- A job id that is only digits would be read as an index, so the screen passes the index it just listed.
