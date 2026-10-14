## MODIFIED Requirements

### Requirement: Pre-tool-use hook
`rgw-ast hook` SHALL read a single JSON object from stdin describing a
proposed agent tool call and write a JSON decision to stdout.

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
- shell commands that write files outside `rgw-ast`, per the shell analysis
  requirement

The hook SHALL allow:
- tools whose name is `rgw-ast`
- shell commands with no write target, including `rgw-ast` invocations
- all tools when no target is enforced
- a `Write` (or `write_file`, `create`) whose target path does not exist,
  because it is equivalent to `rgw-ast create --expect-absent --parents`

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

### Requirement: Bounded host reads
For a host `Read` tool call (`file_path`, optional `offset` and `limit`) whose
target is in an enforced, non-ignored worktree and is non-binary text, the hook
SHALL deny when the file has more than `max_read_lines` lines and `limit` is
absent or greater than `max_read_lines`. Otherwise it SHALL allow. The deny
reason SHALL name the line count and the exact `rgw-ast read <path> --lines`
command, and SHALL say that a host `Read` with `limit` of at most
`max_read_lines` is also accepted. Shell readers (`cat`, `grep`) are not
policed.

#### Scenario: Long file without a limit is denied
- **GIVEN** enforcement is active and `max_read_lines` is 200
- **AND** `big.go` has 900 lines
- **WHEN** stdin is a `Read` of `big.go` with no `limit`
- **THEN** stdout JSON SHALL have `permissionDecision` equal to `deny`
- **AND** the reason SHALL contain `rgw-ast read` and `--lines 1-200`

#### Scenario: Bounded read is allowed
- **GIVEN** enforcement is active and `max_read_lines` is 200
- **WHEN** stdin is a `Read` of `big.go` with `limit` 150
- **THEN** stdout JSON SHALL have `permissionDecision` equal to `allow`

#### Scenario: Short file is allowed whole
- **GIVEN** enforcement is active and `max_read_lines` is 200
- **WHEN** stdin is a `Read` of a 40-line file with no `limit`
- **THEN** stdout JSON SHALL have `permissionDecision` equal to `allow`

### Requirement: Inline interpreter writes
The shell analysis SHALL treat inline code passed to `python`, `python3`,
`node`, `deno`, `bun`, `ruby`, `perl`, or `php` with `-c`, `-e`, `-E`, `-r`,
or `eval` as a write to the current directory when that code contains a
known file-writing call (for example `open(` with a write or append mode,
`write_text`, `write_bytes`, `os.remove`, `os.rename`, `os.replace`,
`shutil.`, `.unlink(`, `writeFile`, `appendFile`, `createWriteStream`,
`rmSync`, `renameSync`, `File.write`). Interpreters running script files are
not analysed.

#### Scenario: Python inline write is denied
- **GIVEN** enforcement is active
- **WHEN** stdin includes `python3 -c "open('a.go','w').write('x')"`
- **THEN** stdout JSON SHALL have `permissionDecision` equal to `deny`

#### Scenario: Python inline read is allowed
- **GIVEN** enforcement is active
- **WHEN** stdin includes `python3 -c "print(open('a.go').read())"`
- **THEN** stdout JSON SHALL have `permissionDecision` equal to `allow`

### Requirement: Generators run through exec
When a shell command (after wrapper prefixes) matches a `generators.allow`
pattern and the current directory is in an enforced worktree, the hook SHALL
deny it and the reason SHALL give `rgw-ast exec -- <command>`. A matched
`openspec` command SHALL be denied only for the mutating subcommands `new`,
`archive`, `init`, and `update`; other `openspec` subcommands SHALL be
allowed. A command already wrapped in `rgw-ast exec` SHALL be allowed.

#### Scenario: Bare openspec archive is denied
- **GIVEN** enforcement is active and `openspec` is in `generators.allow`
- **WHEN** stdin includes `openspec archive x`
- **THEN** stdout JSON SHALL have `permissionDecision` equal to `deny`
- **AND** the reason SHALL contain `rgw-ast exec -- openspec archive x`

#### Scenario: Read-only openspec is allowed
- **GIVEN** enforcement is active
- **WHEN** stdin includes `openspec validate x --strict`
- **THEN** stdout JSON SHALL have `permissionDecision` equal to `allow`

### Requirement: Actionable deny reasons
A deny reason for a file tool SHALL name the exact `rgw-ast` command that
performs the same change with the target's path relative to its worktree:
`rgw-ast patch <path> --expect-hash <sha> …` with the file's current SHA-256
for an existing file, `rgw-ast create <path> --expect-absent --parents --stdin`
for a missing one. A shell deny for `git mv` or `mv` SHALL name
`rgw-ast move`.

#### Scenario: Edit deny carries the hash
- **GIVEN** enforcement is active and `a.go` has SHA-256 H
- **WHEN** stdin is an `Edit` of `a.go`
- **THEN** the deny reason SHALL contain `rgw-ast patch a.go --expect-hash H`

#### Scenario: git mv deny names move
- **GIVEN** enforcement is active
- **WHEN** stdin includes `git mv a.go b.go`
- **THEN** the deny reason SHALL contain `rgw-ast move`
