# c3po

c3po is one CLI for scouting tools. The scouts today are `ast` and `complexity`.

```sh
c3po scout ast [args...]
c3po scout complexity [args...]
```

`ast` is the AST-aware agent boundary: trace a symbol, then create, patch, and apply files by hash. `complexity` scores functions and schema shape, and it reports what became new or worse.

The installed names are `c3po` and `rgw-ast`. `rgw-ast` is the same binary, for tools that still launch that command. `cx` and `complexity` are not installed.

## Install

Public source is https://github.com/ryangerardwilson/c3po. Release tags are `vX.Y.Z`. The current release is v0.1.0.

From a checkout:

```sh
./install.sh
```

The binary is installed to `$C3PO_INSTALL_DIR` or `~/.local/bin`, with an `rgw-ast` symlink beside it. A clean checkout reports `c3po 0.1.0+<commit>`. Uncommitted changes add `-dirty`.

Go installs the same module without the `rgw-ast` link:

```sh
GOBIN="$HOME/.local/bin" go install github.com/ryangerardwilson/c3po/cmd/c3po@v0.1.0
ln -sfn c3po "$HOME/.local/bin/rgw-ast"
```

## Develop

```sh
GOWORK=off go test ./...
```
