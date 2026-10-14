## MODIFIED Requirements

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
