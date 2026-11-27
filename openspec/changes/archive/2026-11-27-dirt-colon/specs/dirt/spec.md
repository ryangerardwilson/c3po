## MODIFIED Requirements

### Requirement: Dirt lists dirty repositories
`c3po dirt` SHALL scan the home directory that `os.UserHomeDir` returns. `c3po dirt <path>` SHALL scan that path. A relative path SHALL be resolved from the process working directory and cleaned to an absolute path. Stdout SHALL list only dirty repositories, one line per repository, sorted by absolute path, with no blank lines and no file names. Each line SHALL be `<path>: <tally>`. A path equal to the home directory SHALL be `~`. A path under that directory SHALL be `~/` plus the slash-separated relative path. Any other path SHALL stay absolute. The colon SHALL follow the path immediately. One space SHALL follow the colon. The tally SHALL list only the marks whose count is greater than zero, in the order `M`, `+`, `-`, `??`, each mark followed immediately by its positive decimal count, with one space between marks. An example line is `~/src/app: M2 +3 -7 ??5`. `M` SHALL be modified, `+` added, `-` deleted, and `??` untracked. Each porcelain line other than the `##` branch line SHALL add one mark: `?` in either column is untracked, `D` in either column is deleted, `A` or `C` in either column is added, and every other line, including a rename, is modified. A line with `!` in either column SHALL add nothing. An untracked directory SHALL count as one untracked. A rename SHALL count as one modified. The `##` line SHALL NOT be printed. A repository whose tally is empty, including one whose status is only the `##` branch line, SHALL be omitted. A scan that finds no dirty repository SHALL print nothing and SHALL exit 0. A newline SHALL follow every line, including the last. The status SHALL come from `git --no-optional-locks -C <repo> status --porcelain=v1 -b --untracked-files=normal`. The command SHALL NOT stage, commit, clean, or check out. There SHALL be no dirt screen.

#### Scenario: A clean repository is omitted
- **GIVEN** a scan directory that holds one repository with nothing to commit
- **WHEN** the agent runs `c3po dirt <dir>`
- **THEN** stdout SHALL be empty
- **AND** the command SHALL exit 0

#### Scenario: A dirty repository is printed with its porcelain
- **GIVEN** a repository `<dir>/app` on branch `main` with a modified `README.md` and an untracked `notes.txt`
- **WHEN** the agent runs `c3po dirt <dir>`
- **THEN** stdout SHALL be the one line `<dir>/app: M1 ??1` followed by a newline
- **AND** stdout SHALL NOT name `## main`, `README.md`, or `notes.txt`
- **AND** the command SHALL exit 0

### Requirement: A scan inside a repository
When the scan path is inside a repository and is not its root, the nearest enclosing directory that contains `.git` SHALL be reported, and its tally SHALL count only the porcelain lines whose paths are inside the scan path. A rename SHALL count when either path is inside the scan path. An untracked directory that contains the scan path SHALL count. The enclosing repository SHALL be omitted when its tally is empty. Nested repositories under the scan path SHALL be separate entries. A repository whose root is the scan path, or under it, SHALL be tallied whole.

#### Scenario: Only the dirt under the scan path is shown
- **GIVEN** a repository `<dir>/app` with a modified `README.md` and a modified `src/main.go`
- **WHEN** the agent runs `c3po dirt <dir>/app/src`
- **THEN** stdout SHALL be the one line `<dir>/app: M1`
- **AND** stdout SHALL NOT name `README.md` or `src/main.go`
