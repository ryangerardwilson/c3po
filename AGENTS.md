# Working on c3po

c3po is one CLI. Apps are top-level commands, `c3po <name>`. The apps are `ast`, `deport`, `gcal`, `gdrive`, `gmail`, `looptab`, `pass`, `slack`, `tunnel`, and `tui`. Terminal screens are only `c3po tui cyclotree`, `c3po tui l`, `c3po tui o`, `c3po tui tunnel`, `c3po tui gdrive`, `c3po tui pass`, `c3po tui deport`, `c3po tui xl`, and `c3po tui looptab`.

- `ast` is the tree in `ast/`. Its command is `c3po ast`.
- A terminal screen is only `c3po tui <name>`. The screens are `cyclotree`, `l`, `o`, `tunnel`, `gdrive`, `pass`, `deport`, `xl`, and `looptab`. `c3po looptab` registers jobs files, edits one in vim, and installs them onto the Looptab scheduler. `c3po tui looptab` lists the jobs and Enter runs the selected job once through `looptab now`. `xl` is the spreadsheet. It is not a separate command.
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
