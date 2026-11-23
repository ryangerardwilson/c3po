## Why

An agent that finishes work in several checkouts has no single command that shows which git repositories under the home directory still have uncommitted or untracked changes. It has to guess the repositories and run `git status` in each one.

## What Changes

- `c3po dirt` scans the home directory and `c3po dirt <path>` scans one directory. A relative path is resolved from the working directory.
- Stdout lists only dirty repositories, sorted by absolute path and separated by one blank line. Each is its absolute path, then `git status --porcelain=v1 -b`. A repository whose status is only the `##` line is clean and is omitted.
- A directory that holds `.git` is a repository. The walk does not enter `.git`, does enter work trees so nested repositories are found, does not follow symlinks, skips unreadable directories, and skips `node_modules`, `.venv`, `venv`, `.cache`, `__pycache__`, `.npm`, `.cargo`, and `site-packages` below the scan root.
- A scan path inside a repository reports that repository with only the porcelain paths under the scan path.
- A bare repository or a failed `git status` is named on stderr and exits 1 after the judged repositories are printed. A usage error exits 2.
- `dirt` is listed in `c3po` and `c3po help` between `deport` and `gates`, and is the usage feature `dirt`. There is no dirt screen.

## Capabilities

### New Capabilities

- `dirt`: list every dirty git repository under the home directory or one path.

### Modified Capabilities

- `scout`: `dirt` is an app in the command list, the help list, and the usage catalog.

## Impact

- Specs: `openspec/specs/dirt/spec.md`, `openspec/specs/scout/spec.md`
- Code: `dirt/`, `internal/scout`, `internal/usage`
- Docs: `README.md`, `AGENTS.md`, `openspec/config.yaml`, and the c3po app list in `/home/ryan/AGENTS.md`
- Migration: none. The command only reads. The source version stays 0.5.0.
