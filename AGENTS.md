# Working on c3po

c3po is one CLI. Apps are top-level commands, `c3po <name>`. The apps are `ast`, `deport`, `gcal`, `gdrive`, `gmail`, `looptab`, `pass`, `slack`, `tunnel`, and `tui`. Terminal screens are only `c3po tui cyclotree`, `c3po tui l`, `c3po tui o`, `c3po tui tunnel`, `c3po tui gdrive`, `c3po tui pass`, `c3po tui deport`, `c3po tui xl`, `c3po tui looptab`.

- `ast` is the tree in `ast/`. Its command is `c3po ast`.
- A terminal screen is only `c3po tui <name>`. The screens are `cyclotree`, `l`, `o`, `tunnel`, `gdrive`, `pass`, `deport`, `xl`, and `looptab`. `c3po looptab` registers jobs files, edits one in vim, and installs them onto the Looptab scheduler. `c3po tui looptab` lists the jobs and Enter runs the selected job once through `looptab now`. `xl` is the spreadsheet. It is not a separate command. The trainer is the crusade mode `crusade -m nlp`. Courses stay in the sqlx database `nlp`.
- `deport` frees a safe external port, 8000 through 8999. The screen is `c3po tui deport`. `gcal` is Google Calendar. `gdrive` syncs Google Drive. `gmail` is the Gmail desk. `pass` is the 1Password interface. The item screen is `c3po tui pass`. `tunnel` publishes Cloudflare Tunnel routes. The route screen is `c3po tui tunnel`.
- A new app is a new top-level name, with its own package. A new screen is a name under `c3po tui`.

`rgw-ast` on `PATH` is this binary. Nutshell and shadow-vim launch that name. Do not install `cx` or `complexity`. `vibetree`, `hide`, `iha`, and `visualize` are not commands.

Verification:

```sh
GOWORK=off go test ./...
./install.sh
c3po version
```

Do not add this module to the home `go.work`. Work and commit inside this Git root.

<!-- rgw-ast:begin -->
# c3po ast boundary

Run `c3po ast status --json` before the first edit in a repository. When it reports
`"enforced": true`, the `c3po ast hook` PreToolUse hook enforces the rules below. Every
change is checked against the file's hash, so concurrent agents and stale context
cannot silently overwrite a file.

- **Trace** code before editing it: `c3po ast callers|callees|impact <symbol>`, then
  `map` and `show`. `search` is for a literal you already know. `c3po ast help callers`
  explains confidence levels and partial results.
- **Read** with `c3po ast read <file> --lines A-B`, or a host Read with `limit` of at
  most `max_read_lines` (200 by default).
- **Edit** with `c3po ast patch <file> --expect-hash <sha> --old <text> --new <text>`.
  `c3po ast hash <file>` prints the hash; each patch prints the next one. For several
  files at once: `c3po ast apply --from-file <manifest.json|git diff>`.
  A Go create, patch, symbol patch, or apply that adds a function at `max_cc` or above,
  or raises one from below to `max_cc` or above, is refused and not written; functions
  already at or above `max_cc` stay editable (default 30). `c3po ast gap` lists them.
- **Create** with a host Write to a new path, or `c3po ast create <file> --expect-absent --stdin`.
- **Move** with `c3po ast move <src> <dst> --expect-hash <sha>`; **delete** with
  `c3po ast delete <file> --expect-hash <sha>`.
- **Generators** listed in `generators.allow` (for example `openspec new|archive`) run as
  `c3po ast exec -- <command>`.
- A denied tool call's reason names the exact command to run instead.

Unenforced repositories use normal tools. Policy is global
(`~/.config/c3po/ast/config.toml`). A nested AGENTS.md does not need to repeat this block.
<!-- rgw-ast:end -->
