## Why

Unused c3po commands stay in the tree because there is no record of who runs them. A guess is a weak reason to delete a feature.

## What Changes

- Every c3po invocation adds one to a daily counter for a catalog feature and an actor: human, agent, or service.
- The counter stores no arguments, paths, presets, ports, or message text.
- `c3po usage` lists catalog features that have never been used, then the ones that have, with human, agent, and service counts.
- Reading help or version counts as help or version, not as the feature the text describes.
- A failure to write the counter does not change the command's exit code and prints nothing.

## Capabilities

### New Capabilities

### Modified Capabilities

- `scout`: `c3po usage` reports feature use, and every invocation records one catalog feature.

## Impact

The record is a `usage` model in the existing sqlx database `c3po`. TUI key chords are not features in this change. Operator config stays in files.
