## ADDED Requirements

### Requirement: The mess report
`c3po mess` SHALL report the linked worktrees of the git repository that contains the current directory and SHALL delete nothing. It SHALL exit 0. It SHALL NOT scan any other directory. The report SHALL be plain text: the line `mess <root>` where root is the main worktree, then the counts `clean`, `keep`, `other`, and `blocked`, then the groups `clean`, `keep`, `other`, and `blocked` in that order. An empty group SHALL be omitted, but every count SHALL be printed. The main worktree SHALL never appear in a group.

- A clean entry SHALL be a marked linked worktree that is eligible. Its line SHALL be the path, `pushed and clean`, and `c3po mess tidy`.
- A keep entry SHALL be a marked linked worktree that is not eligible. Its line SHALL be the path and the first failed condition: `locked`, `submodule`, `dirty`, `ignored file <name>`, `detached`, `remote missing`, `remote is a local path`, `not contained`, or `in use`.
- An other entry SHALL be an unmarked linked worktree. Its line SHALL be the path and `unmarked linked worktree`. It SHALL NOT be judged or removed.
- The blocked group SHALL always contain `git worktree add` with `use c3po mess tidy add`. When the clean count is not zero, it SHALL also contain `c3po mess tidy add` with `run c3po mess tidy first` and the clean paths. A keep entry SHALL NOT add a blocked action. The report SHALL NOT tell the agent to reset or stash.

Outside a git repository `c3po mess` SHALL exit 1 and say so on stderr.

#### Scenario: A repository with no linked worktrees
- **GIVEN** a git repository with only its main worktree
- **WHEN** the agent runs `c3po mess`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL show `clean  0`, `keep   0`, `other  0`, and `blocked  1`
- **AND** the blocked group SHALL contain `git worktree add` with `use c3po mess tidy add`

#### Scenario: An unmarked worktree is other
- **GIVEN** a linked worktree created with `git worktree add`
- **WHEN** the agent runs `c3po mess`
- **THEN** the worktree SHALL be listed under other as `unmarked linked worktree`

#### Scenario: A clean entry blocks tidy add
- **GIVEN** one clean marked worktree
- **WHEN** the agent runs `c3po mess`
- **THEN** the blocked group SHALL contain `c3po mess tidy add` with `run c3po mess tidy first` and that path

### Requirement: Eligibility
A marked linked worktree SHALL be eligible only when every condition holds. The first failure SHALL keep it.

- It SHALL be a marked linked worktree, not the main worktree, not locked, and without submodules.
- `git status --porcelain` SHALL be empty.
- Every ignored path's first component, relative to the worktree, SHALL be one of `node_modules`, `.next`, `dist`, `build`, `.turbo`, `coverage`, `.venv`, `__pycache__`, `.ruff_cache`, or `.pytest_cache`. Any other ignored path, including `.env`, SHALL keep it.
- It SHALL NOT be detached. The remote and branch SHALL be the branch's upstream when it has one, otherwise `origin` and the current branch. `git ls-remote` SHALL succeed, and the sha it reports SHALL contain HEAD. Any error, a missing branch, a missing remote, or a remote URL that is a local path SHALL keep it. A remote-tracking ref alone SHALL NOT count as evidence.
- No live process SHALL have its current directory inside the worktree. When `/proc` cannot answer, the worktree SHALL be kept.

#### Scenario: A dirty worktree is kept
- **GIVEN** a marked worktree with an untracked file
- **WHEN** the agent runs `c3po mess`
- **THEN** it SHALL be listed under keep with `dirty`

#### Scenario: A local-path remote is kept
- **GIVEN** a marked worktree whose remote URL is a local path
- **WHEN** the agent runs `c3po mess`
- **THEN** it SHALL be listed under keep with `remote is a local path`

#### Scenario: An ignored .env is kept
- **GIVEN** a marked worktree whose only ignored file is `.env`
- **WHEN** the agent runs `c3po mess`
- **THEN** it SHALL be listed under keep with `ignored file .env`

#### Scenario: Junk-only ignored paths are clean
- **GIVEN** a marked worktree whose only ignored path is `node_modules` and whose remote branch contains HEAD
- **WHEN** the agent runs `c3po mess`
- **THEN** it SHALL be listed under clean

### Requirement: Tidy removes clean worktrees
`c3po mess tidy` SHALL remove only the worktrees the report calls clean, each with `git worktree remove <path>` and without `--force`. It SHALL NOT drop a stash, delete a branch, or prune. It SHALL print `removed <path>` for each removed worktree, then the report of what remains. When a removal was attempted and the path still exists, the command SHALL exit 1, SHALL NOT print that path as removed, and SHALL NOT list it under clean. `c3po mess tidy --session <id>` SHALL remove only clean worktrees whose marker session is that id. The main worktree, keep entries, and other entries SHALL NOT be removed.

#### Scenario: Tidy removes a clean worktree
- **GIVEN** a clean marked worktree and an unmarked worktree
- **WHEN** the agent runs `c3po mess tidy`
- **THEN** the clean worktree SHALL be removed and printed as `removed <path>`
- **AND** the unmarked worktree and the main worktree SHALL remain

#### Scenario: A failed removal is not reported as removed
- **GIVEN** a clean marked worktree whose removal leaves the path in place
- **WHEN** the agent runs `c3po mess tidy`
- **THEN** the command SHALL exit 1
- **AND** the text SHALL NOT say `removed <path>`
- **AND** the path SHALL NOT be listed under clean

### Requirement: Tidy add creates a marked worktree
`c3po mess tidy add <path> [-b <branch>]` SHALL create one linked worktree with `git worktree add` and SHALL write the file `c3po-tidy` in that worktree's git admin directory. The marker SHALL record the session id from `--session`, else `C3PO_SESSION_ID`, else empty, the UTC time, and the path. When the report already has a clean entry, the command SHALL create nothing, SHALL exit 1, and SHALL print the blocked line `c3po mess tidy add`.

#### Scenario: Tidy add writes the marker
- **WHEN** the agent runs `c3po mess tidy add <path> -b feat --session s1`
- **THEN** the worktree SHALL exist on branch `feat`
- **AND** `.git/worktrees/<name>/c3po-tidy` SHALL record `s1`, the time, and the path

#### Scenario: Tidy add refuses while a clean entry exists
- **GIVEN** a clean marked worktree
- **WHEN** the agent runs `c3po mess tidy add <path>`
- **THEN** the command SHALL exit 1
- **AND** no worktree SHALL be created at that path
- **AND** the text SHALL say `run c3po mess tidy first`

### Requirement: The SessionEnd hook
`c3po mess tidy --hook` SHALL read one SessionEnd JSON object on stdin with `session_id` and `cwd` and SHALL run the session tidy from that cwd. Empty or unreadable stdin SHALL exit 0 and print nothing. A tidy failure on the hook path SHALL still exit 0.

#### Scenario: The hook reaps only its session
- **GIVEN** clean marked worktrees for sessions `s1` and `s2`
- **WHEN** stdin is `{"session_id":"s1","cwd":"<repo>"}`
- **THEN** only the `s1` worktree SHALL be removed
- **AND** the command SHALL exit 0

#### Scenario: Bad stdin is ignored
- **WHEN** stdin is empty or not JSON
- **THEN** the command SHALL exit 0 and print nothing

### Requirement: Mess help
`c3po mess help` SHALL describe the report, tidy, tidy add, and the hook. `c3po mess version` SHALL print the c3po version. There SHALL be no mess screen.

#### Scenario: Help names the command
- **WHEN** the agent runs `c3po mess help`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL name `c3po mess tidy add`
