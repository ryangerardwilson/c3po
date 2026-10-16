# Working on c3po

c3po is one CLI. Scouts live under `c3po scout <name>`. The scouts are `ast` and `vibetree`.

- `ast` is the tree in `ast/`. Its command is `c3po scout ast`.
- `vibetree` is the tree in `vibetree/`. Its command is `c3po scout vibetree`.
- A new scout is a new name under `c3po scout`, with its own package. Do not add its commands at the top level.

`rgw-ast` on `PATH` is this binary. Nutshell and shadow-vim launch that name. Do not install `cx` or `complexity`.

Verification:

```sh
GOWORK=off go test ./...
./install.sh
c3po version
```

Do not add this module to the home `go.work`. Work and commit inside this Git root.
