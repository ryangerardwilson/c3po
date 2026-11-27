## MODIFIED Requirements

### Requirement: Dirt lists dirty repositories
`c3po dirt` SHALL scan the home directory that `os.UserHomeDir` returns. `c3po dirt <path>` SHALL scan that path. A relative path SHALL be resolved from the process working directory and cleaned to an absolute path. Stdout SHALL list only dirty repositories, one line per repository, sorted by absolute path, with no blank lines and no file names. Each line SHALL be `<path> : <tally>`. A path equal to the home directory SHALL be `~`. A path under that directory SHALL be `~/` plus the slash-separated relative path. Any other path SHALL stay absolute. One space SHALL stand on each side of the colon. The tally SHALL list only the marks whose count is greater than zero, in the order `M`, `+`, `-`, `??`, each mark followed immediately by its positive decimal count, with one space between marks. An example line is `~/src/app : M2 +3 -7 ??5`. `M` SHALL be modified, `+` added, `-` deleted, and `??` untracked. Each porcelain line other than the `##` branch line SHALL add one mark: `?` in either column is untracked, `D` in either column is deleted, `A` or `C` in either column is added, and every other line, including a rename, is modified. A line with `!` in either column SHALL add nothing. An untracked directory SHALL count as one untracked. A rename SHALL count as one modified. The `##` line SHALL NOT be printed. A repository whose tally is empty, including one whose status is only the `##` branch line, SHALL be omitted. A scan that finds no dirty repository SHALL print nothing and SHALL exit 0. A newline SHALL follow every line, including the last. The status SHALL come from `git --no-optional-locks -C <repo> status --porcelain=v1 -b --untracked-files=normal`. The command SHALL NOT stage, commit, clean, or check out. There SHALL be no dirt screen.

#### Scenario: A clean repository is omitted
- **GIVEN** a scan directory that holds one repository with nothing to commit
- **WHEN** the agent runs `c3po dirt <dir>`
- **THEN** stdout SHALL be empty
- **AND** the command SHALL exit 0

#### Scenario: A dirty repository is printed with its porcelain
- **GIVEN** a repository `<dir>/app` on branch `main` with a modified `README.md` and an untracked `notes.txt`
- **WHEN** the agent runs `c3po dirt <dir>`
- **THEN** stdout SHALL be the one line `<dir>/app : M1 ??1` followed by a newline
- **AND** stdout SHALL NOT name `## main`, `README.md`, or `notes.txt`
- **AND** the command SHALL exit 0

### Requirement: Dirt walks the tree
A directory that contains `.git`, as a directory or a file, SHALL be a repository. The walk SHALL NOT descend into `.git`. The walk SHALL descend through a repository's work tree, so a nested repository is its own entry. The walk SHALL NOT follow symlinks. A directory the process cannot list SHALL be skipped and SHALL NOT fail the scan. Below the scan root the walk SHALL skip directories named `node_modules`, `.venv`, `venv`, `.cache`, `__pycache__`, `.npm`, `.cargo`, and `site-packages`. A scan root with one of those names SHALL still be scanned. The walk SHALL NOT enter the directory `~/.local` returned with the home directory from `os.UserHomeDir`, unless that directory is the scan root. A directory named `.local` that is not that path SHALL still be entered.

#### Scenario: A nested repository is reported separately
- **GIVEN** a dirty repository `<dir>/app` and a dirty repository `<dir>/app/vendor/lib` inside its work tree
- **WHEN** the agent runs `c3po dirt <dir>`
- **THEN** stdout SHALL be the line for `<dir>/app` and then the line for `<dir>/app/vendor/lib`, each with its own tally
- **AND** no blank line SHALL separate them

#### Scenario: A symlink is not followed
- **GIVEN** a dirty repository outside the scan directory and a symlink inside the scan directory that points at it
- **WHEN** the agent runs `c3po dirt <dir>`
- **THEN** stdout SHALL NOT name the repository or the symlink

#### Scenario: node_modules is skipped
- **GIVEN** a dirty repository at `<dir>/web/node_modules/pkg`
- **WHEN** the agent runs `c3po dirt <dir>`
- **THEN** stdout SHALL NOT name `<dir>/web/node_modules/pkg`
- **AND** `c3po dirt <dir>/web/node_modules` SHALL list it

#### Scenario: The home .local directory is skipped
- **GIVEN** the home directory holds a dirty repository at `.local/share/plugin` and a dirty repository at `Apps/app`
- **WHEN** the agent runs `c3po dirt`
- **THEN** stdout SHALL name `~/Apps/app`
- **AND** stdout SHALL NOT name `.local/share/plugin`
- **AND** `c3po dirt` of that `.local` directory SHALL name `~/.local/share/plugin`
- **AND** a dirty repository under a different directory named `.local` SHALL still be named

### Requirement: A scan inside a repository
When the scan path is inside a repository and is not its root, the nearest enclosing directory that contains `.git` SHALL be reported, and its tally SHALL count only the porcelain lines whose paths are inside the scan path. A rename SHALL count when either path is inside the scan path. An untracked directory that contains the scan path SHALL count. The enclosing repository SHALL be omitted when its tally is empty. Nested repositories under the scan path SHALL be separate entries. A repository whose root is the scan path, or under it, SHALL be tallied whole.

#### Scenario: Only the dirt under the scan path is shown
- **GIVEN** a repository `<dir>/app` with a modified `README.md` and a modified `src/main.go`
- **WHEN** the agent runs `c3po dirt <dir>/app/src`
- **THEN** stdout SHALL be the one line `<dir>/app : M1`
- **AND** stdout SHALL NOT name `README.md` or `src/main.go`

### Requirement: A failed status
A bare repository, or any repository whose status command fails, SHALL be named on stderr with the git error and SHALL make the exit code 1. The repositories that were judged SHALL still be printed to stdout. The command SHALL exit 0 when every repository was judged.

#### Scenario: A failed git status exits 1 after printing the judged repositories
- **GIVEN** a dirty repository `<dir>/app` and a directory `<dir>/broken` whose `.git` is a bare repository
- **WHEN** the agent runs `c3po dirt <dir>`
- **THEN** stdout SHALL be the line `<dir>/app` with its tally
- **AND** stdout SHALL NOT name `<dir>/broken`
- **AND** stderr SHALL name `<dir>/broken`
- **AND** the command SHALL exit 1
