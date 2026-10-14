# c3po

c3po is one CLI for scouting tools. The scouts today are `ast` and `complexity`.

```sh
c3po scout ast [args...]
c3po scout complexity [args...]
```

`ast` is the AST-aware agent boundary: trace a symbol, then create, patch, and apply files by hash. `complexity` scores functions and schema shape, and it reports what became new or worse.

`rgw-ast`, `cx`, and `complexity` are names for this same binary. Invoking one of those names selects that scout, so `rgw-ast patch` and `cx note` keep working.

## Install

From this checkout:

```sh
./install.sh
```

The binary is installed to `$C3PO_INSTALL_DIR` or `~/.local/bin`. The installer also links `rgw-ast`, `cx`, and `complexity` to it. Install from the changed checkout. Publishing a GitHub release is a separate task.

## Develop

```sh
GOWORK=off go test ./...
```
