# Working on c3po

c3po is one CLI. Apps are top-level commands, `c3po <name>`. The apps are `ast`, `cyclotree`, `deport`, `gcal`, `gdrive`, `gmail`, `l`, `o`, `pass`, and `tunnel`.

- `ast` is the tree in `ast/`. Its command is `c3po ast`.
- `cyclotree` is the tree in `cyclotree/`. Its command is `c3po cyclotree`.
- `deport` frees a local port. `gcal` is Google Calendar. `gdrive` syncs Google Drive. `gmail` is the Gmail desk. `l` is the compact tree listing. `o` is the file navigator. `pass` is the 1Password interface. `tunnel` publishes Cloudflare Tunnel routes.
- A new app is a new top-level name, with its own package.

`rgw-ast` on `PATH` is this binary. Nutshell and shadow-vim launch that name. Do not install `cx` or `complexity`. `vibetree`, `hide`, `iha`, and `visualize` are not commands.

Verification:

```sh
GOWORK=off go test ./...
./install.sh
c3po version
```

Do not add this module to the home `go.work`. Work and commit inside this Git root.
