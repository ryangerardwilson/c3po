# c3po

c3po is one CLI. The apps are `ast`, `deport`, `gcal`, `gdrive`, `gmail`, `pass`, `tunnel`, and `tui`. Terminal screens are only `c3po tui cyclotree`, `c3po tui l`, `c3po tui o`, `c3po tui tunnel`, and `c3po tui gdrive`.

```sh
c3po ast [args...]
c3po deport <port>
c3po gcal [args...]
c3po gdrive [args...]
c3po gmail [args...]
c3po pass <vault> <item> [args...]
c3po tunnel add|remove|list|start|stop|status|logs|run
c3po tui cyclotree|l|o|tunnel|gdrive [args...]
c3po help [-f <category> [feature]]
```

Help for every tool is `c3po help`. `c3po help -f ast` lists the ast commands, and `c3po help -f ast patch` explains one of them. `c3po help -f tui <screen>` explains one screen. `c3po help -f <category>` explains that app.

`ast` is the AST-aware agent boundary: trace a symbol, then create, patch, and apply files by hash. `cyclotree` shows the calls of one function. With no symbol, the tree starts at the program entrypoint and includes every call reached from it. Each row is that function, its cyclomatic complexity, and how many levels of calls can be opened under it. The screen is yellow and updates as those files change. A nested row shows its file relative to the caller. The bottom shows the call chain from the root to the row under the cursor, then that row's full path. `enter` opens that file in vim at its line. `?` toggles the shortcut list. `j` and `k` move, `l` opens the calls, and `h` moves to the caller. A call back to a function already on the path is marked `cycle` and stays closed.

The installed names are `c3po` and `rgw-ast`. `rgw-ast` is the same binary, for tools that still launch that command. `cx` and `complexity` are not installed.

## Install

Public source is https://github.com/ryangerardwilson/c3po. Release tags are `vX.Y.Z`. The source version is 0.5.0. The published tag stays v0.4.0 until this version is tagged.

From a checkout:

```sh
./install.sh
```

The binary is installed to `$C3PO_INSTALL_DIR` or `~/.local/bin`, with an `rgw-ast` symlink beside it. A clean checkout reports `c3po 0.5.0+<commit>`. Uncommitted changes add `-dirty`.

Go installs the same module without the `rgw-ast` link:

```sh
GOBIN="$HOME/.local/bin" go install github.com/ryangerardwilson/c3po/cmd/c3po@v0.4.0
ln -sfn c3po "$HOME/.local/bin/rgw-ast"
```

## Develop

```sh
GOWORK=off go test ./...
```
