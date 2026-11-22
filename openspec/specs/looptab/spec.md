# looptab Specification

## Purpose
Registers jobs files and installs them onto the Looptab scheduler from the command line. The r2d2 looptab screen runs one selected job now and edits the jobs file in vim, calling c3po to install it.
## Requirements
### Requirement: The screen runs the selected job now
`r2d2 looptab` SHALL open the job list. The screen SHALL be implemented in r2d2 and SHALL call the exported c3po looptab functions in process. The c3po screen package SHALL NOT open it, and `r2d2 looptab` SHALL NOT shell out to `c3po`. The list SHALL come from the jobs `looptab now` prints. Each row SHALL show the schedule, the kind, and the action. `j` and `k` SHALL move and SHALL wrap. On the last job, `j` SHALL select the first job. On the first job, `k` SHALL select the last job. An empty list SHALL leave the cursor in place. `g` and `G` SHALL still move to the first and last job. Enter SHALL run the selected job once with `looptab now` and that row's index. `q` and ctrl-c SHALL leave the screen. `?` SHALL toggle a shortcut list. The job list SHALL NOT show shortcut text while that list is closed. While the shortcut list is open, `j`, `e`, and enter SHALL NOT move, edit, or run a job, Esc SHALL close the list, and `q` SHALL still leave the screen. When `looptab` is not on `PATH`, the screen SHALL say `looptab is not on PATH` and SHALL NOT run a job. An empty schedule SHALL say `no jobs` and enter SHALL NOT run a job. After a run, the status SHALL say `ran <id>` when the command exits 0, and `<id> failed` otherwise. An extra argument SHALL print the screen help and SHALL exit 2. `r2d2 looptab help` SHALL exit 0, SHALL say enter runs the selected job now, and SHALL say `e` edits the jobs file in vim. `r2d2 looptab version` SHALL print the r2d2 version. `r2d2 looptab --resume <id>` SHALL restore the selected job from the c3po screen session.

#### Scenario: Help names the run
- **WHEN** the operator runs `r2d2 looptab help`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL say enter runs the selected job now
- **AND** the text SHALL say `e` edits the jobs file in vim

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

### Requirement: Jobs files are registered and installed from the CLI
`c3po looptab` SHALL be a command line and SHALL NOT open vim or any other terminal screen. With no arguments it SHALL print the help, SHALL exit 0, and SHALL NOT create or register a file. `edit` SHALL NOT be a command; `c3po looptab edit` SHALL print the help and SHALL exit 2. `c3po looptab register <path>` SHALL record an existing regular file and install it. The registered name SHALL be the lowercase basename without its extension. A duplicate name SHALL exit 2. The scheduler file `~/.config/looptab/looptab` SHALL be installed output, and registering that path SHALL exit 2. `c3po looptab remove <name>` SHALL drop that registration and install the rest. An unknown name SHALL exit 2. `c3po looptab list` SHALL print each name and path. An empty set SHALL print `no jobs files`. `c3po looptab show <name>` SHALL print the name, the path, and the file. `c3po looptab help` SHALL exit 0, SHALL say the jobs file is edited with `r2d2 looptab`, and SHALL NOT name vim. `c3po looptab version` SHALL print the c3po version. Extra arguments SHALL print the help and SHALL exit 2. Registrations SHALL be stored in the sqlx database `c3po`. An empty registration set SHALL write a comment-only scheduler file and SHALL keep that file. c3po SHALL be the only writer of the scheduler file, and register, remove, and the install after an edit SHALL write it through one install function. c3po SHALL export `JobsFile`, which returns the first registered jobs file, and with none registered creates `~/.config/c3po/looptab/jobs` from a comment template, registers it as `jobs`, and installs it. c3po SHALL export `Install`, which installs every registered file and returns how many it installed.

#### Scenario: No arguments print the help
- **WHEN** the operator runs `c3po looptab` with nothing registered
- **THEN** the command SHALL exit 0 and SHALL print the help
- **AND** no jobs file SHALL be created and no scheduler file SHALL be written

#### Scenario: Edit is not a command
- **WHEN** the operator runs `c3po looptab edit`
- **THEN** the command SHALL exit 2
- **AND** the text SHALL include the help

#### Scenario: Help names the screen
- **WHEN** the operator runs `c3po looptab help`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL say the jobs file is edited with `r2d2 looptab`
- **AND** the text SHALL NOT name vim

#### Scenario: A jobs file is registered and removed
- **WHEN** the operator registers a regular file and then removes its name
- **THEN** list shows the name
- **AND** the scheduler file contains the file body
- **AND** remove leaves `no jobs files`
- **AND** a second remove exits 2

#### Scenario: An empty set gets the template
- **WHEN** nothing is registered and the screen asks c3po for the jobs file
- **THEN** `~/.config/c3po/looptab/jobs` SHALL hold the comment template
- **AND** it SHALL be registered as `jobs` and installed

### Requirement: The screen edits the jobs file in vim
On `r2d2 looptab`, `e` SHALL ask c3po for the jobs file, SHALL open it in vim, and after vim exits SHALL ask c3po to install every registered file and SHALL reload the job list. With no registered file, `e` SHALL leave `~/.config/c3po/looptab/jobs` created from the template, registered as `jobs`, opened in vim, and installed. When vim is not on `PATH`, the screen SHALL say `vim is not on PATH`, SHALL stay open, and SHALL NOT install. The screen SHALL NOT write the scheduler file itself.

#### Scenario: e installs the saved file and reloads
- **WHEN** the operator presses `e` and vim saves a job line and exits
- **THEN** c3po SHALL install the registered files
- **AND** the job list SHALL reload

#### Scenario: Missing vim leaves the screen up
- **WHEN** vim is not on `PATH` and the operator presses `e`
- **THEN** the screen SHALL say `vim is not on PATH`
- **AND** the screen SHALL stay open and nothing SHALL be installed

