# c3po

c3po is one CLI for scouting tools. The scouts today are `ast` and `cyclotree`.

```sh
c3po scout ast [args...]
c3po scout cyclotree [symbol] [dir]
c3po help [-f <category> [feature]]
```

Help for every tool is `c3po help`. `c3po help -f ast` lists the ast commands, and `c3po help -f ast patch` explains one of them. `c3po help -f cyclotree` explains that tool.

`ast` is the AST-aware agent boundary: trace a symbol, then create, patch, and apply files by hash. `cyclotree` shows the calls of one function. With no symbol, the tree starts at the program entrypoint and includes every call reached from it. Each row is that function, its cyclomatic complexity, and how many levels of calls can be opened under it. The screen is yellow and updates as those files change. A nested row shows its file relative to the caller. The bottom shows the call chain from the root to the row under the cursor, then that row's full path. `enter` opens that file in vim at its line. `?` toggles the shortcut list. `j` and `k` move, `l` opens the calls, and `h` moves to the caller. A call back to a function already on the path is marked `cycle` and stays closed.

The installed names are `c3po` and `rgw-ast`. `rgw-ast` is the same binary, for tools that still launch that command. `cx` and `complexity` are not installed.

## Install

Public source is https://github.com/ryangerardwilson/c3po. Release tags are `vX.Y.Z`. The current release is v0.4.0.

From a checkout:

```sh
./install.sh
```

The binary is installed to `$C3PO_INSTALL_DIR` or `~/.local/bin`, with an `rgw-ast` symlink beside it. A clean checkout reports `c3po 0.4.0+<commit>`. Uncommitted changes add `-dirty`.

Go installs the same module without the `rgw-ast` link:

```sh
GOBIN="$HOME/.local/bin" go install github.com/ryangerardwilson/c3po/cmd/c3po@v0.4.0
ln -sfn c3po "$HOME/.local/bin/rgw-ast"
```

## Develop

```sh
GOWORK=off go test ./...
```
