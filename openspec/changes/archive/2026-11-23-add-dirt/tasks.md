## 1. Command

- [x] 1.1 Add the `dirt` package with `Run`, `Help`, the scan root, the walk, and the status report
- [x] 1.2 Register `dirt` in `internal/scout` apps and list it in the help usage and the app list between `deport` and `gates`
- [x] 1.3 Record the usage feature `dirt` in `internal/usage/feature.go` and `internal/usage/catalog.go`
- [x] 1.4 List `dirt` in `README.md`, `AGENTS.md`, `openspec/config.yaml`, and the c3po app list in `/home/ryan/AGENTS.md`

## 2. Check

- [x] 2.1 Test with temporary git repositories: a clean repository omitted, a dirty one printed, a nested repository, a symlink not followed, a skipped `node_modules`, a scan path inside a repository, and a failed status
- [x] 2.2 Test usage errors, help, and version
- [x] 2.3 Test that `c3po` and `c3po help` list dirt, `c3po help -f dirt` matches `c3po dirt help`, and the usage feature is `dirt`
- [x] 2.4 Validate this change with `openspec validate add-dirt --strict --no-interactive` and archive it
