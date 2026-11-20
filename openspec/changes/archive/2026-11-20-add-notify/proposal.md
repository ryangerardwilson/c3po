## Why

The toast command is a separate binary, `~/.local/bin/notify`, built from the home monorepo. The gmail and gdrive timers do not call it. They redefine a shell `notify` function that talks to quickshell and falls back to `notify-send`. One toast has three implementations, and the timer copies skip omarchy-shell.

## What Changes

- `c3po notify heading "<title>" body "<message>"` sends an omarchy-shell toast, with `--urgency low|normal|critical` (`-u`) and the legacy positional form. Delivery tries `omarchy-shell notify show`, then the quickshell bar. Success is silent.
- A bad message exits 2 and sends nothing. A failed delivery exits 1 and names the omarchy-shell error.
- The `ryan.notify` overlay plugin moves to `notify/plugin/`. `install.sh` installs it to `${XDG_CONFIG_HOME:-$HOME/.config}/omarchy/plugins/ryan.notify/` and removes `~/.local/bin/notify`. No `notify` link is installed.
- The gmail and gdrive timer units call `c3po notify` from the same binary. The inline shell function and the `notify-send` fallback are gone.
- Looptab's jobs template shows `c3po notify heading ...` for the `?` and `:` lines.
- There is no notify screen.

## Capabilities

### New Capabilities

- `notify`: send an omarchy-shell toast through the `ryan.notify` overlay.

### Modified Capabilities

- `scout`: `notify` is an app in the command list, the help list, and the usage catalog.

## Impact

- Specs: `openspec/specs/notify/spec.md`, `openspec/specs/scout/spec.md`
- Code: `notify/`, `internal/scout`, `internal/usage`, `gmail/timer.go`, `gdrive/internal/cli/timer.go`, `looptab/command.go`, `install.sh`
- Migration: install c3po, then rewrite the gmail and gdrive timers with `c3po gmail timer install` and `c3po gdrive timer install`. Callers of `notify heading ...` change to `c3po notify heading ...`. The source version stays 0.5.0.
