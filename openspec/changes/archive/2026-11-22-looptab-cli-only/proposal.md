## Why

c3po is the agent CLI and r2d2 is the terminal program above it. Today `c3po looptab` with no arguments, and `c3po looptab edit`, open a jobs file in vim, which is a terminal screen inside the CLI, and the `r2d2 looptab` screen code lives in c3po. The screen should live in r2d2 and call c3po for the operation, so there is one copy of each looptab operation and c3po never opens a screen.

## What Changes

- **BREAKING** `c3po looptab` with no arguments prints the help and exits 0. It does not open vim or create a file.
- **BREAKING** `edit` is not a command. `c3po looptab edit` exits 2 with the help.
- The help says the jobs file is edited with `r2d2 looptab` and does not name vim.
- `looptab.JobsFile` and `looptab.Install` are exported. `JobsFile` returns the first registered file, and with none creates `~/.config/c3po/looptab/jobs` from the template, registers it as `jobs`, and installs. `Install` installs every registered file. Register, remove, and `Install` write the scheduler file through the one install function.
- The looptab screen, its job list, `looptab now` parsing, and its terminal code leave this repository for r2d2. The `screen` package no longer registers `looptab`, and exports `TakeResume`, `OpenSession`, and `Session` so a screen r2d2 owns keeps its c3po session.
- The catalog stays in the sqlx database `c3po`. `~/.config/looptab/looptab` stays the installed scheduler file, because the `looptab` binary reads that path.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `looptab`: the CLI no longer opens vim, the screen edits with `e`, and c3po exports the jobs file and install.
- `scout`: `looptab` is no longer a screen in this module.

## Impact

- Code: `looptab/command.go`, `looptab/store.go`, `screen/screen.go`, `internal/scout/help.go`; removed `looptab/run.go`, `looptab/tui.go`, `looptab/terminal.go`, `looptab/term_linux.go`, `looptab/term_other.go`, `looptab/jobs.go`
- Tests: `looptab/command_test.go`, `screen/screen_test.go`, `internal/scout/scout_test.go`; removed `looptab/tui_test.go` and `looptab/jobs_test.go`, which move to r2d2
- r2d2 requires this commit and owns `r2d2 looptab`.
- Not changed: the other screens in the `screen` package, the catalog storage, the template path, and the scheduler file path.
