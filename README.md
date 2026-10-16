# c3po

c3po is one CLI for scouting tools. The scouts today are `ast` and `vibetree`.

```sh
c3po scout ast [args...]
c3po scout vibetree <dir> [-p N] [-m loc|cyclo] [--json]
```

`ast` is the AST-aware agent boundary: trace a symbol, then create, patch, and apply files by hash. `vibetree` prints a directory as a tree of files and directories. The default cut is p95 and the default mode is `loc`, which sizes each module by lines. `-m cyclo` sizes it by the sum of function cyclomatic complexity in Go, Python, Rust, TypeScript, and JavaScript. A directory is the sum of its files. The report prints that chosen percentile and the modules at or above it. When the percentile is the largest module, it gives the rank and asks for a lower `-p`. `--json` prints the same cutoff as one JSON object. Brighter yellow means a larger size.

The installed names are `c3po` and `rgw-ast`. `rgw-ast` is the same binary, for tools that still launch that command. `cx` and `complexity` are not installed.

## Install

Public source is https://github.com/ryangerardwilson/c3po. Release tags are `vX.Y.Z`. The current release is v0.2.0.

From a checkout:

```sh
./install.sh
```

The binary is installed to `$C3PO_INSTALL_DIR` or `~/.local/bin`, with an `rgw-ast` symlink beside it. A clean checkout reports `c3po 0.2.0+<commit>`. Uncommitted changes add `-dirty`.

Go installs the same module without the `rgw-ast` link:

```sh
GOBIN="$HOME/.local/bin" go install github.com/ryangerardwilson/c3po/cmd/c3po@v0.2.0
ln -sfn c3po "$HOME/.local/bin/rgw-ast"
```

## Develop

```sh
GOWORK=off go test ./...
```
