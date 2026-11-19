## MODIFIED Requirements

### Requirement: Pre-tool-use hook
`c3po ast hook` SHALL read a single JSON object from stdin describing a
proposed agent tool call and write a JSON decision to stdout. `rgw-ast hook`
SHALL do the same.

Recognized input fields SHALL include at least:
- `tool_name` (string)
- `tool_input` (object, optional)
- `command` (string, optional; shell command text)
- `cwd` (string, optional; the host session directory)

When `--root` is not given and `cwd` is present, the hook SHALL use `cwd`
as the directory relative paths and shell commands start from.

The hook SHALL decide each write by its target path:
- with `--root`, every target SHALL be judged by the enforcement of that root
- otherwise a target outside every git worktree SHALL be allowed
- a target that its git worktree ignores SHALL be allowed
- any other target SHALL be judged by the enforcement of its git worktree

The hook SHALL deny, for an enforced target:
- direct editor/mutation tool names in a built-in deny list (for example
  `Write`, `Edit`, `NotebookEdit`, `apply_patch`, `str_replace`,
  `search_replace`), judged by `file_path`, `notebook_path`, or `path`, or by
  the session directory when no path is given
- shell commands that write files outside `c3po ast` or `rgw-ast`, per the shell analysis
  requirement

Independent of enforcement, the hook SHALL deny `git worktree add` and
`c3po mess tidy add` as the linked worktree requirement says.

The hook SHALL allow:
- tools whose name is `rgw-ast`
- shell commands with no write target, including `c3po ast` and `rgw-ast` invocations
- all tools when no target is enforced
- a `Write` (or `write_file`, `create`) whose target path does not exist,
  because it is equivalent to `c3po ast create --expect-absent --parents`.
  The hook SHALL decide this before it measures, so such a create SHALL NOT
  measure the tree

Decision JSON SHALL include `permissionDecision` of `allow` or `deny` and a
`permissionDecisionReason` string when denied. Exit code SHALL be 0 even when
denying (hosts expect successful hook delivery).

#### Scenario: Deny Write when enforced
- **GIVEN** enforcement is active for the workspace
- **AND** `x.go` exists
- **WHEN** stdin is `{"tool_name":"Write","tool_input":{"path":"x.go"}}`
- **THEN** stdout JSON SHALL have `permissionDecision` equal to `deny`
- **AND** the process SHALL exit 0

#### Scenario: Write of a new file is allowed
- **GIVEN** enforcement is active for the workspace
- **AND** `notes/new.md` does not exist
- **WHEN** stdin is a `Write` whose `file_path` is `notes/new.md`
- **THEN** stdout JSON SHALL have `permissionDecision` equal to `allow`

#### Scenario: Allow rgw-ast shell when enforced
- **GIVEN** enforcement is active
- **WHEN** stdin includes a shell command whose leading executable is `rgw-ast`
- **AND** the command has no write target
- **THEN** stdout JSON SHALL have `permissionDecision` equal to `allow`

#### Scenario: Deny compound command after rgw-ast
- **GIVEN** enforcement is active
- **WHEN** stdin includes `rgw-ast status; printf x > f`
- **THEN** stdout JSON SHALL have `permissionDecision` equal to `deny`

#### Scenario: Allow quoted patch punctuation
- **GIVEN** enforcement is active
- **WHEN** an rgw-ast argument contains shell punctuation protected by single or double quotes
- **AND** the command contains no command substitution that writes
- **THEN** stdout JSON SHALL have `permissionDecision` equal to `allow`

#### Scenario: Write outside every repository is allowed
- **GIVEN** the session directory is an enforced repository
- **WHEN** a `Write` targets a path outside every git worktree
- **THEN** stdout JSON SHALL have `permissionDecision` equal to `allow`

#### Scenario: Write into an enforced repository from elsewhere is denied
- **GIVEN** the session directory is outside every git worktree
- **WHEN** an `Edit` targets a tracked file in an enforced repository
- **THEN** stdout JSON SHALL have `permissionDecision` equal to `deny`

#### Scenario: Ignored target is allowed
- **GIVEN** a repository is enforced
- **WHEN** a `Write` targets a path the repository ignores
- **THEN** stdout JSON SHALL have `permissionDecision` equal to `allow`

## ADDED Requirements

### Requirement: Linked worktrees go through c3po mess
The hook SHALL deny a shell command that runs `git worktree add`, including
`git -C <dir> worktree add` and the same command inside `sh -c` or `bash -c`,
when the effective directory (after `cd` and `git -C`) is inside a git
repository. This SHALL NOT depend on enforcement, and SHALL apply even though
the new worktree path is outside every current worktree. The reason SHALL say
`use c3po mess tidy add`. When that repository already has a clean marked
worktree, the reason SHALL also say to run `c3po mess tidy` first.

The hook SHALL deny `c3po mess tidy add` only when the repository of the
effective directory already has a clean marked worktree, and the reason SHALL
name `c3po mess tidy`. `rgw-ast` is not involved in either match.

The hook SHALL allow `git worktree list`, `git worktree remove`, `c3po mess`,
and `c3po mess tidy` without `add`. Outside every git repository `git
worktree add` SHALL be allowed.

#### Scenario: git worktree add is denied without enforcement
- **GIVEN** a git repository that is not enforced
- **WHEN** stdin includes `git worktree add ../x -b x`
- **THEN** stdout JSON SHALL have `permissionDecision` equal to `deny`
- **AND** the reason SHALL contain `use c3po mess tidy add`

#### Scenario: A new path outside every worktree is still denied
- **GIVEN** a git repository, enforced or not
- **WHEN** stdin includes `git worktree add /tmp/elsewhere`
- **THEN** stdout JSON SHALL have `permissionDecision` equal to `deny`
- **AND** the reason SHALL contain `use c3po mess tidy add`

#### Scenario: git -C and bash -c are denied
- **WHEN** stdin includes `git -C <repo> worktree add ../x` or `bash -c 'git worktree add ../x'`
- **THEN** stdout JSON SHALL have `permissionDecision` equal to `deny`

#### Scenario: Read and remove stay allowed
- **WHEN** stdin includes `git worktree list`, `git worktree remove ../x`, `c3po mess`, or `c3po mess tidy`
- **THEN** stdout JSON SHALL have `permissionDecision` equal to `allow`

#### Scenario: tidy add is allowed when nothing is clean
- **GIVEN** a git repository with no clean marked worktree
- **WHEN** stdin includes `c3po mess tidy add ../x`
- **THEN** stdout JSON SHALL have `permissionDecision` equal to `allow`

#### Scenario: tidy add is denied while a clean worktree exists
- **GIVEN** a git repository with a clean marked worktree
- **WHEN** stdin includes `c3po mess tidy add ../x`
- **THEN** stdout JSON SHALL have `permissionDecision` equal to `deny`
- **AND** the reason SHALL name `c3po mess tidy`
