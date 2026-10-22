## Why

`c3po tui o open Apps/xl` makes the operator type the word open before the path. The path is the argument.

## What Changes

- **BREAKING** `c3po tui o <path> [<path> ...]` starts at a directory or opens files. The word `open` is not a command.
- `c3po tui o` with no path still starts the navigator in the working directory.
- `help`, `version`, `reveal`, `pick`, `save`, and `config` stay named commands.
- A word that is not one of those commands is a path. A missing path exits 1 and says `Path not found`.

## Capabilities

### New Capabilities

- `o`: Give a path directly to the file navigator.

### Modified Capabilities

## Impact

`c3po tui o` dispatch and its help. `reveal`, `pick`, `save`, and `config` are unchanged. Typing `c3po tui o open Apps/xl` looks up a path named `open`.
