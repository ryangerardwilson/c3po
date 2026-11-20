## 1. Command

- [x] 1.1 Add `gates/statute/code/1.md` and embed it with `go:embed`
- [x] 1.2 Add `c3po gates` with `list`, `show`, help, and version, and refuse unknown types, sections, and subcommands with exit 2
- [x] 1.3 Order sections numerically, component by component
- [x] 1.4 List `gates` in help, the usage catalog, the usage feature, the app docs, and the scout spec

## 2. Check

- [x] 2.1 Test type order, the code list line, show, essays with no gates, unknown type, unknown section, `edit`, and the section order `1`, `2`, `2.3`, `3`, `3.5.1`, `10`
- [x] 2.2 Test that `c3po` and `c3po help` list gates and `c3po help -f gates` matches `c3po gates help`
- [x] 2.3 Validate this change with `openspec validate add-gates --strict --no-interactive` and archive it
