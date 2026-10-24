# looptab Specification

## Purpose
Registers jobs files, edits them in vim, and installs them onto the Looptab scheduler. The c3po tui looptab screen runs one selected job now.
## Requirements
### Requirement: The screen runs the selected job now
`c3po tui looptab` SHALL open the job list. The list SHALL come from the jobs `looptab now` prints. Each row SHALL show the schedule, the kind, and the action. `j` and `k` SHALL move. Enter SHALL run the selected job once with `looptab now` and that row's index. `q` and ctrl-c SHALL leave the screen. `?` SHALL toggle a shortcut list. The job list SHALL NOT show shortcut text while that list is closed. While the shortcut list is open, `j` and enter SHALL NOT move or run a job, Esc SHALL close the list, and `q` SHALL still leave the screen. When `looptab` is not on `PATH`, the screen SHALL say `looptab is not on PATH` and SHALL NOT run a job. An empty schedule SHALL say `no jobs` and enter SHALL NOT run a job. After a run, the status SHALL say `ran <id>` when the command exits 0, and `<id> failed` otherwise. An extra argument SHALL print the screen help and SHALL exit 2. `c3po tui looptab help` SHALL exit 0 and SHALL say enter runs the selected job now. `c3po tui looptab version` SHALL print the c3po version.

#### Scenario: Help names the run
- **WHEN** the operator runs `c3po tui looptab help`
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
- **WHEN** the operator runs `c3po tui looptab extra`
- **THEN** the command SHALL exit 2
- **AND** the text SHALL include the screen help

### Requirement: Jobs files are registered and edited in vim
`c3po looptab` SHALL open the first registered jobs file in vim and then install every registered file onto the Looptab scheduler file. An empty registration set SHALL create `~/.config/c3po/looptab/jobs` from a comment template, register it as `jobs`, open it in vim, and install it. `c3po looptab edit [name]` SHALL open that file in vim and install. `c3po looptab register <path>` SHALL record an existing regular file and install it. The registered name SHALL be the lowercase basename without its extension. A duplicate name SHALL exit 2. The scheduler file SHALL be installed output, and registering that path SHALL exit 2. `c3po looptab remove <name>` SHALL drop that registration and install the rest. An unknown name SHALL exit 2. `c3po looptab list` SHALL print each name and path. An empty set SHALL print `no jobs files`. `c3po looptab show <name>` SHALL print the name, the path, and the file. `c3po looptab help` SHALL exit 0 and SHALL say the file opens in vim and SHALL name `c3po tui looptab`. `c3po looptab version` SHALL print the c3po version. Extra arguments SHALL print the help and SHALL exit 2. Registrations SHALL be stored in the sqlx database `c3po`. When vim is not on PATH, the command SHALL say `vim is not on PATH` and SHALL return. An empty registration set SHALL write a comment-only scheduler file and SHALL keep that file.

#### Scenario: Help names vim
- **WHEN** the operator runs `c3po looptab help`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL say in vim
- **AND** the text SHALL name `c3po tui looptab`

#### Scenario: A jobs file is registered and removed
- **WHEN** the operator registers a regular file and then removes its name
- **THEN** list shows the name
- **AND** the scheduler file contains the file body
- **AND** remove leaves `no jobs files`
- **AND** a second remove exits 2

#### Scenario: Edit installs the saved document
- **WHEN** the editor writes a job line and returns
- **THEN** the scheduler file contains that line
- **AND** the command prints `installed 1 jobs files`

