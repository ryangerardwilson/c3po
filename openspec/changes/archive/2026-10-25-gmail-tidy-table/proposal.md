## Why

The tidy progress line names the account while it runs, and the finished summary packs every account onto that same line.

## What Changes

- The bar shows only the blocks and how many sender groups are done.
- When the run finishes, tidy prints a table with one row per preset: preset, email, and trashed count.
- The completion line stays `gmail tidy complete: presets=%d trashed=%d`.

## Capabilities

### New Capabilities

### Modified Capabilities

- `gmail`: Tidy prints a table under one progress bar.

## Impact

The progress and summary for `c3po gmail tidy`. What gets trashed stays the configured spam list.
