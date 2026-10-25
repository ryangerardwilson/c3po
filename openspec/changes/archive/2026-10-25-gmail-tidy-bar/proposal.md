## Why

`c3po gmail tidy` prints one line for every spam-sender group. A preset with dozens of groups scrolls a list of empty counts.

## What Changes

- Each preset draws one block progress line and updates that line in place on a terminal.
- The finished line stays, with the preset, the email, the full bar, and the number trashed.
- A pipe or a log gets that one finished line, not a line per group.
- The completion line stays `gmail tidy complete: presets=%d trashed=%d`.

## Capabilities

### New Capabilities

### Modified Capabilities

- `gmail`: Tidy shows one progress line per preset.

## Impact

The cleanup progress used by `c3po gmail tidy`. What gets trashed stays the configured spam list.
