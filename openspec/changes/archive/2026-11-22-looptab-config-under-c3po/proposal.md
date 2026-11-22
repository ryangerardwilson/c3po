## Why

c3po owns `~/.config/c3po/`, and the registered jobs files already sit in `~/.config/c3po/looptab/`. The installed scheduler file and the Looptab timezone file still sit in a second directory, `~/.config/looptab/`, and c3po writes the scheduler file there while ignoring `XDG_CONFIG_HOME`. One looptab directory under c3po is simpler to track and to find.

## What Changes

- The scheduler file c3po installs is `<config>/c3po/looptab/looptab`, where `<config>` is `XDG_CONFIG_HOME` when it is an absolute path and `~/.config` otherwise.
- The Looptab timezone file is `<config>/c3po/looptab/config.json`. Jobs files stay beside them, and the empty-catalog template stays `<config>/c3po/looptab/jobs`.
- Registering the scheduler file, or a path that opens the same file such as a symlink to it, exits 2.
- The installed `looptab` binary cannot be rebuilt and still opens `~/.config/looptab/looptab`. On the workstation that directory is a relative symlink to `c3po/looptab`, so the binary reads the file c3po writes. The home repository owns that symlink.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `looptab`: the scheduler file and the timezone file live under the c3po looptab config directory.

## Impact

- Code: `looptab/store.go`, `looptab/command.go`
- Tests: `looptab/command_test.go`
- Docs: `AGENTS.md`
- Not changed: the exported `JobsFile` and `Install` signatures, the catalog storage, the install header, and the `looptab` binary.
