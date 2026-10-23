## Context

c3po apps wrote durable records to `~/.local/share/c3po` and `~/.config/c3po/tunnel/routes.json`. sqlx backs up `~/.sqlx/data/<name>.sqlite` when `<name>` matches its database name rule. Config, credentials, and signatures were just moved into tracked `~/.config/c3po/<app>` files and must stay there.

## Goals / Non-Goals

**Goals:**

- One database, `c3po`, for OAuth tokens, gdrive sync state, and tunnel routes.
- Import those files once, leave them on disk, and ignore them after import.
- Keep tests that use a temporary home on files.

**Non-Goals:**

- Moving operator config, credentials, signatures, the pass token, or caches into sqlx.
- Storing generated `credentials.json` or `cloudflared.yml`.
- A separate database per c3po app.
- Changing Genesys or the other app databases.

## Decisions

Production paths under the login home `~/.local/share/c3po` and the tunnel routes path use sqlx. `SQLX_ROOT` selects sqlx for a path containing `/c3po/` and skips import of the real files. `C3PO_SQLX=off` forces files. A missing record is `os.ErrNotExist`, so a new token is found without the legacy file.

## Risks / Trade-offs

- The running tunnel, gmail, and gdrive units keep the old binary until they are restarted.
- Rollback to the old binary reads the leftover files, which are not updated after the import.
