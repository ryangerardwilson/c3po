## Decisions

1. **Measure scope** — When walking from a Git root, skip directories that
   contain their own `.git` (nested repos). Load root `.gitignore` patterns
   (and `.git/info/exclude` when present) and skip matching paths. Keep
   existing hard skips for node_modules etc.
2. **Hidden dirs** — Do not skip all `.*` directories during measure/search/map;
   only skip known bulk hidden names (`.git`, `.cache`, …). Tracked
   `.bashrc.d` becomes visible.
3. **Includes** — Add `**/*.{sh,bash}`, `**/.bashrc`, `**/.profile`,
   `**/*.{toml,yml,yaml,json}`, `**/.gitignore` to defaults.
4. **create** — `create path --expect-absent` content from `--stdin` or
   `--from-file`; fail if exists; optional `--parents`.
5. **patch input** — `--old-file`/`--new-file` and `--ops-file` JSON array of
   `{old,new}` applied sequentially against one expect-hash, one atomic write.
6. **read** — default agent-friendly headers + optional `--number`; if range
   exceeds max, clamp and print `next_start` on stderr or with `--json`.
7. **search** — parse flags before query; `search --help`; `--path` and
   `--glob`.
8. **cache** — `status --refresh` bypasses cache; JSON includes
   `cache_hit` and `cache_age_ms`.
