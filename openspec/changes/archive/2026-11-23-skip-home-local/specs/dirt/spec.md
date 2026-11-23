## MODIFIED Requirements

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

