# agent-hook Specification

## Purpose
Decide, from a host PreToolUse event, whether an agent tool call may run, judging each write and bounded read by the enforcement of the worktree it touches, and naming the c3po ast command to use instead when it denies.
## Requirements
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

The hook SHALL allow:
- tools whose name is `rgw-ast`
- shell commands with no write target, including `c3po ast` and `rgw-ast` invocations
- all tools when no target is enforced
- a `Write` (or `write_file`, `create`) whose target path does not exist,
  because it is equivalent to `c3po ast create --expect-absent --parents`

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
and a leading `~` SHALL expand to the home directory. A command made only of
`NAME=value` assignments (optionally after `export`, `readonly`, or `local`)
with a static value SHALL define `$NAME` and `${NAME}` for the analysis; a
dynamic value SHALL make the name unknown again. A target containing any
other expansion SHALL be judged as the current directory.

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

#### Scenario: Assigned variable resolves the target
- **GIVEN** the session directory is an enforced repository
- **WHEN** stdin includes `S=/tmp/elsewhere; rm -rf $S/x`
- **AND** `/tmp/elsewhere` is outside every git worktree
- **THEN** stdout JSON SHALL have `permissionDecision` equal to `allow`

#### Scenario: Substituted variable stays unknown
- **GIVEN** enforcement is active
- **WHEN** stdin includes `D=$(pwd); rm $D/x`
- **THEN** stdout JSON SHALL have `permissionDecision` equal to `deny`

### Requirement: Bounded host reads
For a host `Read` tool call (`file_path`, optional `offset` and `limit`) whose
target is in an enforced, non-ignored worktree and is non-binary text, the hook
SHALL deny when the file has more than `max_read_lines` lines and `limit` is
absent or greater than `max_read_lines`. Otherwise it SHALL allow. The deny
reason SHALL name the line count and the exact `c3po ast read <path> --lines`
command, and SHALL say that a host `Read` with `limit` of at most
`max_read_lines` is also accepted. Shell readers (`cat`, `grep`) are not
policed.

#### Scenario: Long file without a limit is denied
- **GIVEN** enforcement is active and `max_read_lines` is 200
- **AND** `big.go` has 900 lines
- **WHEN** stdin is a `Read` of `big.go` with no `limit`
- **THEN** stdout JSON SHALL have `permissionDecision` equal to `deny`
- **AND** the reason SHALL contain `c3po ast read` and `--lines 1-200`

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
deny it and the reason SHALL give `c3po ast exec -- <command>`. A matched
`openspec` command SHALL be denied only for the mutating subcommands `new`,
`archive`, `init`, and `update`; other `openspec` subcommands SHALL be
allowed. A command already wrapped in `c3po ast exec` or `rgw-ast exec` SHALL be allowed.

#### Scenario: Bare openspec archive is denied
- **GIVEN** enforcement is active and `openspec` is in `generators.allow`
- **WHEN** stdin includes `openspec archive x`
- **THEN** stdout JSON SHALL have `permissionDecision` equal to `deny`
- **AND** the reason SHALL contain `c3po ast exec -- openspec archive x`

#### Scenario: Read-only openspec is allowed
- **GIVEN** enforcement is active
- **WHEN** stdin includes `openspec validate x --strict`
- **THEN** stdout JSON SHALL have `permissionDecision` equal to `allow`

#### Scenario: c3po ast exec is allowed
- **GIVEN** enforcement is active
- **WHEN** stdin includes `c3po ast exec -- openspec archive x`
- **THEN** stdout JSON SHALL have `permissionDecision` equal to `allow`

### Requirement: Actionable deny reasons
A deny reason for a file tool SHALL name the exact `c3po ast` command that
performs the same change with the target's path relative to its worktree:
`c3po ast patch <path> --expect-hash <sha> …` with the file's current SHA-256
for an existing file, `c3po ast create <path> --expect-absent --parents --stdin`
for a missing one. A shell deny for `git mv` or `mv` SHALL name
`c3po ast move`.

#### Scenario: Edit deny carries the hash
- **GIVEN** enforcement is active and `a.go` has SHA-256 H
- **WHEN** stdin is an `Edit` of `a.go`
- **THEN** the deny reason SHALL contain `c3po ast patch a.go --expect-hash H`

#### Scenario: git mv deny names move
- **GIVEN** enforcement is active
- **WHEN** stdin includes `git mv a.go b.go`
- **THEN** the deny reason SHALL contain `c3po ast move`

