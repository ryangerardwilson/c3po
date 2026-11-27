## MODIFIED Requirements

### Requirement: Dirt lists dirty repositories
`c3po dirt` SHALL scan the home directory that `os.UserHomeDir` returns. `c3po dirt <path>` SHALL scan that path. A relative path SHALL be resolved from the process working directory and cleaned to an absolute path. Stdout SHALL list only dirty repositories, one line per repository, sorted by absolute path, with no blank lines and no file names. Each line SHALL be `<absolute-path>  <count>`: the absolute path, two spaces, and the count as a positive decimal integer. A newline SHALL follow every line, including the last. The count SHALL be the number of `git status --porcelain=v1 -b` lines for that repository that are not the `##` branch line. The `##` line SHALL NOT count and SHALL NOT be printed. A rename SHALL count as 1. An untracked-directory line SHALL count as 1. A repository whose count is 0, including one whose status is only the `##` branch line, SHALL be clean and SHALL be omitted. A scan that finds no dirty repository SHALL print nothing and SHALL exit 0. The status SHALL come from `git --no-optional-locks -C <repo> status --porcelain=v1 -b --untracked-files=normal`. The command SHALL NOT stage, commit, clean, or check out. There SHALL be no dirt screen.

#### Scenario: A clean repository is omitted
- **GIVEN** a scan directory that holds one repository with nothing to commit
- **WHEN** the agent runs `c3po dirt <dir>`
- **THEN** stdout SHALL be empty
- **AND** the command SHALL exit 0

#### Scenario: A dirty repository is printed with its porcelain
- **GIVEN** a repository `<dir>/app` on branch `main` with a modified `README.md` and an untracked `notes.txt`
- **WHEN** the agent runs `c3po dirt <dir>`
- **THEN** stdout SHALL be the one line `<dir>/app  2` followed by a newline
- **AND** stdout SHALL NOT name `## main`, `README.md`, or `notes.txt`
- **AND** the command SHALL exit 0

### Requirement: Dirt walks the tree
A directory that contains `.git`, as a directory or a file, SHALL be a repository. The walk SHALL NOT descend into `.git`. The walk SHALL descend through a repository's work tree, so a nested repository is its own entry. The walk SHALL NOT follow symlinks. A directory the process cannot list SHALL be skipped and SHALL NOT fail the scan. Below the scan root the walk SHALL skip directories named `node_modules`, `.venv`, `venv`, `.cache`, `__pycache__`, `.npm`, `.cargo`, and `site-packages`. A scan root with one of those names SHALL still be scanned. The walk SHALL NOT enter the directory `~/.local` returned with the home directory from `os.UserHomeDir`, unless that directory is the scan root. A directory named `.local` that is not that path SHALL still be entered.

#### Scenario: A nested repository is reported separately
- **GIVEN** a dirty repository `<dir>/app` and a dirty repository `<dir>/app/vendor/lib` inside its work tree
- **WHEN** the agent runs `c3po dirt <dir>`
- **THEN** stdout SHALL be the line for `<dir>/app` and then the line for `<dir>/app/vendor/lib`, each with its own count
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
- **THEN** stdout SHALL name `Apps/app`
- **AND** stdout SHALL NOT name `.local/share/plugin`
- **AND** `c3po dirt` of that `.local` directory SHALL name the repository under it
- **AND** a dirty repository under a different directory named `.local` SHALL still be named

### Requirement: A scan inside a repository
When the scan path is inside a repository and is not its root, the nearest enclosing directory that contains `.git` SHALL be reported, and its count SHALL be only the non-`##` porcelain lines whose paths are inside the scan path. A rename SHALL count when either path is inside the scan path. An untracked directory that contains the scan path SHALL count. The enclosing repository SHALL be omitted when its count is 0. Nested repositories under the scan path SHALL be separate entries. A repository whose root is the scan path, or under it, SHALL be counted whole.

#### Scenario: Only the dirt under the scan path is shown
- **GIVEN** a repository `<dir>/app` with a modified `README.md` and a modified `src/main.go`
- **WHEN** the agent runs `c3po dirt <dir>/app/src`
- **THEN** stdout SHALL be the one line `<dir>/app  1`
- **AND** stdout SHALL NOT name `README.md` or `src/main.go`

### Requirement: A failed status
A bare repository, or any repository whose status command fails, SHALL be named on stderr with the git error and SHALL make the exit code 1. The repositories that were judged SHALL still be printed to stdout. The command SHALL exit 0 when every repository was judged.

#### Scenario: A failed git status exits 1 after printing the judged repositories
- **GIVEN** a dirty repository `<dir>/app` and a directory `<dir>/broken` whose `.git` is a bare repository
- **WHEN** the agent runs `c3po dirt <dir>`
- **THEN** stdout SHALL be the line `<dir>/app` with its count
- **AND** stdout SHALL NOT name `<dir>/broken`
- **AND** stderr SHALL name `<dir>/broken`
- **AND** the command SHALL exit 1
