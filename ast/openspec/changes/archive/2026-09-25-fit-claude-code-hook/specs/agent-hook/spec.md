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

Decision JSON SHALL include `permissionDecision` of `allow` or `deny` and a
`permissionDecisionReason` string when denied. Exit code SHALL be 0 even when
denying (hosts expect successful hook delivery).

#### Scenario: Deny Write when enforced
- **GIVEN** enforcement is active for the workspace
- **WHEN** stdin is `{"tool_name":"Write","tool_input":{"path":"x.go"}}`
- **THEN** stdout JSON SHALL have `permissionDecision` equal to `deny`
- **AND** the process SHALL exit 0

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

### Requirement: Shell write analysis
The hook SHALL analyse shell command text with a quote-aware lexer. The lexer
SHALL honour single quotes, double quotes, backslash escapes, and comments.
It SHALL split commands on unquoted `;`, `&`, `&&`, `|`, `||`, `|&`,
newlines, and parentheses. It SHALL analyse command substitution, backquote,
and process substitution bodies, and `sh -c`/`bash -c`/`eval` arguments, as
nested commands.

Write targets SHALL be:
- the target of an output redirection (`>`, `>>`, `>|`, `&>`, `&>>`, `<>`,
  `>&` to a file)
- file operands of `rm`, `rmdir`, `unlink`, `truncate`, `touch`, `shred`,
  `mv`, and `tee`
- the destination of `cp`, `ln`, `install`, and `rsync`
- file operands of `chmod`, `chown`, and `chgrp`
- file operands of `sed`, `perl`, and `ruby` run with in-place editing
- `of=` of `dd`
- search roots of `find` with `-delete`, or of a writing `-exec` command
- the working tree for `git apply`, `git am`, `git checkout --`,
  `git restore`, `git rm`, `git mv`, and `git clean`, honouring `git -C`
- the current directory for `apply_patch`, `patch`, a writing `xargs`
  command, and a heredoc whose unquoted body contains command substitution

The analysis SHALL NOT treat these as writes: file-descriptor duplication
(`2>&1`, `>&2`), redirection to `/dev/null` or other `/dev/` paths, input
redirection, here-strings, and heredoc bodies. `cd` SHALL change the
directory that later commands resolve paths against. `$HOME`, `${HOME}`,
and a leading `~` SHALL expand to the home directory. A target containing
any other expansion SHALL be judged as the current directory.

The workspace SHALL be measured only when a write target exists. A command
the lexer cannot parse SHALL be judged as a write to the current directory.

#### Scenario: Descriptor duplication is not a write
- **GIVEN** enforcement is active
- **WHEN** stdin includes `rgw-ast status --json 2>&1`
- **THEN** stdout JSON SHALL have `permissionDecision` equal to `allow`

#### Scenario: Heredoc into rgw-ast is allowed
- **GIVEN** enforcement is active
- **WHEN** stdin includes `rgw-ast create f --expect-absent --stdin` fed by a quoted-delimiter heredoc
- **THEN** stdout JSON SHALL have `permissionDecision` equal to `allow`

#### Scenario: Copy out of an enforced repository is allowed
- **GIVEN** the session directory is an enforced repository
- **WHEN** stdin includes `cp a.go /tmp/elsewhere/a.go`
- **AND** `/tmp/elsewhere` is outside every git worktree
- **THEN** stdout JSON SHALL have `permissionDecision` equal to `allow`

#### Scenario: Redirect into an enforced repository is denied
- **GIVEN** enforcement is active
- **WHEN** stdin includes `echo x > a.go`
- **THEN** stdout JSON SHALL have `permissionDecision` equal to `deny`

#### Scenario: Write inside a nested shell is denied
- **GIVEN** enforcement is active
- **WHEN** stdin includes `bash -c 'rm a.go'`
- **THEN** stdout JSON SHALL have `permissionDecision` equal to `deny`

#### Scenario: Read-only pipeline is allowed
- **GIVEN** enforcement is active
- **WHEN** stdin includes `rgw-ast read a.go --lines 1-5 | head -n 2`
- **THEN** stdout JSON SHALL have `permissionDecision` equal to `allow`
