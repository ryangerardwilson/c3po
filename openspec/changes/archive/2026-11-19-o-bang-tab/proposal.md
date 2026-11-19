## Why

Tab on a `:!` line inserts nothing. The operator has to type the whole command and every path.

## What Changes

- Tab on a `:!` line completes the shell word at the cursor. The listed directory is the working directory.
- The first word completes an executable on `PATH`. A word with `/` or `~`, and every later word, completes a path.
- One directory match ends with `/`. Any other one match ends with a space. Several matches fill the longest common prefix, and the prompt line lists the names when that prefix does not grow.

## Capabilities

### New Capabilities

### Modified Capabilities

- `o`: Tab completes a `:!` word and can list the matches under the title.

## Impact

The `o` command line, its help, and tests. No new dependency.
