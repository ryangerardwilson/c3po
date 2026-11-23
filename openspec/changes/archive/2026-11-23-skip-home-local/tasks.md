## 1. Walk

- [x] 1.1 Skip `~/.local` unless it is the scan root
- [x] 1.2 Say so in `dirt` help, the README, and `openspec/config.yaml`

## 2. Check

- [x] 2.1 Test that a home scan omits a repository under `~/.local`, that `c3po dirt ~/.local` lists it, and that another directory named `.local` is still entered
- [x] 2.2 Validate this change with `openspec validate skip-home-local --strict --no-interactive` and archive it
