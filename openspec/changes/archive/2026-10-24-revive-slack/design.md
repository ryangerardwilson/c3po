## Context

The last Slack user CLI lived in the home monorepo as `sluser` and was deleted. Its behavior is the revived command. c3po apps are dispatched by the first argument. Config for an absorbed app is renamed into `~/.config/c3po/<app>` on startup, and that rename does not run under `go test`.

## Goals / Non-Goals

**Goals:**

- One command, `c3po slack`, with the previous message, auth, and account behavior.
- User tokens only (`xoxp-`). Bot tokens are rejected.
- The operator's existing account file and event-cache directory are adopted from `sluser`.
- The official Slack CLI directories `~/.config/slack` and `~/.local/state/slack` are not migration sources.

**Non-Goals:**

- A `slack` or `sluser` binary.
- A terminal screen.
- Posting or mark-read behavior in the Quickshell desk.
- Storing these tokens in the sqlx database.

## Decisions

1. The package is `github.com/ryangerardwilson/c3po/slack`. The revived command code stays in `slack/internal/app`. `slack.Run` maps a usage error to exit 2 and any other error to exit 1.
2. Every usage line says `c3po slack`. `c3po slack version` prints `version.String()`.
3. `appName` is `c3po/slack`. The legacy directory name is `sluser`. `xdg.Migrate` renames `~/.config/sluser` onto `~/.config/c3po/slack` and `~/.local/state/sluser` onto `~/.local/state/c3po/slack`. The in-app migrate does the same file and state rename when `c3po slack` loads config, and both are no-ops under `go test`.
4. The event cache stays a sqlite file via `modernc.org/sqlite`. It is state, not the sqlx database `c3po`.
5. Help does not print the config path, a token, or the word sluser.

## Risks / Trade-offs

- The first `c3po` process after install renames the live `sluser` config directory. The desk must already accept `~/.config/c3po/slack/config.json`.
- Migration is skipped during tests, so the rename is covered by the path the tests read and by the legacy pair list.

## Migration Plan

1. Ship `c3po slack` and the legacy pair.
2. Point the desk at the c3po file when it exists, and at the sluser file otherwise.
3. Track `.config/c3po/slack/config.json` in place of `.config/sluser/config.json`.

## Open Questions

None.
