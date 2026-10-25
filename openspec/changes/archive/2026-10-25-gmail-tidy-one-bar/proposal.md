## Why

`c3po gmail tidy` leaves one finished progress bar per preset. Three accounts leave three bars on screen.

## What Changes

- Tidy draws one block bar for the whole run and updates that line in place on a terminal.
- The finished line stays and ends with one status per preset: preset, email, and how many messages were trashed.
- A pipe or a log gets that one finished line, not a line per preset or per group.
- The completion line stays `gmail tidy complete: presets=%d trashed=%d`.

## Capabilities

### New Capabilities

### Modified Capabilities

- `gmail`: Tidy shows one progress bar for the run.

## Impact

The progress line for `c3po gmail tidy`. What gets trashed stays the configured spam list. `c3po gmail spam clean` keeps a bar per account.
