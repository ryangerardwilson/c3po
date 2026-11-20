## MODIFIED Requirements

### Requirement: The screen runs the selected job now
`r2d2 looptab` SHALL open the job list. The list SHALL come from the jobs `looptab now` prints. Each row SHALL show the schedule, the kind, and the action. `j` and `k` SHALL move and SHALL wrap. On the last job, `j` SHALL select the first job. On the first job, `k` SHALL select the last job. An empty list SHALL leave the cursor in place. `g` and `G` SHALL still move to the first and last job. Enter SHALL run the selected job once with `looptab now` and that row's index. `q` and ctrl-c SHALL leave the screen. `?` SHALL toggle a shortcut list. The job list SHALL NOT show shortcut text while that list is closed. While the shortcut list is open, `j` and enter SHALL NOT move or run a job, Esc SHALL close the list, and `q` SHALL still leave the screen. When `looptab` is not on `PATH`, the screen SHALL say `looptab is not on PATH` and SHALL NOT run a job. An empty schedule SHALL say `no jobs` and enter SHALL NOT run a job. After a run, the status SHALL say `ran <id>` when the command exits 0, and `<id> failed` otherwise. An extra argument SHALL print the screen help and SHALL exit 2. `r2d2 looptab help` SHALL exit 0 and SHALL say enter runs the selected job now. `r2d2 looptab version` SHALL print the c3po version.

#### Scenario: Help names the run
- **WHEN** the operator runs `r2d2 looptab help`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL say enter runs the selected job now

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
- **WHEN** the operator runs `r2d2 looptab extra`
- **THEN** the command SHALL exit 2
- **AND** the text SHALL include the screen help

#### Scenario: j and k wrap the jobs
- **WHEN** the list has two jobs and the cursor is on the second
- **AND** the operator presses `j`
- **THEN** the cursor SHALL be on the first job
- **AND** `k` SHALL move it to the second job
