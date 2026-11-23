# dirt Specification

## Purpose
c3po dirt shows an agent every dirty git repository under the home directory or one path. It only reads.
## Requirements
### Requirement: Dirt lists dirty repositories
`c3po dirt` SHALL scan the home directory that `os.UserHomeDir` returns. `c3po dirt <path>` SHALL scan that path. A relative path SHALL be resolved from the process working directory and cleaned to an absolute path. Stdout SHALL list only dirty repositories, sorted by absolute path, with one blank line between repositories. Each repository SHALL be its absolute path on one line, then the output of `git status --porcelain=v1 -b` for that repository. A repository whose status is only the `##` branch line SHALL be clean and SHALL be omitted. A scan that finds no dirty repository SHALL print nothing and SHALL exit 0. The status SHALL come from `git --no-optional-locks -C <repo> status --porcelain=v1 -b --untracked-files=normal`. The command SHALL NOT stage, commit, clean, or check out. There SHALL be no dirt screen.

#### Scenario: A clean repository is omitted
- **GIVEN** a scan directory that holds one repository with nothing to commit
- **WHEN** the agent runs `c3po dirt <dir>`
- **THEN** stdout SHALL be empty
- **AND** the command SHALL exit 0

#### Scenario: A dirty repository is printed with its porcelain
- **GIVEN** a repository `<dir>/app` on branch `main` with a modified `README.md` and an untracked `notes.txt`
- **WHEN** the agent runs `c3po dirt <dir>`
- **THEN** stdout SHALL be `<dir>/app`, then `## main`, ` M README.md`, and `?? notes.txt`
- **AND** the command SHALL exit 0

### Requirement: Dirt walks the tree
A directory that contains `.git`, as a directory or a file, SHALL be a repository. The walk SHALL NOT descend into `.git`. The walk SHALL descend through a repository's work tree, so a nested repository is its own entry. The walk SHALL NOT follow symlinks. A directory the process cannot list SHALL be skipped and SHALL NOT fail the scan. Below the scan root the walk SHALL skip directories named `node_modules`, `.venv`, `venv`, `.cache`, `__pycache__`, `.npm`, `.cargo`, and `site-packages`. A scan root with one of those names SHALL still be scanned. The walk SHALL NOT enter the directory `~/.local` returned with the home directory from `os.UserHomeDir`, unless that directory is the scan root. A directory named `.local` that is not that path SHALL still be entered.

#### Scenario: A nested repository is reported separately
- **GIVEN** a dirty repository `<dir>/app` and a dirty repository `<dir>/app/vendor/lib` inside its work tree
- **WHEN** the agent runs `c3po dirt <dir>`
- **THEN** stdout SHALL list `<dir>/app` and then `<dir>/app/vendor/lib`, each with its own porcelain
- **AND** one blank line SHALL separate them

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
When the scan path is inside a repository and is not its root, the nearest enclosing directory that contains `.git` SHALL be reported, and its porcelain SHALL keep the `##` line and only the paths inside the scan path. A rename SHALL be kept when either path is inside the scan path. An untracked directory that contains the scan path SHALL be kept. The enclosing repository SHALL be omitted when no path is kept. Nested repositories under the scan path SHALL be separate entries. A repository whose root is the scan path, or under it, SHALL be included whole.

#### Scenario: Only the dirt under the scan path is shown
- **GIVEN** a repository `<dir>/app` with a modified `README.md` and a modified `src/main.go`
- **WHEN** the agent runs `c3po dirt <dir>/app/src`
- **THEN** stdout SHALL be `<dir>/app`, then its `##` line and ` M src/main.go`
- **AND** stdout SHALL NOT name `README.md`

### Requirement: A failed status
A bare repository, or any repository whose status command fails, SHALL be named on stderr with the git error and SHALL make the exit code 1. The repositories that were judged SHALL still be printed to stdout. The command SHALL exit 0 when every repository was judged.

#### Scenario: A failed git status exits 1 after printing the judged repositories
- **GIVEN** a dirty repository `<dir>/app` and a directory `<dir>/broken` whose `.git` is a bare repository
- **WHEN** the agent runs `c3po dirt <dir>`
- **THEN** stdout SHALL list `<dir>/app` with its porcelain
- **AND** stderr SHALL name `<dir>/broken`
- **AND** the command SHALL exit 1

### Requirement: Dirt usage, help, and version
A scan path that does not exist, a scan path that is not a directory, or more than one argument SHALL exit 2, SHALL print nothing on stdout, and SHALL print the usage `c3po dirt [<path>]` on stderr. `c3po dirt help`, `c3po dirt -h`, and `c3po dirt --help` SHALL print the help and exit 0. `c3po dirt version`, `c3po dirt --version`, and `c3po dirt -v` SHALL print `c3po ` and the c3po version and exit 0.

#### Scenario: Usage errors exit 2
- **WHEN** the agent runs `c3po dirt /no/such/dir`
- **THEN** the command SHALL exit 2
- **AND** stderr SHALL name `usage: c3po dirt [<path>]`
- **AND** `c3po dirt <file>` and `c3po dirt a b` SHALL exit 2 the same way

#### Scenario: Help and version
- **WHEN** the agent runs `c3po dirt help`
- **THEN** the command SHALL exit 0
- **AND** stdout SHALL name `c3po dirt <path>`
- **AND** `c3po dirt version` SHALL print `c3po ` and the c3po version and exit 0

