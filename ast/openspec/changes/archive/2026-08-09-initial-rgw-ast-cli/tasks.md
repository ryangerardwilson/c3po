## 1. OpenSpec and project skeleton

- [x] 1.1 Write proposal, design, and capability deltas for cli-surface,
      global-config, measure-enforcement, code-intelligence, file-boundary,
      distribution
- [x] 1.2 Validate with
      `npm exec -- openspec validate initial-rgw-ast-cli --strict --no-interactive`
- [x] 1.3 Scaffold `go.mod`, `cmd/rgw-ast`, internal packages, `.gitignore`,
      README, `install.sh`

## 2. Config and enforcement

- [x] 2.1 Implement global config load/ensure with defaults (threshold 5000,
      mode auto, include/exclude)
- [x] 2.2 Implement root discovery and LOC measure
- [x] 2.3 Implement `enforced` decision and `status`/`measure` commands
      (text + `--json`)

## 3. Intelligence and file boundary

- [x] 3.1 Implement `map`, `show`, `search` with caps from config
- [x] 3.2 Implement `hash`, bounded `read`, hash-guarded exact `patch`
- [x] 3.3 Enforce whole-file read rejection and hash requirement when enforced

## 4. CLI surface, distribution, verification

- [x] 4.1 Wire `help`, `version`, exit codes, `config` (print path / ensure)
- [x] 4.2 Add unit tests; `go test ./...` passes
- [x] 4.3 Local verify: `go run ./cmd/rgw-ast help|version|status`
- [x] 4.4 Archive change after strict validation and implementation evidence
