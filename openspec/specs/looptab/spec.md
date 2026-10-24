# looptab Specification

## Purpose
Lists the configured Looptab jobs and runs the selected job once, immediately, from the c3po tui looptab screen.

## Requirements
### Requirement: The screen runs the selected job now
`c3po tui looptab` SHALL be the only command that opens the job list. The list SHALL come from the jobs `looptab now` prints. Each row SHALL show the schedule, the kind, and the action. `j` and `k` SHALL move. Enter SHALL run the selected job once with `looptab now` and that row's index. `q` and ctrl-c SHALL leave the screen. `?` SHALL toggle a shortcut list. The job list SHALL NOT show shortcut text while that list is closed. While the shortcut list is open, `j` and enter SHALL NOT move or run a job, Esc SHALL close the list, and `q` SHALL still leave the screen. When `looptab` is not on `PATH`, the screen SHALL say `looptab is not on PATH` and SHALL NOT run a job. An empty schedule SHALL say `no jobs` and enter SHALL NOT run a job. After a run, the status SHALL say `ran <id>` when the command exits 0, and `<id> failed` otherwise. An extra argument SHALL print the screen help and SHALL exit 2. `c3po tui looptab help` SHALL exit 0 and SHALL say enter runs the selected job now. `c3po tui looptab version` SHALL print the c3po version. `c3po looptab` SHALL exit 2.

#### Scenario: Help names the run
- **WHEN** the operator runs `c3po tui looptab help`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL say enter runs the selected job now
- **AND** `c3po looptab` SHALL exit 2

#### Scenario: Enter runs the selected index
- **WHEN** the list has two jobs and the operator moves to the second and presses enter
- **THEN** the screen SHALL run `looptab now` with that row's index
- **AND** the status SHALL say `ran` and that job's id

#### Scenario: Question mark blocks a run
- **WHEN** the shortcut list is open and the operator presses `j` and then enter
- **THEN** the cursor SHALL stay on the same job
- **AND** no job SHALL run

#### Scenario: A missing binary does not run
- **WHEN** `looptab` is not on `PATH` and the operator presses enter
- **THEN** the screen SHALL say `looptab is not on PATH`
- **AND** no job SHALL run

#### Scenario: An empty list does not run
- **WHEN** no jobs are configured and the operator presses enter
- **THEN** the screen SHALL say `no jobs`
- **AND** no job SHALL run

#### Scenario: Extra arguments are rejected
- **WHEN** the operator runs `c3po tui looptab extra`
- **THEN** the command SHALL exit 2
- **AND** the text SHALL include the screen help

